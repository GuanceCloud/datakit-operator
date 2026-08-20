// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"fmt"
	"reflect"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/podcompare"
	corev1 "k8s.io/api/core/v1"
	k8sreflect "k8s.io/apimachinery/third_party/forked/golang/reflect"
)

const (
	otelInitContainerName          = "datakit-otel-lib-init"
	otelEnabledAnnotationKey       = "admission.datakit/otel.enabled"
	otelVersionAnnotationKeyFormat = "admission.datakit/otel-%s-lib.version"

	otelVolumeName          = "datakit-otel-auto-instrument"
	otelJavaMountPath       = "/otel-auto-instrumentation-java"
	otelJavaAgentSourcePath = "/javaagent.jar"
	otelJavaAgentPath       = otelJavaMountPath + "/javaagent.jar"
	otelJavaToolOptionsKey  = "JAVA_TOOL_OPTIONS"
	otelJavaAgentOption     = "-javaagent:" + otelJavaAgentPath
	ddJavaAgentFileName     = "dd-java-agent.jar"

	otelPythonMountPath        = "/otel-auto-instrumentation-python"
	otelPythonSourcePath       = "/autoinstrumentation/."
	otelPythonPathKey          = "PYTHONPATH"
	otelPythonPathPrefix       = otelPythonMountPath + "/opentelemetry/instrumentation/auto_instrumentation"
	otelPythonInitCopyArgument = "-r"

	otelNodeJSMountPath        = "/otel-auto-instrumentation-nodejs"
	otelNodeJSSourcePath       = "/autoinstrumentation/."
	otelNodeJSOptionsKey       = "NODE_OPTIONS"
	otelNodeJSRequirePath      = otelNodeJSMountPath + "/autoinstrumentation.js"
	otelNodeJSRequireOption    = "--require " + otelNodeJSRequirePath
	otelNodeJSInitCopyArgument = "-r"
)

func InjectOTelToPod(namespace, parent string, pod *corev1.Pod) (bool, error) {
	if pod == nil {
		return false, fmt.Errorf("cannot inject otel-lib into nil pod")
	}

	before := pod.DeepCopy()
	r := &otelResource{namespace: namespace, parent: parent, pod: pod}
	r.process()
	return podcompare.Changed(before, pod), nil
}

type otelResource struct {
	namespace string
	parent    string
	pod       *corev1.Pod
}

func (r *otelResource) process() {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), otelEnabledAnnotationKey) {
		log.Debugf("otel annotation disabled: pod=%s, namespace=%s", r.parent, r.namespace)
		return
	}

	matched, rules := otelMatchAllNamespaceOrLabelsForConfig(r.namespace, r.pod.GetLabels())
	if !matched || len(rules) == 0 {
		return
	}

	rule, imageVersion := selectOTelRule(rules, r.pod.GetAnnotations())
	if rule == nil {
		return
	}
	r.warnIfMultipleRulesMatched(rules, rule)
	ruleLanguage := language(rule.Language)
	library, supported := r.libraryForLanguage(ruleLanguage)
	if !supported {
		log.Warnf("otel language not supported: pod=%s, namespace=%s, rule=%s, language=%s", r.parent, r.namespace, rule.Name, rule.Language)
		return
	}

	image := rule.Image
	if imageVersion != "" {
		image = replaceImageVersion(image, imageVersion)
	}
	initContainer := r.initContainer(rule, image, library)
	if err := library.validate(rule.Image, &initContainer); err != nil {
		log.Warnf("otel inject skipped: pod=%s, namespace=%s, rule=%s, reason=%v", r.parent, r.namespace, rule.Name, err)
		return
	}

	log.Infof("otel injection started: pod=%s, namespace=%s, language=%s, rule=%s", r.parent, r.namespace, rule.Language, rule.Name)
	r.injectInitContainer(&initContainer)
	r.injectVolume()
	library.inject()

	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)
	envs = envbuilder.FilterAndSetResourceFieldRefEnvVars(envs, r.pod)
	for idx := range r.pod.Spec.Containers {
		container := &r.pod.Spec.Containers[idx]
		container.Env = manager.AddOrUpdateEnvVars(container.Env, envs, manager.KeepExistingEnvVar)
	}
	log.Infof("otel injection completed: pod=%s, namespace=%s, image=%s, rule=%s", r.parent, r.namespace, image, rule.Name)
}

