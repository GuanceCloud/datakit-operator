// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/podcompare"
	corev1 "k8s.io/api/core/v1"
)

const (
	ddtraceInitContainerName          = "datakit-lib-init"
	ddtraceEnabledAnnotationKey       = "admission.datakit/ddtrace.enabled"
	ddtraceVersionAnnotationKeyFormat = "admission.datakit/%s-lib.version"

	ddtraceVolumeName = "datakit-auto-instrument"
	ddtraceMountPath  = "/datadog-lib"
	ddtraceDDTagsKey  = "DD_TAGS"

	javaToolOptionsKey  = "JAVA_TOOL_OPTIONS"
	javaAgentOption     = "-javaagent:/datadog-lib/dd-java-agent.jar"
	phpDefaultINIPath   = "/usr/local/etc/php/conf.d"
	phpLoaderFileName   = "dd_library_loader.ini"
	phpLoaderFlavorGNU  = "linux-gnu"
	phpLoaderFlavorMusl = "linux-musl"
)

func InjectDDTraceToPod(namespace, parent string, pod *corev1.Pod) (bool, error) {
	_, changed, err := injectDDTraceToPod(namespace, parent, pod)
	return changed, err
}

func injectDDTraceToPod(namespace, parent string, pod *corev1.Pod) (bool, bool, error) {
	if pod == nil {
		return false, false, fmt.Errorf("cannot inject ddtrace-lib into nil pod")
	}

	before := pod.DeepCopy()
	r := newDDTraceResource(namespace, parent, pod)
	selected := r.process()
	return selected, podcompare.Changed(before, pod), nil
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

func (r *ddtraceResource) process() bool {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	should, rule, imageVersion := r.getMatchingRule()
	if !should || rule == nil {
		return false
	}

	lang := language(rule.Language)
	if manager.NewContainerManager(r.pod).ContainsInitContainer(ddtraceInitContainerName) {
		if lang == java {
			r.repairJavaAgentEnv()
		} else {
			log.Debugf("ddtrace init container already exists: pod=%s, language=%s", r.parent, rule.Language)
		}
		return true
	}

	log.Infof("ddtrace injection started: pod=%s, namespace=%s, language=%s, rule=%s", r.parent, r.namespace, rule.Language, rule.Name)

	var lib ddtraceLibrary
	switch lang {
	case java:
		lib = &ddtraceJava{}
	case python:
		lib = &ddtracePython{}
	case php:
		lib = &ddtracePHP{}
	case nodejs:
		lib = &ddtraceNodejs{}
	default:
		log.Warnf("ddtrace language not supported: lang=%s pod=%s", rule.Language, r.parent)
		return true
	}

	image := rule.Image
	// 如果 annotation 中有版本信息，替换 image 版本
	if imageVersion != "" {
		image = replaceImageVersion(image, imageVersion)
	}

	// Build the complete DDTrace mutation on a copy. If one component cannot be
	// injected (for example, JAVA_TOOL_OPTIONS uses valueFrom), do not leave a
	// partial init-container-only mutation behind.
	mutatedPod := r.pod.DeepCopy()
	mutatedResource := newDDTraceResource(r.namespace, r.parent, mutatedPod)
	phpLoaderFlavor := mutatedResource.getPHPLoaderFlavor(rule)
	mutatedResource.injectInitContainer(image, rule.Resources, lang, phpLoaderFlavor)

	if err := lib.injectConfig(mutatedPod); err != nil {
		log.Warnf("ddtrace inject failed: pod=%s, error=%v", r.parent, err)
		return true
	}

	mutatedResource.injectGlobalVolume()

	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)
	envs = envbuilder.FilterAndSetResourceFieldRefEnvVars(envs, mutatedPod)
	mutatedResource.injectGlobalEnvs(envs)
	log.Debugf("ddtrace config injected: pod=%s, envs=%d, containers=%d", r.parent, len(envs), len(mutatedPod.Spec.Containers))

	*r.pod = *mutatedPod

	log.Infof("ddtrace injection completed: pod=%s, image=%s, rule=%s", r.parent, image, rule.Name)
	return true
}

