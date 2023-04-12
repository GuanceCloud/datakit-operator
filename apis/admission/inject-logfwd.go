package admission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	logfwdContainerName          = "datakit-logfwd"
	logfwdInstancesAnnotationKey = "admission.datakit/logfwd.instances"

	logfwdJSONConfigKey   = "LOGFWD_JSON_CONFIG"
	logfwdPodNameKey      = "LOGFWD_POD_NAME"
	logfwdPodNamespaceKey = "LOGFWD_POD_NAMESPACE"
)

type logfwdConfig struct {
	DataKitAddr string `json:"datakit_addr"`
	Loggings    []struct {
		LogFiles              []string          `json:"logfiles"`
		Ignore                []string          `json:"ignore"`
		Source                string            `json:"source"`
		Service               string            `json:"service"`
		Pipeline              string            `json:"pipeline"`
		CharacterEncoding     string            `json:"character_encoding"`
		MultilineMatch        string            `json:"multiline_match"`
		RemoveAnsiEscapeCodes bool              `json:"remove_ansi_escape_codes"`
		Tags                  map[string]string `json:"tags"`
	} `json:"loggings"`
}

func injectLogfwdToPodTemplate(parent string, podTemplate *corev1.PodTemplateSpec) error {
	if podTemplate == nil {
		return fmt.Errorf("cannot inject logfwd into nil podTemplate")
	}

	if manager.NewContainerManager(podTemplate).ContainsContainer(logfwdContainerName) {
		return nil
	}

	config, volumePaths, shouldInject := extractLogfwdInfo(parent, podTemplate)
	if !shouldInject {
		return nil
	}

	image := logfwdAppImage()
	injectLogfwd(podTemplate, image, config, volumePaths)
	return nil
}

func extractLogfwdInfo(parent string, podTemplate *corev1.PodTemplateSpec) (string, []string, bool) {
	annotations := podTemplate.GetAnnotations()
	instances, found := annotations[logfwdInstancesAnnotationKey]
	if !found {
		return "", nil, false
	}

	var configBuff bytes.Buffer
	if err := json.Compact(&configBuff, []byte(instances)); err != nil {
		l.Warnf("Logfwd of %s failed to compact config: %s, err: %s", parent, instances, err)
		return "", nil, false
	}

	var configs []*logfwdConfig

	if err := json.Unmarshal(configBuff.Bytes(), &configs); err != nil {
		l.Warnf("Logfwd of %s failed to unmarshal config: %s, err: %s", parent, instances, err)
		return "", nil, false
	}

	l.Infof("Use of logfwd instances to %s, config: %s", parent, configBuff.String())

	var paths []string
	for _, cfg := range configs {
		for _, logging := range cfg.Loggings {
			for _, file := range logging.LogFiles {
				paths = append(paths, filepath.Dir(file))
			}

			for _, ignoreFile := range logging.Ignore {
				paths = append(paths, filepath.Dir(ignoreFile))
			}
		}
	}

	return configBuff.String(), unique(paths), true
}

func injectLogfwd(podTemplate *corev1.PodTemplateSpec, image, config string, volumePaths []string) {
	var volumeNames []string
	for idx := range volumePaths {
		volumeNames = append(volumeNames, fmt.Sprintf("datakit-logfwd-volume-%d", idx))
	}

	injectLogfwdVolume(podTemplate, volumeNames)
	injectLogfwdVolumeMount(podTemplate, volumeNames, volumePaths)
	injectLogfwdContainer(podTemplate, image, config, volumeNames, volumePaths)
}

func injectLogfwdContainer(podTemplate *corev1.PodTemplateSpec, image, config string, volumeNames, volumePaths []string) {
	container := corev1.Container{
		Name:            logfwdContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{
				Name:  logfwdJSONConfigKey,
				Value: config,
			},
			{
				Name: logfwdPodNameKey,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
			{
				Name: logfwdPodNamespaceKey,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
			},
		},
	}

	for idx := range volumeNames {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
			ReadOnly:  true,
		})
	}

	manager.NewContainerManager(podTemplate).AddContainer(&container)
}

func injectLogfwdVolume(podTemplate *corev1.PodTemplateSpec, volumeNames []string) {
	manager := manager.NewVolumeManager(podTemplate)
	for _, name := range volumeNames {
		volume := corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		manager.AddVolume(&volume)
	}
}

func injectLogfwdVolumeMount(podTemplate *corev1.PodTemplateSpec, volumeNames, volumePaths []string) {
	manager := manager.NewVolumeMountManager(podTemplate)
	for idx := range volumeNames {
		volumeMount := corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
		}
		manager.AddVolumeMount(&volumeMount)
	}
}

func unique(slice []string) []string {
	var uniqMap = make(map[string]struct{})
	var uniqSlice []string
	for _, s := range slice {
		_, exist := uniqMap[s]
		if exist {
			continue
		}
		uniqMap[s] = struct{}{}
		uniqSlice = append(uniqSlice, s)
	}
	return uniqSlice
}