type otelLibrary struct {
	mountPath string
	command   []string
	validate  func(string, *corev1.Container) error
	inject    func()
}

func (r *otelResource) libraryForLanguage(lang language) (otelLibrary, bool) {
	switch lang {
	case java:
		return otelLibrary{
			mountPath: otelJavaMountPath,
			command:   []string{"cp", otelJavaAgentSourcePath, otelJavaAgentPath},
			validate:  r.validateJavaInjection,
			inject:    r.injectJavaConfig,
		}, true
	case python:
		return otelLibrary{
			mountPath: otelPythonMountPath,
			command:   []string{"cp", otelPythonInitCopyArgument, otelPythonSourcePath, otelPythonMountPath},
			validate:  r.validatePythonInjection,
			inject:    r.injectPythonConfig,
		}, true
	case nodejs:
		return otelLibrary{
			mountPath: otelNodeJSMountPath,
			command:   []string{"cp", otelNodeJSInitCopyArgument, otelNodeJSSourcePath, otelNodeJSMountPath},
			validate:  r.validateNodeJSInjection,
			inject:    r.injectNodeJSConfig,
		}, true
	default:
		return otelLibrary{}, false
	}
}

func (r *otelResource) warnIfMultipleRulesMatched(rules []*config.OTelRule, selected *config.OTelRule) {
	matchedRules := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		matchedRules = append(matchedRules, fmt.Sprintf("%s(%s)", rule.Name, rule.Language))
	}
	if len(matchedRules) < 2 {
		return
	}
	log.Warnf(
		"multiple otel rules matched: pod=%s, namespace=%s, selected_rule=%s, selected_language=%s, matched_rules=%s",
		r.parent,
		r.namespace,
		selected.Name,
		selected.Language,
		strings.Join(matchedRules, ","),
	)
}

func (r *otelResource) validatePythonInjection(configuredImage string, desiredInitContainer *corev1.Container) error {
	if err := r.validateCommonInjection(configuredImage, desiredInitContainer, otelPythonMountPath); err != nil {
		return err
	}

	for _, container := range r.pod.Spec.Containers {
		pythonPathCount := 0
		for idx := range container.Env {
			env := &container.Env[idx]
			if env.Name != otelPythonPathKey {
				continue
			}
			pythonPathCount++
			if pythonPathCount > 1 {
				return fmt.Errorf("container %q has duplicate %s entries", container.Name, otelPythonPathKey)
			}
			if env.ValueFrom != nil {
				return fmt.Errorf("container %q has %s defined via ValueFrom", container.Name, otelPythonPathKey)
			}
			if containsPythonPathComponent(env.Value, ddtraceMountPath) {
				return fmt.Errorf("container %q already uses the Datadog Python library", container.Name)
			}
			if !isOTelPythonPathInjected(env.Value) &&
				(containsPythonPathComponent(env.Value, otelPythonPathPrefix) ||
					containsPythonPathComponent(env.Value, otelPythonMountPath)) {
				return fmt.Errorf("container %q has an incompatible OpenTelemetry %s", container.Name, otelPythonPathKey)
			}
		}
	}
	return nil
}

