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

func InjectProfilerToPod(parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject profiler into nil pod")
	}

	r := newProfilerResource(parent, pod)
	r.process()
	return nil
}

type profilerResource struct {
	parent string
	pod    *corev1.Pod
}

func newProfilerResource(parent string, pod *corev1.Pod) *profilerResource {
	return &profilerResource{
		parent: parent,
		pod:    pod,
	}
}

func (r *profilerResource) process() {
	should, lang, imageVersion := r.shouldInject()
	if !should {
		return
	}

	image := profilerReleaseImage(lang, imageVersion)
	l.Infof("Use of %s-profiler image %s to %s", lang, image, r.parent)

	r.resetSpec()
	r.injectContainer(image, profilerEnvObjects())
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
			l.Debugf("profiler %s finds annotation for %s", lang, r.parent)
			return true, lang, imageVersion
		}
	}

	var lang string
	if v := profilerGetLanguageFromLabels(r.pod.GetLabels()); v != "" {
		lang = v
		l.Debugf("profiler %s finds labelSelector for %s", lang, r.parent)
	}
	if v := profilerGetLanguageFromNamespace(r.pod.Namespace); v != "" {
		lang = v
		l.Debugf("profiler %s finds namespace for %s", lang, r.parent)
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
		l.Infof("Found that the volumeMount with path %s already exists in %s, skip the injection.", profilerTmpMountPath, r.parent)
	} else {
		tmp := corev1.VolumeMount{
			Name:      profilerTmp,
			MountPath: profilerTmpMountPath,
		}
		manager.AddVolumeMount(&tmp)
	}

	if exists, _ := manager.FindVolumeMountPathInContainer(profilerTimezoneMountPath); exists {
		l.Infof("Found that the volumeMount with path %s already exists in %s, skip the injection.", profilerTimezoneMountPath, r.parent)
	} else {
		timezone := corev1.VolumeMount{
			Name:      profilerTimezone,
			MountPath: profilerTimezoneMountPath,
		}
		manager.AddVolumeMount(&timezone)
	}
}

func (r *profilerResource) injectContainer(image string, envs []corev1.EnvVar) {
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

	container.Env = append(container.Env, envs...)

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
