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
	if language(rule.Language) != java {
		log.Warnf("otel language not supported: pod=%s, namespace=%s, rule=%s, language=%s", r.parent, r.namespace, rule.Name, rule.Language)
		return
	}

	image := rule.Image
	if imageVersion != "" {
		image = replaceImageVersion(image, imageVersion)
	}
	initContainer := r.javaInitContainer(rule, image)
	if err := r.validateJavaInjection(rule.Image, &initContainer); err != nil {
		log.Warnf("otel inject skipped: pod=%s, namespace=%s, rule=%s, reason=%v", r.parent, r.namespace, rule.Name, err)
		return
	}

	log.Infof("otel injection started: pod=%s, namespace=%s, language=%s, rule=%s", r.parent, r.namespace, rule.Language, rule.Name)
	r.injectInitContainer(&initContainer)
	r.injectVolume()
	r.injectJavaConfig()

	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)
	envs = envbuilder.FilterAndSetResourceFieldRefEnvVars(envs, r.pod)
	envManager := manager.NewEnvVarManager(r.pod)
	for idx := range envs {
		envManager.AddEnvVar(&envs[idx])
	}
	log.Infof("otel injection completed: pod=%s, namespace=%s, image=%s, rule=%s", r.parent, r.namespace, image, rule.Name)
}

func (r *otelResource) validateJavaInjection(configuredImage string, desiredInitContainer *corev1.Container) error {
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
			if !reflect.DeepEqual(container, desiredInitContainer) {
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

		for _, mount := range container.VolumeMounts {
			usesOTelName := mount.Name == otelVolumeName
			usesOTelPath := mount.MountPath == otelJavaMountPath
			if usesOTelName != usesOTelPath {
				return fmt.Errorf("container %q has an incompatible volume mount using name %q or path %q", container.Name, otelVolumeName, otelJavaMountPath)
			}
			if usesOTelName && (mount.SubPath != "" || mount.SubPathExpr != "") {
				return fmt.Errorf("container %q has an incompatible volume mount using SubPath or SubPathExpr", container.Name)
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

func (r *otelResource) javaInitContainer(rule *config.OTelRule, image string) corev1.Container {
	container := corev1.Container{
		Name:            otelInitContainerName,
		Image:           image,
		Command:         []string{"cp", otelJavaAgentSourcePath, otelJavaAgentPath},
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts: []corev1.VolumeMount{
			{Name: otelVolumeName, MountPath: otelJavaMountPath},
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
			container.Env = append(container.Env, corev1.EnvVar{Name: otelJavaToolOptionsKey, Value: otelJavaAgentOption})
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

func containsJavaToolOption(value, option string) bool {
	for _, field := range strings.Fields(value) {
		if field == option {
			return true
		}
	}
	return false
}