func (r *otelResource) validateNodeJSInjection(configuredImage string, desiredInitContainer *corev1.Container) error {
	if err := r.validateCommonInjection(configuredImage, desiredInitContainer, otelNodeJSMountPath); err != nil {
		return err
	}

	for _, container := range r.pod.Spec.Containers {
		nodeOptionsCount := 0
		for idx := range container.Env {
			env := &container.Env[idx]
			if env.Name != otelNodeJSOptionsKey {
				continue
			}
			nodeOptionsCount++
			if nodeOptionsCount > 1 {
				return fmt.Errorf("container %q has duplicate %s entries", container.Name, otelNodeJSOptionsKey)
			}
			if env.ValueFrom != nil {
				return fmt.Errorf("container %q has %s defined via ValueFrom", container.Name, otelNodeJSOptionsKey)
			}
			if strings.Contains(env.Value, ddtraceMountPath) {
				return fmt.Errorf("container %q already uses the Datadog Node.js library", container.Name)
			}
			requireCount := countNodeJSRequireOptions(env.Value)
			pathCount := strings.Count(env.Value, strings.TrimPrefix(otelNodeJSMountPath, "/"))
			if requireCount > 1 || pathCount > 0 && (requireCount != 1 || pathCount != 1) {
				return fmt.Errorf("container %q has incompatible OpenTelemetry %s", container.Name, otelNodeJSOptionsKey)
			}
		}
	}
	return nil
}

func (r *otelResource) validateCommonInjection(configuredImage string, desiredInitContainer *corev1.Container, mountPath string) error {
	if strings.TrimSpace(configuredImage) == "" {
		return fmt.Errorf("image is empty")
	}
	if len(r.pod.Spec.Containers) == 0 {
		return fmt.Errorf("pod has no application containers")
	}

	for idx := range r.pod.Spec.InitContainers {
		container := &r.pod.Spec.InitContainers[idx]
		switch container.Name {
		case ddtraceInitContainerName:
			return fmt.Errorf("datadog init container %q already exists", ddtraceInitContainerName)
		case otelInitContainerName:
			if !otelInitContainerCompatible(container, desiredInitContainer) {
				return fmt.Errorf("init container %q already exists with incompatible configuration", otelInitContainerName)
			}
		}
	}

	for idx := range r.pod.Spec.Volumes {
		volume := &r.pod.Spec.Volumes[idx]
		if volume.Name == otelVolumeName && volume.EmptyDir == nil {
			return fmt.Errorf("volume %q already exists and is not an EmptyDir", otelVolumeName)
		}
	}

	for _, container := range r.pod.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == ddtraceMountPath || strings.TrimSuffix(mount.MountPath, "/") == ddtraceMountPath {
				return fmt.Errorf("container %q already uses the Datadog library mount path", container.Name)
			}
			usesOTelName := mount.Name == otelVolumeName
			usesOTelPath := mount.MountPath == mountPath
			if usesOTelName != usesOTelPath {
				return fmt.Errorf("container %q has an incompatible volume mount using name %q or path %q", container.Name, otelVolumeName, mountPath)
			}
			if usesOTelName && (mount.SubPath != "" || mount.SubPathExpr != "") {
				return fmt.Errorf("container %q has an incompatible volume mount using SubPath or SubPathExpr", container.Name)
			}
		}
	}
	return nil
}

func otelInitContainerCompatible(existing, desired *corev1.Container) bool {
	if existing.Image != desired.Image ||
		!reflect.DeepEqual(existing.Command, desired.Command) ||
		!reflect.DeepEqual(existing.Args, desired.Args) ||
		existing.ImagePullPolicy != desired.ImagePullPolicy ||
		!reflect.DeepEqual(existing.Resources, desired.Resources) {
		return false
	}
	if !securityContextCompatible(desired.SecurityContext, existing.SecurityContext) {
		return false
	}

	desiredMount, desiredFound := otelInitVolumeMount(desired.VolumeMounts)
	if !desiredFound {
		return false
	}

	ownedMountCount := 0
	for idx := range existing.VolumeMounts {
		mount := &existing.VolumeMounts[idx]
		if mount.Name != desiredMount.Name && mount.MountPath != desiredMount.MountPath {
			continue
		}
		ownedMountCount++
		if !reflect.DeepEqual(mount, desiredMount) {
			return false
		}
	}
	return ownedMountCount == 1
}

