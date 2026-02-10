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
	profilerContainerName = "datakit-profiler"

	profilerEnabledAnnotationKey       = "admission.datakit/profiler.enabled"
	profilerVersionAnnotationKeyFormat = "admission.datakit/%s-profiler.version"

	profilerVolumeName        = "datakit-profiler-volume"
	profilerMountPath         = "/app/datakit-profiler"
	profilerTimezone          = "timezone"
	profilerTimezoneMountPath = "/etc/localtime"
	profilerTmp               = "tmp"
	profilerTmpMountPath      = "/tmp"
)

var (
	supportedLanguagesForProfiler = []language{java, python, golang}
)

func InjectProfilerToPod(namespace, parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject profiler into nil pod")
	}

	r := newProfilerResource(namespace, parent, pod)
	r.process()
	return nil
}

type profilerResource struct {
	namespace string
	parent    string
	pod       *corev1.Pod
}

func newProfilerResource(namespace, parent string, pod *corev1.Pod) *profilerResource {
	return &profilerResource{
		namespace: namespace,
		parent:    parent,
		pod:       pod,
	}
}

func (r *profilerResource) process() {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	should, rule, lang, imageVersion := r.getMatchingRule()
	if !should || rule == nil {
		return
	}

	// 从 rule.Images 中获取对应 language 的 image
	image := r.getImageFromRule(rule, lang)
	if image == "" {
		log.Warnf("profiler image not found for language %s: pod=%s", lang, r.parent)
		return
	}

	// 如果 annotation 中有版本信息，替换 image 版本
	if imageVersion != "" {
		image = replaceImageVersion(image, imageVersion)
	}

	log.Infof("profiler injection started: pod=%s, language=%s, image=%s, rule=%s", r.parent, lang, image, rule.Name)

	r.resetSpec()
	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)
	envs = envbuilder.FilterAndSetResourceFieldRefEnvVars(envs, r.pod)
	r.injectContainer(image, rule.Resources, envs)
	r.injectVolume()
	r.injectVolumeMount()
	log.Infof("profiler injection completed: pod=%s, image=%s, rule=%s", r.parent, image, rule.Name)
}

func (r *profilerResource) getMatchingRule() (matched bool, ruleConfig *config.InjectRule, lang language, imageVersion string) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), profilerEnabledAnnotationKey) {
		log.Debugf("profiler annotation disabled: pod=%s", r.parent)
		return false, nil, "", ""
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(profilerContainerName) {
		log.Debugf("profiler container already exists: pod=%s", r.parent)
		return false, nil, "", ""
	}

	matched, rule := profilerMatchNamespaceOrLabelsForConfig(r.namespace, r.pod.GetLabels())
	if !matched || rule == nil {
		return false, nil, "", ""
	}

	if rule.CheckAnnotation {
		annotations := r.pod.GetAnnotations()
		for _, l := range supportedLanguagesForProfiler {
			profilerVersionAnnotation := fmt.Sprintf(profilerVersionAnnotationKeyFormat, l)
			if version, exists := annotations[profilerVersionAnnotation]; exists {
				log.Infof("profiler rule matched for annotation %s-profiler.version: pod=%s", l, r.parent)
				return true, rule, l, version
			}
		}
		log.Debugf("profiler rule requires profiler version annotation but not found: pod=%s", r.parent)
		return false, nil, "", ""
	}

	lang = language(rule.Language)
	if lang == "" {
		log.Debugf("profiler language not specified in rule: pod=%s", r.parent)
		return false, nil, "", ""
	}

	switch lang {
	case java, python, golang:
		log.Infof("profiler rule matched: pod=%s, language=%s, image=%s", r.parent, lang, rule.Image)
		return true, rule, lang, ""
	default:
		log.Debugf("profiler language not supported: lang=%s pod=%s", lang, r.parent)
		return false, nil, "", ""
	}
}

func (r *profilerResource) getImageFromRule(rule *config.InjectRule, lang language) string {
	// 从 rule.Images map 中获取对应 language 的 image
	var imageKey string
	switch lang {
	case java:
		imageKey = config.DeprecatedProfilerJavaImageKey
	case python:
		imageKey = config.DeprecatedProfilerPythonImageKey
	case golang:
		imageKey = config.DeprecatedProfilerGolangImageKey
	default:
		return ""
	}

	if rule.Images != nil {
		if image, exists := rule.Images[imageKey]; exists && image != "" {
			return image
		}
	}

	// 如果 Images map 中没有找到，尝试使用 rule.Image（向后兼容）
	return rule.Image
}

func (r *profilerResource) resetSpec() {
	var b = true
	r.pod.Spec.ShareProcessNamespace = &b
	r.pod.Spec.RestartPolicy = corev1.RestartPolicyAlways
}

func (r *profilerResource) injectVolume() {
	workdir := corev1.Volume{
		Name: profilerVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	tmp := corev1.Volume{
		Name: profilerTmp,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	fileOrCreate := corev1.HostPathFileOrCreate
	timezone := corev1.Volume{
		Name: profilerTimezone,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: profilerTimezoneMountPath,
				Type: &fileOrCreate,
			},
		},
	}

	manager := manager.NewVolumeManager(r.pod)
	manager.AddVolume(&workdir)
	manager.AddVolume(&tmp)
	manager.AddVolume(&timezone)
}

func (r *profilerResource) injectVolumeMount() {
	workdir := corev1.VolumeMount{
		Name:      profilerVolumeName,
		MountPath: profilerMountPath,
	}

	manager := manager.NewVolumeMountManager(r.pod)
	// This is a special volumeMount, do not need to check for duplicates.
	manager.AddVolumeMount(&workdir)

	if exists, _ := manager.FindVolumeMountPathInContainer(profilerTmpMountPath); exists {
		log.Infof("Found that the volumeMount with path %s already exists in %s, skip the injection.", profilerTmpMountPath, r.parent)
	} else {
		tmp := corev1.VolumeMount{
			Name:      profilerTmp,
			MountPath: profilerTmpMountPath,
		}
		manager.AddVolumeMount(&tmp)
	}

	if exists, _ := manager.FindVolumeMountPathInContainer(profilerTimezoneMountPath); exists {
		log.Infof("Found that the volumeMount with path %s already exists in %s, skip the injection.", profilerTimezoneMountPath, r.parent)
	} else {
		timezone := corev1.VolumeMount{
			Name:      profilerTimezone,
			MountPath: profilerTimezoneMountPath,
		}
		manager.AddVolumeMount(&timezone)
	}
}

func (r *profilerResource) injectContainer(image string, resources config.ResourceRequirements, envs []corev1.EnvVar) {
	container := corev1.Container{
		Name:            profilerContainerName,
		Image:           image,
		Command:         []string{"bash", "cmd.sh"},
		ImagePullPolicy: corev1.PullAlways,
		WorkingDir:      profilerMountPath,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_PTRACE", "SYS_ADMIN"},
			},
		},
	}

	setContainerResources(&container, resources.Requests.CPU, resources.Requests.Memory, resources.Limits.CPU, resources.Limits.Memory)
	container.Env = append(container.Env, envs...)

	manager.NewContainerManager(r.pod).AddContainer(&container)
}