func (r *ddtraceResource) getMatchingRule() (matched bool, ruleConfig *config.DDTraceRule, imageVersion string) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), ddtraceEnabledAnnotationKey) {
		log.Debugf("ddtrace annotation disabled: pod=%s", r.parent)
		return false, nil, ""
	}
	if manager.NewContainerManager(r.pod).ContainsInitContainer(otelInitContainerName) {
		log.Warnf("ddtrace inject skipped: pod=%s, namespace=%s, reason=otel init container already exists", r.parent, r.namespace)
		return false, nil, ""
	}

	matched, rules := ddtraceMatchAllNamespaceOrLabelsForConfig(r.namespace, r.pod.GetLabels())
	if !matched || len(rules) == 0 {
		return false, nil, ""
	}

	annotations := r.pod.GetAnnotations()

	for _, rule := range rules {
		if !rule.CheckAnnotation {
			log.Infof("ddtrace rule matched: pod=%s, language=%s, image=%s", r.parent, rule.Language, rule.Image)
			return true, rule, ""
		}

		ruleLang := language(rule.Language)
		versionAnnotation := fmt.Sprintf(ddtraceVersionAnnotationKeyFormat, ruleLang)
		if version, exists := annotations[versionAnnotation]; exists {
			log.Debugf("ddtrace found %s-lib.version annotation: pod=%s", ruleLang, r.parent)
			return true, rule, version
		}
	}

	return false, nil, ""
}

func (r *ddtraceResource) repairJavaAgentEnv() {
	if !manager.NewVolumeManager(r.pod).IsEmptyDirVolume(ddtraceVolumeName) {
		log.Warnf("ddtrace java env repair skipped: pod=%s, reason=volume_missing_or_invalid", r.parent)
		return
	}

	initContainerHasMount := false
	for idx := range r.pod.Spec.InitContainers {
		if r.pod.Spec.InitContainers[idx].Name == ddtraceInitContainerName {
			initContainerHasMount = hasDDTraceLibraryMount(&r.pod.Spec.InitContainers[idx])
			break
		}
	}
	if !initContainerHasMount {
		log.Warnf("ddtrace java env repair skipped: pod=%s, reason=init_container_mount_missing", r.parent)
		return
	}

	mutatedPod := r.pod.DeepCopy()
	targets := 0
	for idx := range mutatedPod.Spec.Containers {
		container := &mutatedPod.Spec.Containers[idx]
		if !hasDDTraceLibraryMount(container) {
			continue
		}
		targets++
		if err := injectDDTraceEnvToContainer(container, javaToolOptionsKey, appendJavaAgentIfMissing); err != nil {
			log.Warnf("ddtrace java env repair failed: pod=%s, container=%s, error=%v", r.parent, container.Name, err)
			return
		}
	}

	if targets == 0 {
		log.Warnf("ddtrace java env repair skipped: pod=%s, reason=application_mount_missing", r.parent)
		return
	}
	if !podcompare.Changed(r.pod, mutatedPod) {
		log.Debugf("ddtrace java env already complete: pod=%s, containers=%d", r.parent, targets)
		return
	}

	*r.pod = *mutatedPod
	log.Infof("ddtrace java env repaired: pod=%s, containers=%d", r.parent, targets)
}

func hasDDTraceLibraryMount(container *corev1.Container) bool {
	for idx := range container.VolumeMounts {
		mount := container.VolumeMounts[idx]
		if mount.Name == ddtraceVolumeName && mount.MountPath == ddtraceMountPath {
			return true
		}
	}
	return false
}