func securityContextCompatible(desired, existing *corev1.SecurityContext) bool {
	if desired == nil {
		return true
	}
	if existing == nil {
		return false
	}
	return k8sreflect.Equalities{}.DeepDerivative(desired, existing)
}

func otelInitVolumeMount(mounts []corev1.VolumeMount) (*corev1.VolumeMount, bool) {
	var result *corev1.VolumeMount
	for idx := range mounts {
		if mounts[idx].Name != otelVolumeName {
			continue
		}
		if result != nil {
			return nil, false
		}
		result = &mounts[idx]
	}
	return result, result != nil
}

func containsPythonPathComponent(value, path string) bool {
	for _, component := range strings.Split(value, ":") {
		if component == path || component == path+"/" {
			return true
		}
	}
	return false
}

func isOTelPythonPathInjected(value string) bool {
	components := strings.Split(value, ":")
	if len(components) < 2 || components[0] != otelPythonPathPrefix || components[len(components)-1] != otelPythonMountPath {
		return false
	}

	prefixCount := 0
	suffixCount := 0
	for _, component := range components {
		if component == otelPythonPathPrefix {
			prefixCount++
		}
		if component == otelPythonMountPath {
			suffixCount++
		}
	}
	return prefixCount == 1 && suffixCount == 1
}

func (r *otelResource) validateJavaInjection(configuredImage string, desiredInitContainer *corev1.Container) error {
	if err := r.validateCommonInjection(configuredImage, desiredInitContainer, otelJavaMountPath); err != nil {
		return err
	}

	for _, container := range r.pod.Spec.Containers {
		javaToolOptionsCount := 0
		for idx := range container.Env {
			env := &container.Env[idx]
			if env.Name != otelJavaToolOptionsKey {
				continue
			}
			javaToolOptionsCount++
			if javaToolOptionsCount > 1 {
				return fmt.Errorf("container %q has duplicate %s entries", container.Name, otelJavaToolOptionsKey)
			}
			if env.ValueFrom != nil {
				return fmt.Errorf("container %q has %s defined via ValueFrom", container.Name, otelJavaToolOptionsKey)
			}
			if strings.Contains(env.Value, ddJavaAgentFileName) {
				return fmt.Errorf("container %q already uses the Datadog Java agent", container.Name)
			}
		}
	}
	return nil
}

func selectOTelRule(rules []*config.OTelRule, annotations map[string]string) (*config.OTelRule, string) {
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if !rule.CheckAnnotation {
			return rule, ""
		}

		versionAnnotation := fmt.Sprintf(otelVersionAnnotationKeyFormat, language(rule.Language))
		if version, exists := annotations[versionAnnotation]; exists {
			return rule, version
		}
	}
	return nil, ""
}

func (r *otelResource) initContainer(rule *config.OTelRule, image string, library otelLibrary) corev1.Container {
	container := corev1.Container{
		Name:            otelInitContainerName,
		Image:           image,
		Command:         library.command,
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			{Name: otelVolumeName, MountPath: library.mountPath},
		},
	}
	if len(r.pod.Spec.Containers) > 0 && r.pod.Spec.Containers[0].SecurityContext != nil {
		container.SecurityContext = r.pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}
	setContainerResources(&container, rule.Resources.Requests.CPU, rule.Resources.Requests.Memory, rule.Resources.Limits.CPU, rule.Resources.Limits.Memory)
	return container
}

func (r *otelResource) injectInitContainer(container *corev1.Container) {
	manager.NewContainerManager(r.pod).AddInitContainer(container)
}

