// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	should, lang, imageVersion := r.shouldInject()
	if !should {
		return
	}

	image := profilerReleaseImage(lang, imageVersion)
	log.Infof("profiler use_image=%s lang=%s pod=%s", image, lang, r.parent)

	r.resetSpec()
	r.injectContainer(image)
	r.injectVolume()
	r.injectVolumeMount()
}

func (r *profilerResource) shouldInject() (bool, language, string) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), profilerEnabledAnnotationKey) {
		return false, null, ""
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(profilerContainerName) {
		return false, null, ""
	}

	annotations := r.pod.GetAnnotations()
	for _, lang := range supportedLanguagesForProfiler {
		profilerVersionAnnotation := strings.ToLower(fmt.Sprintf(profilerVersionAnnotationKeyFormat, lang))

		if imageVersion, found := annotations[profilerVersionAnnotation]; found {
			log.Debugf("profiler_find_annotation lang=%s pod=%s", lang, r.parent)
			return true, lang, imageVersion
		}
	}

	var lang string
	if v := profilerGetLanguageFromLabels(r.pod.GetLabels()); v != "" {
		lang = v
		log.Debugf("profiler_find_label lang=%s pod=%s", lang, r.parent)
	}
	if v := profilerGetLanguageFromNamespace(r.namespace); v != "" {
		lang = v
		log.Debugf("profiler_find_namespace lang=%s pod=%s", lang, r.parent)
	}

	switch language(lang) {
	case java, python, golang:
		return true, language(lang), ""
	default:
		// nil
	}

	return false, null, ""
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
		log.Infof("profiler_volume_mount_exists path=%s pod=%s", profilerTmpMountPath, r.parent)
	} else {
		tmp := corev1.VolumeMount{
			Name:      profilerTmp,
			MountPath: profilerTmpMountPath,
		}
		manager.AddVolumeMount(&tmp)
	}

	if exists, _ := manager.FindVolumeMountPathInContainer(profilerTimezoneMountPath); exists {
		log.Infof("profiler_volume_mount_exists path=%s pod=%s", profilerTimezoneMountPath, r.parent)
	} else {
		timezone := corev1.VolumeMount{
			Name:      profilerTimezone,
			MountPath: profilerTimezoneMountPath,
		}
		manager.AddVolumeMount(&timezone)
	}
}

func (r *profilerResource) injectContainer(image string) {
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
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{},
			Limits:   map[corev1.ResourceName]resource.Quantity{},
		},
	}

	// set requests
	cpuRequest, memoryRequest := profilerResourceRequests()
	if cpuRequest != "" {
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpuRequest)
	}
	if memoryRequest != "" {
		container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memoryRequest)
	}

	// set limits
	cpuLimit, memoryLimit := profilerResourceLimits()
	if cpuLimit != "" {
		container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLimit)
	}
	if memoryLimit != "" {
		container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(memoryLimit)
	}

	container.Env = append(container.Env, profilerEnvObjects()...)

	manager.NewContainerManager(r.pod).AddContainer(&container)
}

func profilerReleaseImage(lang language, imageVersion string) string {
	var image string

	switch lang {
	case java:
		image = profilerJavaImage()
	case python:
		image = profilerPythonImage()
	case golang:
		image = profilerGolangImage()
	default:
		return ""
	}

	if imageVersion == "" {
		return image
	}

	imageName, _, _ := ParseImage(image)
	return fmt.Sprintf("%s:%s", imageName, imageVersion)
}
