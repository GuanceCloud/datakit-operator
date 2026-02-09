// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"fmt"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	ddtraceInitContainerName    = "datakit-lib-init"
	ddtraceEnabledAnnotationKey = "admission.datakit/ddtrace.enabled"
	javaLibVersionAnnotationKey = "admission.datakit/java-lib.version"

	ddtraceVolumeName = "datakit-auto-instrument"
	ddtraceMountPath  = "/datadog-lib"
	ddtraceDDTagsKey  = "DD_TAGS"
)

func InjectDDTraceToPod(namespace, parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject ddtrace-lib into nil pod")
	}

	r := newDDTraceResource(namespace, parent, pod)
	r.process()
	return nil
}

type ddtraceResource struct {
	// Kubernetes 1.19 and earlier versions do not include the namespace in the AdmissionReview.
	// Therefore, the namespace from the upper-level AdmissionReview is recorded first
	namespace string
	parent    string
	pod       *corev1.Pod
}

func newDDTraceResource(namespace, parent string, pod *corev1.Pod) *ddtraceResource {
	return &ddtraceResource{
		namespace: namespace,
		parent:    parent,
		pod:       pod,
	}
}

func (r *ddtraceResource) process() {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	should, rule := r.getMatchingRule()
	if !should || rule == nil {
		return
	}

	log.Infof("ddtrace injection started: pod=%s, namespace=%s, language=%s, rule=%s", r.parent, r.namespace, rule.Language, rule.Name)

	var lib ddtraceLibrary
	switch language(rule.Language) {
	case java:
		lib = &ddtraceJava{}
	default:
		log.Warnf("ddtrace language not supported: lang=%s pod=%s", rule.Language, r.parent)
		return
	}

	// 如果 CheckAnnotation 为 true，尝试从 annotation 中获取版本并替换
	image := rule.Image
	if rule.CheckAnnotation {
		if imageVersion := r.pod.GetAnnotations()[javaLibVersionAnnotationKey]; imageVersion != "" {
			image = replaceImageVersion(image, imageVersion)
		}
	}
	r.injectInitContainer(image, rule.Resources)

	if err := lib.injectConfig(r.pod); err != nil {
		log.Warnf("ddtrace inject failed: pod=%s, error=%v", r.parent, err)
		return
	}

	r.injectGlobalVolume()

	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)
	envs = envbuilder.FilterAndSetResourceFieldRefEnvVars(envs, r.pod)
	r.injectGlobalEnvs(envs)
	log.Debugf("ddtrace config injected: pod=%s, envs=%d, containers=%d", r.parent, len(envs), len(r.pod.Spec.Containers))

	log.Infof("ddtrace injection completed: pod=%s, image=%s, rule=%s", r.parent, image, rule.Name)
}

func (r *ddtraceResource) getMatchingRule() (bool, *config.InjectRule) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), ddtraceEnabledAnnotationKey) {
		log.Debugf("ddtrace annotation disabled: pod=%s", r.parent)
		return false, nil
	}

	if manager.NewContainerManager(r.pod).ContainsInitContainer(ddtraceInitContainerName) {
		log.Debugf("ddtrace init container already exists: pod=%s", r.parent)
		return false, nil
	}

	matched, rule := ddtraceMatchNamespaceOrLabelsForConfig(r.namespace, r.pod.GetLabels())
	if !matched || rule == nil {
		return false, nil
	}

	// 如果 rule.CheckAnnotation 为 true，必须存在 java-lib.version Annotation
	if rule.CheckAnnotation {
		annotations := r.pod.GetAnnotations()
		if _, exists := annotations[javaLibVersionAnnotationKey]; !exists {
			log.Debugf("ddtrace rule requires java-lib.version annotation but not found: pod=%s", r.parent)
			return false, nil
		}
	}

	log.Infof("ddtrace rule matched: pod=%s, language=%s, image=%s", r.parent, rule.Language, rule.Image)
	return true, rule
}

func (r *ddtraceResource) injectInitContainer(image string, resources config.ResourceRequirements) {
	container := corev1.Container{
		Name:            ddtraceInitContainerName,
		Image:           image,
		Command:         []string{"sh", "copy-lib.sh", ddtraceMountPath},
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      ddtraceVolumeName,
				MountPath: ddtraceMountPath,
			},
		},
	}

	setContainerResources(&container, resources.Requests.CPU, resources.Requests.Memory, resources.Limits.CPU, resources.Limits.Memory)
	manager.NewContainerManager(r.pod).AddInitContainer(&container)
}