func (r *otelResource) injectVolume() {
	manager.NewVolumeManager(r.pod).AddVolume(&corev1.Volume{
		Name: otelVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
}

func (r *otelResource) injectJavaConfig() {
	for idx := range r.pod.Spec.Containers {
		container := &r.pod.Spec.Containers[idx]
		envIdx := envIndex(container.Env, otelJavaToolOptionsKey)
		if envIdx < 0 {
			container.Env = manager.AddOrUpdateEnvVar(
				container.Env,
				corev1.EnvVar{Name: otelJavaToolOptionsKey, Value: otelJavaAgentOption},
				manager.KeepExistingEnvVar,
			)
		} else if !containsJavaToolOption(container.Env[envIdx].Value, otelJavaAgentOption) {
			if container.Env[envIdx].Value == "" {
				container.Env[envIdx].Value = otelJavaAgentOption
			} else {
				container.Env[envIdx].Value += " " + otelJavaAgentOption
			}
		}
		manager.AddVolumeMountToContainer(container, &corev1.VolumeMount{Name: otelVolumeName, MountPath: otelJavaMountPath})
	}
}

func (r *otelResource) injectPythonConfig() {
	for idx := range r.pod.Spec.Containers {
		container := &r.pod.Spec.Containers[idx]
		envIdx := envIndex(container.Env, otelPythonPathKey)
		if envIdx >= 0 && isOTelPythonPathInjected(container.Env[envIdx].Value) {
			manager.AddVolumeMountToContainer(container, &corev1.VolumeMount{Name: otelVolumeName, MountPath: otelPythonMountPath})
			continue
		}
		if envIdx < 0 || container.Env[envIdx].Value == "" {
			pythonPath := otelPythonPathPrefix + ":" + otelPythonMountPath
			if envIdx < 0 {
				container.Env = manager.AddOrUpdateEnvVar(
					container.Env,
					corev1.EnvVar{Name: otelPythonPathKey, Value: pythonPath},
					manager.KeepExistingEnvVar,
				)
			} else {
				container.Env[envIdx].Value = pythonPath
			}
		} else {
			container.Env[envIdx].Value = otelPythonPathPrefix + ":" + container.Env[envIdx].Value + ":" + otelPythonMountPath
		}
		manager.AddVolumeMountToContainer(container, &corev1.VolumeMount{Name: otelVolumeName, MountPath: otelPythonMountPath})
	}
}

func (r *otelResource) injectNodeJSConfig() {
	for idx := range r.pod.Spec.Containers {
		container := &r.pod.Spec.Containers[idx]
		envIdx := envIndex(container.Env, otelNodeJSOptionsKey)
		if envIdx < 0 {
			container.Env = manager.AddOrUpdateEnvVar(
				container.Env,
				corev1.EnvVar{Name: otelNodeJSOptionsKey, Value: " " + otelNodeJSRequireOption},
				manager.KeepExistingEnvVar,
			)
		} else if !containsNodeJSRequireOption(container.Env[envIdx].Value) {
			container.Env[envIdx].Value += " " + otelNodeJSRequireOption
		}
		manager.AddVolumeMountToContainer(container, &corev1.VolumeMount{Name: otelVolumeName, MountPath: otelNodeJSMountPath})
	}
}

func containsNodeJSRequireOption(value string) bool {
	return countNodeJSRequireOptions(value) == 1
}

func countNodeJSRequireOptions(value string) int {
	fields := strings.Fields(value)
	count := 0
	for idx := 0; idx+1 < len(fields); idx++ {
		if fields[idx] == "--require" && fields[idx+1] == otelNodeJSRequirePath {
			count++
		}
	}
	return count
}

func containsJavaToolOption(value, option string) bool {
	for _, field := range strings.Fields(value) {
		if field == option {
			return true
		}
	}
	return false
}

func envIndex(envs []corev1.EnvVar, name string) int {
	for idx := range envs {
		if envs[idx].Name == name {
			return idx
		}
	}
	return -1
}
