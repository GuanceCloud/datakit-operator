package injector

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	profilerContainerName              = "datakit-profiler"
	profilerVersionAnnotationKeyFormat = "admission.datakit/%s-profiler.version"

	profilerVolumeName        = "datakit-profiler-volume"
	profilerMountPath         = "/app/datakit-profiler"
	profilerTimezone          = "timezone"
	profilerTimezoneMountPath = "/etc/localtime"
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
	if !r.checkIfNeedsOperation() {
		return
	}

	image, shouldInject := r.extractInfo()
	if !shouldInject {
		return
	}

	r.resetSpec()
	r.injectContainer(image, profilerEnvs())
	r.injectVolume()
	r.injectVolumeMount()
}

func (r *profilerResource) checkIfNeedsOperation() bool {
	return !manager.NewContainerManager(r.pod).ContainsContainer(profilerContainerName)
}

func (r *profilerResource) extractInfo() (string, bool) {
	annotations := r.pod.GetAnnotations()

	for _, lang := range supportedLanguagesForProfiler {
		profilerVersionAnnotation := strings.ToLower(fmt.Sprintf(profilerVersionAnnotationKeyFormat, lang))

		if imageVersion, found := annotations[profilerVersionAnnotation]; found {
			image := profilerReleaseImage(lang, imageVersion)
			l.Infof("Use of %s-profiler image %s to %s", lang, image, r.parent)

			return image, true
		}
	}

	return "", false
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
		Name: "tmp",
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

	tmp := corev1.VolumeMount{
		Name:      "tmp",
		MountPath: "/tmp",
	}

	timezone := corev1.VolumeMount{
		Name:      profilerTimezone,
		MountPath: profilerTimezoneMountPath,
	}

	manager := manager.NewVolumeMountManager(r.pod)
	manager.AddVolumeMount(&workdir)
	manager.AddVolumeMount(&tmp)
	manager.AddVolumeMount(&timezone)
}

func (r *profilerResource) injectContainer(image string, envs []struct{ Key, Value string }) {
	container := corev1.Container{
		Name:            profilerContainerName,
		Image:           image,
		Command:         []string{"bash", "cmd.sh"},
		ImagePullPolicy: corev1.PullIfNotPresent,
		WorkingDir:      profilerMountPath,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_PTRACE", "SYS_ADMIN"},
			},
		},
	}

	for _, env := range envs {
		envvar := corev1.EnvVar{
			Name:  env.Key,
			Value: env.Value,
		}
		container.Env = append(container.Env, envvar)
	}

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