func (r *ddtraceResource) injectGlobalVolume() {
	volume := corev1.Volume{
		Name: ddtraceVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	manager.NewVolumeManager(r.pod).AddVolume(&volume)
}

func (r *ddtraceResource) injectGlobalEnvs(envs []corev1.EnvVar) {
	m := manager.NewEnvVarManager(r.pod)
	for idx := range envs {
		// DD_TAGS need to be merged
		if envs[idx].Name == ddtraceDDTagsKey {
			r.specialDDTagsEnv(&envs[idx])
			continue
		}
		m.AddEnvVar(&envs[idx])
	}
}

func (r *ddtraceResource) specialDDTagsEnv(newEnv *corev1.EnvVar) {
	if newEnv.Value == "" {
		return
	}

	m := manager.NewEnvVarManager(r.pod)

	for cIdx, container := range r.pod.Spec.Containers {
		oldDDTagsIndex := -1

		newEnvWithContainer := &corev1.EnvVar{
			Name:  newEnv.Name,
			Value: newEnv.Value,
		}

		for envIdx, env := range container.Env {
			if env.Name != ddtraceDDTagsKey {
				continue
			}
			if env.ValueFrom == nil {
				oldDDTagsIndex = envIdx
				kvStr := appendKVPairs(env.Value, newEnvWithContainer.Value)
				newEnvWithContainer.Value = kvStr
			}
			break
		}

		if oldDDTagsIndex != -1 {
			env := DeleteSlice(r.pod.Spec.Containers[cIdx].Env, oldDDTagsIndex, oldDDTagsIndex+1)
			r.pod.Spec.Containers[cIdx].Env = env

		}

		m.AddEnvVarToContainer(container.Name, newEnvWithContainer)
	}
}

type ddtraceLibrary interface {
	injectConfig(pod *corev1.Pod) error
}

type ddtraceJava struct{}

func (d *ddtraceJava) injectConfig(pod *corev1.Pod) error {
	// Java config
	const (
		javaToolOptionsKey   = "JAVA_TOOL_OPTIONS"
		javaToolOptionsValue = " -javaagent:/datadog-lib/dd-java-agent.jar"
	)

	envValFunc := func(predefinedVal string) string {
		return predefinedVal + javaToolOptionsValue
	}
	return injectDDTraceConfig(pod, javaToolOptionsKey, envValFunc)
}

/*
type ddtracePython struct{}

func (d *ddtracePython) injectConfig(pod *corev1.Pod) error {
	const (
		pythonPathKey   = "PYTHONPATH"
		pythonPathValue = "/datadog-lib/"
	)

	envValFunc := func(predefinedVal string) string {
		if predefinedVal == "" {
			return pythonPathValue
		}
		return fmt.Sprintf("%s:%s", pythonPathValue, predefinedVal)
	}
	return injectDDTraceConfig(pod, pythonPathKey, envValFunc)
}

type ddtraceNodejs struct{}

func (d *ddtraceNodejs) injectConfig(pod *corev1.Pod) error {
	// Nodejs config
	const (
		nodejsOptionsKey   = "NODE_OPTIONS"
		nodejsOptionsValue = " --require=/datadog-lib/node_modules/dd-trace/init"
	)

	envValFunc := func(predefinedVal string) string {
		return predefinedVal + nodejsOptionsValue
	}
	return injectDDTraceConfig(pod, nodejsOptionsKey, envValFunc)
}
*/

func injectDDTraceConfig(pod *corev1.Pod, envKey string, envVal func(string) string) error {
	podSpec := pod.Spec
	for i, container := range podSpec.Containers {
		index := envIndex(container.Env, envKey)

		if index < 0 {
			podSpec.Containers[i].Env = append(podSpec.Containers[i].Env, corev1.EnvVar{
				Name:  envKey,
				Value: envVal(""),
			})
		} else {
			if podSpec.Containers[i].Env[index].ValueFrom != nil {
				return fmt.Errorf("%q is defined via ValueFrom", envKey)
			}
			podSpec.Containers[i].Env[index].Value = envVal(podSpec.Containers[i].Env[index].Value)
		}
	}

	volumeMount := corev1.VolumeMount{
		Name:      ddtraceVolumeName,
		MountPath: ddtraceMountPath,
	}
	// This is a special volumeMount, do not need to check for duplicates.
	manager.NewVolumeMountManager(pod).AddVolumeMount(&volumeMount)
	return nil
}

func envIndex(envs []corev1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}