func (r *ddtraceResource) injectInitContainer(image string, resources config.ResourceRequirements, lang language, phpLoaderFlavor string) {
	command := []string{"sh", "copy-lib.sh", ddtraceMountPath}
	if lang == php {
		copyCmd := fmt.Sprintf(
			"/datadog-init/copy-lib.sh %s && cp %s/%s/loader/%s %s/%s",
			ddtraceMountPath, ddtraceMountPath, phpLoaderFlavor, phpLoaderFileName, ddtraceMountPath, phpLoaderFileName,
		)
		command = []string{"sh", "-c", copyCmd}
	}

	container := corev1.Container{
		Name:            ddtraceInitContainerName,
		Image:           image,
		Command:         command,
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

func (r *ddtraceResource) getPHPLoaderFlavor(rule *config.DDTraceRule) string {
	if language(rule.Language) != php {
		return ""
	}

	if isValidPHPLoaderFlavor(rule.PHPLoaderFlavor) {
		return rule.PHPLoaderFlavor
	}
	if rule.PHPLoaderFlavor != "" {
		log.Warnf("invalid php_loader_flavor in rule: pod=%s, flavor=%s, fallback=%s", r.parent, rule.PHPLoaderFlavor, phpLoaderFlavorGNU)
	}
	return phpLoaderFlavorGNU
}

func isValidPHPLoaderFlavor(flavor string) bool {
	switch flavor {
	case phpLoaderFlavorGNU, phpLoaderFlavorMusl:
		return true
	default:
		return false
	}
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
	for idx := range envs {
		incoming := envs[idx]
		resolve := manager.KeepExistingEnvVar
		if incoming.Name == ddtraceDDTagsKey {
			if incoming.Value == "" || incoming.ValueFrom != nil {
				continue
			}
			resolve = mergeDDTagsEnvVar
		}

		for containerIdx := range r.pod.Spec.Containers {
			container := &r.pod.Spec.Containers[containerIdx]
			container.Env = manager.AddOrUpdateEnvVar(container.Env, incoming, resolve)
		}
	}
}

func mergeDDTagsEnvVar(existing, incoming corev1.EnvVar) corev1.EnvVar {
	if existing.ValueFrom != nil {
		return existing
	}
	existing.Value = appendKVPairs(existing.Value, incoming.Value)
	return existing
}

type ddtraceLibrary interface {
	injectConfig(pod *corev1.Pod) error
}

type ddtraceJava struct{}

func (d *ddtraceJava) injectConfig(pod *corev1.Pod) error {
	return injectDDTraceConfig(pod, javaToolOptionsKey, appendJavaAgentIfMissing)
}

type ddtracePython struct{}

func (d *ddtracePython) injectConfig(pod *corev1.Pod) error {
	// Python config
	const (
		pythonPathKey   = "PYTHONPATH"
		pythonPathValue = "/datadog-lib/"
	)

	envValFunc := func(predefinedVal string) string {
		if pathListContains(predefinedVal, pythonPathValue) {
			return predefinedVal
		}
		if predefinedVal == "" {
			return pythonPathValue
		}
		return fmt.Sprintf("%s:%s", pythonPathValue, predefinedVal)
	}
	return injectDDTraceConfig(pod, pythonPathKey, envValFunc)
}

type ddtracePHP struct{}

func (d *ddtracePHP) injectConfig(pod *corev1.Pod) error {
	// PHP config
	const (
		phpIniScanDirKey          = "PHP_INI_SCAN_DIR"
		phpLoaderPackagePathKey   = "DD_LOADER_PACKAGE_PATH"
		phpLoaderPackagePathValue = ddtraceMountPath
	)

	if err := injectDDTraceConfig(pod, phpLoaderPackagePathKey, func(predefinedVal string) string {
		if predefinedVal != "" {
			return predefinedVal
		}
		return phpLoaderPackagePathValue
	}); err != nil {
		return err
	}

	envValFunc := func(predefinedVal string) string {
		if pathListContains(predefinedVal, ddtraceMountPath) {
			return predefinedVal
		}
		if predefinedVal == "" {
			return fmt.Sprintf("%s:%s", phpDefaultINIPath, ddtraceMountPath)
		}
		return fmt.Sprintf("%s:%s", predefinedVal, ddtraceMountPath)
	}
	return injectDDTraceConfig(pod, phpIniScanDirKey, envValFunc)
}

type ddtraceNodejs struct{}

func (d *ddtraceNodejs) injectConfig(pod *corev1.Pod) error {
	// Nodejs config
	const (
		nodejsOptionsKey = "NODE_OPTIONS"
		nodejsInitOption = "--require=/datadog-lib/node_modules/dd-trace/init"
	)

	envValFunc := func(predefinedVal string) string {
		return appendOptionIfMissing(predefinedVal, nodejsInitOption)
	}
	return injectDDTraceConfig(pod, nodejsOptionsKey, envValFunc)
}

func injectDDTraceConfig(pod *corev1.Pod, envKey string, envVal func(string) string) error {
	for idx := range pod.Spec.Containers {
		if err := injectDDTraceEnvToContainer(&pod.Spec.Containers[idx], envKey, envVal); err != nil {
			return err
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

func injectDDTraceEnvToContainer(container *corev1.Container, envKey string, envVal func(string) string) error {
	for idx := range container.Env {
		if container.Env[idx].Name != envKey {
			continue
		}
		if container.Env[idx].ValueFrom != nil {
			return fmt.Errorf("%q is defined via ValueFrom", envKey)
		}
		break
	}

	incoming := corev1.EnvVar{Name: envKey, Value: envVal("")}
	resolve := func(existing, _ corev1.EnvVar) corev1.EnvVar {
		existing.Value = envVal(existing.Value)
		return existing
	}

	container.Env = manager.AddOrUpdateEnvVar(container.Env, incoming, resolve)
	return nil
}

func appendJavaAgentIfMissing(value string) string {
	for _, existing := range strings.Fields(value) {
		if existing == javaAgentOption || strings.HasPrefix(existing, javaAgentOption+"=") {
			return value
		}
	}
	return value + " " + javaAgentOption
}

func appendOptionIfMissing(value, option string) string {
	for _, existing := range strings.Fields(value) {
		if existing == option {
			return value
		}
	}
	return value + " " + option
}

func pathListContains(value, path string) bool {
	for _, existing := range strings.Split(value, ":") {
		if existing == path {
			return true
		}
	}
	return false
}
