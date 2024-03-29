package admission

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	initContainerName = "datakit-lib-init"
	volumeName        = "datakit-auto-instrument"
	mountPath         = "/datadog-lib"

	// Java config
	javaToolOptionsKey   = "JAVA_TOOL_OPTIONS"
	javaToolOptionsValue = " -javaagent:/datadog-lib/dd-java-agent.jar"

	// Node config
	nodeOptionsKey   = "NODE_OPTIONS"
	nodeOptionsValue = " --require=/datadog-lib/node_modules/dd-trace/init"

	// Python config
	pythonPathKey   = "PYTHONPATH"
	pythonPathValue = "/datadog-lib/"

	admissionEnableAnnotationKey  = "admission.datakit"
	libVersionAnnotationKeyFormat = "admission.datakit/%s-lib.version"
)

type language string

const (
	java   language = "java"
	js     language = "js"
	python language = "python"
)

var supportedLanguages = []language{java, js, python}

func injectLib(rawPod []byte) ([]byte, error) {
	return mutatePod(rawPod, injectAutoInstrumentation)
}

func injectAutoInstrumentation(pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject lib into nil pod")
	}

	if containsInitContainer(pod, initContainerName) {
		return nil
	}

	lang, image, shouldInject := extractLibInfo(pod)
	if !shouldInject {
		return nil
	}

	return injectAutoInstrumentationConfig(pod, lang, image)
}

func extractLibInfo(pod *corev1.Pod) (language, string, bool) {
	podAnnotations := pod.GetAnnotations()

	for _, lang := range supportedLanguages {
		libVersionAnnotation := strings.ToLower(fmt.Sprintf(libVersionAnnotationKeyFormat, lang))
		if imageVersion, found := podAnnotations[libVersionAnnotation]; found {
			image := libReleaseImage(lang, imageVersion)
			l.Infof("Use of %s-agent image %s to Pod %s", lang, image, pod.GetName())

			return lang, image, true
		}
	}

	return "", "", false
}

func injectAutoInstrumentationConfig(pod *corev1.Pod, lang language, image string) error {
	switch lang {
	case java:
		injectInitContainer(pod, image)
		if err := injectLibConfig(pod, javaToolOptionsKey, javaEnvValFunc); err != nil {
			return err
		}
	case js:
		injectInitContainer(pod, image)
		if err := injectLibConfig(pod, nodeOptionsKey, jsEnvValFunc); err != nil {
			return err
		}
	case python:
		injectInitContainer(pod, image)
		if err := injectLibConfig(pod, pythonPathKey, pythonEnvValFunc); err != nil {
			return err
		}
	default:
		return fmt.Errorf("language %s is no supported, only supported %v", lang, supportedLanguages)
	}

	injectLibVolume(pod)
	return nil
}

func injectInitContainer(pod *corev1.Pod, image string) {
	pod.Spec.InitContainers = append([]corev1.Container{
		{
			Name:            initContainerName,
			Image:           image,
			Command:         []string{"sh", "copy-lib.sh", mountPath},
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: mountPath,
				},
			},
		},
	}, pod.Spec.InitContainers...)
}

func injectLibConfig(pod *corev1.Pod, envKey string, envVal envValFunc) error {
	for i, container := range pod.Spec.Containers {
		index := envIndex(container.Env, envKey)

		if index < 0 {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  envKey,
				Value: envVal(""),
			})
		} else {
			if pod.Spec.Containers[i].Env[index].ValueFrom != nil {
				return fmt.Errorf("%q is defined via ValueFrom", envKey)
			}

			pod.Spec.Containers[i].Env[index].Value = envVal(pod.Spec.Containers[i].Env[index].Value)
		}

		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{Name: volumeName, MountPath: mountPath})
	}
	return nil
}

func injectLibVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	})
}

func containsInitContainer(pod *corev1.Pod, name string) bool {
	for _, container := range pod.Spec.InitContainers {
		if container.Name == name {
			return true
		}
	}
	return false
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

func libReleaseImage(lang language, imageVersion string) string {
	var image string

	switch lang {
	case java:
		image = javaAgentImage()
	case python:
		image = pythonAgentImage()
	case js:
		image = jsAgentImage()
	default:
		return ""
	}

	if imageVersion == "" {
		return image
	}

	imageName, _, _ := ParseImage(image)
	return fmt.Sprintf("%s:%s", imageName, imageVersion)
}

// ParseImage adapts some of the logic from the actual Docker library's image parsing
// routines:
// https://github.com/docker/distribution/blob/release/2.7/reference/normalize.go
func ParseImage(image string) (string, string, string) {
	var domain, remainder string

	i := strings.IndexRune(image, '/')

	if i == -1 || (!strings.ContainsAny(image[:i], ".:") && image[:i] != "localhost") {
		remainder = image
	} else {
		domain, remainder = image[:i], image[i+1:]
	}

	var imageName string
	imageVersion := "unknown"

	i = strings.LastIndex(remainder, ":")
	if i > -1 {
		imageVersion = remainder[i+1:]
		imageName = remainder[:i]
	} else {
		imageName = remainder
	}

	if domain != "" {
		imageName = domain + "/" + imageName
	}

	shortName := imageName
	if imageBlock := strings.Split(imageName, "/"); len(imageBlock) > 0 {
		// there is no need to do
		// Split not return empty slice
		shortName = imageBlock[len(imageBlock)-1]
	}

	return imageName, shortName, imageVersion
}
