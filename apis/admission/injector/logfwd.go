package injector

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

func InjectLogfwdToPod(parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject logfwd into nil pod")
	}

	r := newLogfwdResource(parent, pod)
	r.process()
	return nil
}

type logfwdResource struct {
	parent string
	pod    *corev1.Pod
}

func newLogfwdResource(parent string, pod *corev1.Pod) *logfwdResource {
	return &logfwdResource{
		parent: parent,
		pod:    pod,
	}
}

func (r *logfwdResource) process() {
	if !r.checkIfNeedsOperation() {
		return
	}

	image := logfwdAppImage()
	config, volumePaths, shouldInject := r.extractInfo()
	if !shouldInject {
		return
	}

	var volumeNames []string
	for idx := range volumePaths {
		volumeNames = append(volumeNames, fmt.Sprintf("datakit-logfwd-volume-%d", idx))
	}

	r.injectVolume(volumeNames)

	// First, add volumeMount to the main container.
	r.injectVolumeMount(volumeNames, volumePaths)

	// Then create a logfwd container, the container needs to be ReadOnly.
	r.injectContainer(image, config, volumeNames, volumePaths)
}

func (r *logfwdResource) checkIfNeedsOperation() bool {
	if manager.NewContainerManager(r.pod).ContainsContainer(logfwdContainerName) {
		return false
	}

	annotations := r.pod.GetAnnotations()
	_, found := annotations[logfwdInstancesAnnotationKey]
	return found
}

func (r *logfwdResource) extractInfo() (string, []string, bool) {
	annotations := r.pod.GetAnnotations()
	instances, found := annotations[logfwdInstancesAnnotationKey]
	if !found {
		return "", nil, false
	}

	var configBuff bytes.Buffer
	if err := json.Compact(&configBuff, []byte(instances)); err != nil {
		l.Warnf("Logfwd of %s failed to compact config: %s, err: %s", r.parent, instances, err)
		return "", nil, false
	}

	var configs []*logfwdConfig

	if err := json.Unmarshal(configBuff.Bytes(), &configs); err != nil {
		l.Warnf("Logfwd of %s failed to unmarshal config: %s, err: %s", r.parent, instances, err)
		return "", nil, false
	}

	l.Infof("Use of logfwd instances to %s, config: %s", r.parent, configBuff.String())

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

	return configBuff.String(), Unique(paths), true
}

func (r *logfwdResource) injectVolume(volumeNames []string) {
	manager := manager.NewVolumeManager(r.pod)
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

func (r *logfwdResource) injectVolumeMount(volumeNames, volumePaths []string) {
	manager := manager.NewVolumeMountManager(r.pod)
	for idx := range volumeNames {
		volumeMount := corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
		}
		manager.AddVolumeMount(&volumeMount)
	}
}

func (r *logfwdResource) injectContainer(image, config string, volumeNames, volumePaths []string) {
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

	manager.NewContainerManager(r.pod).AddContainer(&container)
}
