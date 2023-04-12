package admission

import (
	"fmt"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
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

	ddAgentHostKey      = "DD_AGENT_HOST"
	ddTraceAgentPortKey = "DD_TRACE_AGENT_PORT"
)

type language string

const (
	java   language = "java"
	js     language = "js"
	python language = "python"
)

var supportedLanguages = []language{java, js, python}

func injectLibToPodTemplate(parent string, podTemplate *corev1.PodTemplateSpec) error {
	if podTemplate == nil {
		return fmt.Errorf("cannot inject lib into nil podTemplate")
	}

	if manager.NewContainerManager(podTemplate).ContainsInitContainer(initContainerName) {
		return nil
	}

	lang, image, shouldInject := extractLibInfo(parent, podTemplate)
	if !shouldInject {
		return nil
	}

	return injectLibContainer(podTemplate, lang, image)
}

func extractLibInfo(parent string, podTemplate *corev1.PodTemplateSpec) (language, string, bool) {
	annotations := podTemplate.GetAnnotations()

	for _, lang := range supportedLanguages {
		libVersionAnnotation := strings.ToLower(fmt.Sprintf(libVersionAnnotationKeyFormat, lang))

		if imageVersion, found := annotations[libVersionAnnotation]; found {
			image := libReleaseImage(lang, imageVersion)
			l.Infof("Use of %s-agent image %s to %s", lang, image, parent)

			return lang, image, true
		}
	}

	return "", "", false
}

func injectLibContainer(podTemplate *corev1.PodTemplateSpec, lang language, image string) error {
	injectLibInitContainer(podTemplate, image)

	var err error
	switch lang {
	case java:
		err = injectLibConfig(podTemplate, javaToolOptionsKey, javaEnvValFunc)
	case js:
		err = injectLibConfig(podTemplate, nodeOptionsKey, jsEnvValFunc)
	case python:
		err = injectLibConfig(podTemplate, pythonPathKey, pythonEnvValFunc)
	default:
		err = fmt.Errorf("language %s is no supported, only supported %v", lang, supportedLanguages)
	}

	if err != nil {
		return err
	}

	injectLibVolume(podTemplate)
	injectLibEnv(podTemplate)
	return nil
}

func injectLibInitContainer(podTemplate *corev1.PodTemplateSpec, image string) {
	container := corev1.Container{
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
	}
	manager.NewContainerManager(podTemplate).AddInitContainer(&container)
}

func injectLibConfig(podTemplate *corev1.PodTemplateSpec, envKey string, envVal envValFunc) error {
	podSpec := podTemplate.Spec
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
				Name:      volumeName,
				MountPath: mountPath,
			})
	}
	return nil
}

func injectLibVolume(podTemplate *corev1.PodTemplateSpec) {
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	manager.NewVolumeManager(podTemplate).AddVolume(&volume)
}

func injectLibEnv(podTemplate *corev1.PodTemplateSpec) {
	ddHostEnvVar := corev1.EnvVar{
		Name:  ddAgentHostKey,
		Value: ddAgentHost(),
	}
	ddPortEnvVar := corev1.EnvVar{
		Name:  ddTraceAgentPortKey,
		Value: ddTraceAgentPort(),
	}
	m := manager.NewEnvVarManager(podTemplate)
	m.AddEnvVar(&ddHostEnvVar)
	m.AddEnvVar(&ddPortEnvVar)
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
