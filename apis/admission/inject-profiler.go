package admission

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
	supportedLanguagesForProfiler = []language{java, python}
	commandsForProfiler           = [][]string{
		// java
		{
			"bash",
			"-c",
			"mv -f /app/async-profiler/* /app/datakit-profiler/; ./profiling.sh --add-crontab; cron -f",
		},
		// python
		{
			"bash",
			"-c",
			"./profiling.sh --add-crontab; cron -f",
		},
	}
)

func injectProfilerToPodTemplate(parent string, podTemplate *corev1.PodTemplateSpec) error {
	if podTemplate == nil {
		return fmt.Errorf("cannot inject profiler into nil podTemplate")
	}

	r := newProfilerResource(parent, podTemplate)
	r.process()
	return nil
}

type profilerResource struct {
	parent      string
	podTemplate *corev1.PodTemplateSpec
}

func newProfilerResource(parent string, podTemplate *corev1.PodTemplateSpec) *profilerResource {
	return &profilerResource{
		parent:      parent,
		podTemplate: podTemplate,
	}
}

func (r *profilerResource) process() {
	if !r.checkIfNeedsOperation() {
		return
	}

	image, commands, shouldInject := r.extractInfo()
	if !shouldInject {
		return
	}

	r.resetShareProcessNamespace()
	r.injectContainer(image, commands, profilerEnvs())
	r.injectVolume()
	r.injectVolumeMount()
}

func (r *profilerResource) checkIfNeedsOperation() bool {
	return !manager.NewContainerManager(r.podTemplate).ContainsContainer(profilerContainerName)
}

func (r *profilerResource) extractInfo() (string, []string, bool) {
	annotations := r.podTemplate.GetAnnotations()

	for idx, lang := range supportedLanguagesForProfiler {
		profilerVersionAnnotation := strings.ToLower(fmt.Sprintf(profilerVersionAnnotationKeyFormat, lang))

		if imageVersion, found := annotations[profilerVersionAnnotation]; found {
			image := profilerReleaseImage(lang, imageVersion)
			l.Infof("Use of %s-profiler image %s to %s", lang, image, r.parent)

			return image, commandsForProfiler[idx], true
		}
	}

	return "", nil, false
}

func (r *profilerResource) resetShareProcessNamespace() {
	var b = true
	r.podTemplate.Spec.ShareProcessNamespace = &b
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

	manager := manager.NewVolumeManager(r.podTemplate)
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

	manager := manager.NewVolumeMountManager(r.podTemplate)
	manager.AddVolumeMount(&workdir)
	manager.AddVolumeMount(&tmp)
	manager.AddVolumeMount(&timezone)
}

func (r *profilerResource) injectContainer(image string, commands []string, envs []struct{ Key, Value string }) {
	container := corev1.Container{
		Name:            profilerContainerName,
		Image:           image,
		Command:         commands,
		ImagePullPolicy: corev1.PullIfNotPresent,
		WorkingDir:      profilerMountPath,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_PTRACE"},
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

	manager.NewContainerManager(r.podTemplate).AddContainer(&container)
}

func profilerReleaseImage(lang language, imageVersion string) string {
	var image string

	switch lang {
	case java:
		image = profilerJavaImage()
	case python:
		image = profilerPythonImage()
	default:
		return ""
	}

	if imageVersion == "" {
		return image
	}

	imageName, _, _ := ParseImage(image)
	return fmt.Sprintf("%s:%s", imageName, imageVersion)
}
