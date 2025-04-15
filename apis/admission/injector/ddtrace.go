package injector

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	ddtraceInitContainerName = "datakit-lib-init"

	ddtraceEnabledAnnotationKey       = "admission.datakit/ddtrace.enabled"
	ddtraceVersionAnnotationKeyFormat = "admission.datakit/%s-lib.version"

	ddtraceVolumeName = "datakit-auto-instrument"
	ddtraceMountPath  = "/datadog-lib"

	ddtraceDDTagsKey = "DD_TAGS"
)

var supportedLanguagesForDDTrace = []language{java, python, nodejs}

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

	should, lang, imageVersion := r.shouldInject()
	if !should {
		return
	}

	var lib ddtraceLibrary

	switch lang {
	case java:
		lib = &ddtraceJava{}
	case nodejs, nodejsDeprecated:
		lib = &ddtraceNodejs{}
	case python:
		lib = &ddtracePython{}
	default:
		l.Warnf("Language %s is no supported, only supported %v", lang, supportedLanguagesForDDTrace)
		return
	}

	image := lib.joinReleaseImage(imageVersion)
	l.Infof("Use of ddtrace %s-lib image %s to %s for namespace %s", lang, image, r.parent, r.namespace)

	// must be nil
	_ = lib.injectInitContainer(r.pod, image)
	if err := lib.injectConfig(r.pod); err != nil {
		l.Warnf("Unable to inject DDTrace into %s, err: %s", r.parent, err)
		return
	}

	r.injectGlobalVolume()
	r.injectGlobalEnvs(ddtraceEnvObjects())
}

func (r *ddtraceResource) shouldInject() (bool, language, string) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), ddtraceEnabledAnnotationKey) {
		return false, null, ""
	}

	if manager.NewContainerManager(r.pod).ContainsInitContainer(ddtraceInitContainerName) {
		return false, null, ""
	}

	annotations := r.pod.GetAnnotations()
	for _, lang := range supportedLanguagesForDDTrace {
		versionAnnotation := strings.ToLower(fmt.Sprintf(ddtraceVersionAnnotationKeyFormat, lang))
		if imageVersion, found := annotations[versionAnnotation]; found {
			l.Debugf("ddtrace %s-lib finds annotation for %s", lang, r.parent)
			return true, lang, imageVersion
		}
	}

	var lang string
	if v := ddtraceGetLanguageFromLabels(r.pod.GetLabels()); v != "" {
		lang = v
		l.Debugf("ddtrace %s-lib finds labelSelector for %s", lang, r.parent)
	}
	if v := ddtraceGetLanguageFromNamespace(r.namespace); v != "" {
		lang = v
		l.Debugf("ddtrace %s-lib finds namespace for %s", lang, r.parent)
	}

	switch language(lang) {
	case java, python, nodejs, nodejsDeprecated:
		return true, language(lang), ""
	default:
		// nil
	}

	return false, null, ""
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
	joinReleaseImage(imageVersion string) string
	injectInitContainer(pod *corev1.Pod, image string) error
	injectConfig(pod *corev1.Pod) error
}

//
// ddtraceJava
//

type ddtraceJava struct{}

func (d *ddtraceJava) joinReleaseImage(imageVersion string) string {
	return replaceImageVersion(ddtraceJavaAgentImage(), imageVersion)
}

func (d *ddtraceJava) injectInitContainer(pod *corev1.Pod, image string) error {
	return injectDDTraceInitContainer(pod, image)
}

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

//
// ddtracePython
//

type ddtracePython struct{}

func (d *ddtracePython) joinReleaseImage(imageVersion string) string {
	return replaceImageVersion(ddtracePythonAgentImage(), imageVersion)
}

func (d *ddtracePython) injectInitContainer(pod *corev1.Pod, image string) error {
	/*
		return injectDDTraceInitContainer(pod, image)
	*/
	return nil
}

func (d *ddtracePython) injectConfig(pod *corev1.Pod) error {
	// Python config
	/*
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
	*/

	return nil
}

//
// ddtraceNodejs
//

type ddtraceNodejs struct{}

func (d *ddtraceNodejs) joinReleaseImage(imageVersion string) string {
	return replaceImageVersion(ddtraceNodejsAgentImage(), imageVersion)
}

func (d *ddtraceNodejs) injectInitContainer(pod *corev1.Pod, image string) error {
	return injectDDTraceInitContainer(pod, image)
}

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

func injectDDTraceInitContainer(pod *corev1.Pod, image string) error {
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
	manager.NewContainerManager(pod).AddInitContainer(&container)
	return nil
}

func replaceImageVersion(image, imageVersion string) string {
	if imageVersion == "" {
		return image
	}
	imageName, _, _ := ParseImage(image)
	return fmt.Sprintf("%s:%s", imageName, imageVersion)
}

func envIndex(envs []corev1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}
