package injector

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ddtraceInitContainerName          = "datakit-lib-init"
	ddtraceVersionAnnotationKeyFormat = "admission.datakit/%s-lib.version"
	ddtraceVolumeName                 = "datakit-auto-instrument"
	ddtraceMountPath                  = "/datadog-lib"

	// Java config
	javaToolOptionsKey   = "JAVA_TOOL_OPTIONS"
	javaToolOptionsValue = " -javaagent:/datadog-lib/dd-java-agent.jar"

	// Node config
	nodeOptionsKey   = "NODE_OPTIONS"
	nodeOptionsValue = " --require=/datadog-lib/node_modules/dd-trace/init"

	// Python config
	pythonPathKey   = "PYTHONPATH"
	pythonPathValue = "/datadog-lib/"
)

var supportedLanguagesForDDTrace = []language{java, js, python}

func InjectDDTraceToPod(parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject ddtrace-lib into nil pod")
	}

	r := newDDTraceResource(parent, pod)
	r.process()
	return nil
}

type ddtraceResource struct {
	parent string
	pod    *corev1.Pod
}

func newDDTraceResource(parent string, pod *corev1.Pod) *ddtraceResource {
	return &ddtraceResource{
		parent: parent,
		pod:    pod,
	}
}

func (r *ddtraceResource) process() {
	if !r.checkIfNeedsOperation() {
		return
	}

	lang, image, shouldInject := r.extractInfo()
	if !shouldInject {
		return
	}
	r.injectInitContainer(image)

	var err error
	switch lang {
	case java:
		err = r.injectConfig(javaToolOptionsKey, javaEnvValFunc)
	case js:
		err = r.injectConfig(nodeOptionsKey, jsEnvValFunc)
	case python:
		err = r.injectConfig(pythonPathKey, pythonEnvValFunc)
	default:
		err = fmt.Errorf("language %s is no supported, only supported %v", lang, supportedLanguagesForDDTrace)
	}

	if err != nil {
		l.Warnf("Unable to inject DDTrace into %s, err: %s", r.parent, err)
		return
	}

	r.injectVolume()
	r.injectEnvs(ddtraceEnvObjects())
}

func (r *ddtraceResource) checkIfNeedsOperation() bool {
	return !manager.NewContainerManager(r.pod).ContainsInitContainer(ddtraceInitContainerName)
}

func (r *ddtraceResource) extractInfo() (language, string, bool) {
	annotations := r.pod.GetAnnotations()

	for _, lang := range supportedLanguagesForDDTrace {
		versionAnnotation := strings.ToLower(fmt.Sprintf(ddtraceVersionAnnotationKeyFormat, lang))

		if imageVersion, found := annotations[versionAnnotation]; found {
			image := ddtraceReleaseImage(lang, imageVersion)
			l.Infof("Use of %s-agent image %s to %s", lang, image, r.parent)

			return lang, image, true
		}
	}

	return "", "", false
}

func (r *ddtraceResource) injectInitContainer(image string) {
	container := corev1.Container{
		Name:            ddtraceInitContainerName,
		Image:           image,
		Command:         []string{"sh", "copy-lib.sh", ddtraceMountPath},
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      ddtraceVolumeName,
				MountPath: ddtraceMountPath,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		},
	}
	manager.NewContainerManager(r.pod).AddInitContainer(&container)
}
func (r *ddtraceResource) injectConfig(envKey string, envVal envValFunc) error {
	podSpec := r.pod.Spec
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

		podSpec.Containers[i].VolumeMounts = append(podSpec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      ddtraceVolumeName,
				MountPath: ddtraceMountPath,
			})
	}
	return nil
}

func (r *ddtraceResource) injectVolume() {
	volume := corev1.Volume{
		Name: ddtraceVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	manager.NewVolumeManager(r.pod).AddVolume(&volume)
}

func (r *ddtraceResource) injectEnvs(envs []corev1.EnvVar) {
	m := manager.NewEnvVarManager(r.pod)
	for idx := range envs {
		m.AddEnvVar(&envs[idx])
	}
}

func envIndex(envs []corev1.EnvVar, name string) int {
	for i := range envs {
		if envs[i].Name == name {
			return i
		}
	}
	return -1
}

type envValFunc func(string) string

func javaEnvValFunc(predefinedVal string) string {
	return predefinedVal + javaToolOptionsValue
}

func jsEnvValFunc(predefinedVal string) string {
	return predefinedVal + nodeOptionsValue
}

func pythonEnvValFunc(predefinedVal string) string {
	if predefinedVal == "" {
		return pythonPathValue
	}
	return fmt.Sprintf("%s:%s", pythonPathValue, predefinedVal)
}

func ddtraceReleaseImage(lang language, imageVersion string) string {
	var image string

	switch lang {
	case java:
		image = ddtraceJavaAgentImage()
	case python:
		image = ddtracePythonAgentImage()
	case js:
		image = ddtraceJsAgentImage()
	default:
		return ""
	}

	if imageVersion == "" {
		return image
	}

	imageName, _, _ := ParseImage(image)
	return fmt.Sprintf("%s:%s", imageName, imageVersion)
}
