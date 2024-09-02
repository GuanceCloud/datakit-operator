package injector

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	logfwdContainerName          = "datakit-logfwd"
	logfwdInstancesAnnotationKey = "admission.datakit/logfwd.instances"
	logfwdJSONConfigKey          = "LOGFWD_JSON_CONFIG"
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

	image := logfwdImage()
	config, needVolumePaths, shouldInject := r.extractInfo()
	if !shouldInject {
		return
	}

	if !logfwdReuseExistVolume() {
		unique, name, path := r.checkVolumeUnique(needVolumePaths)
		if !unique {
			l.Warnf("The volumeMounts must be unique, found %s(%s) on %s, please enable 'reuse_exist_volume'.",
				path, name, r.parent)
			return
		}
	}

	volumeNames, volumePaths := r.getVolumePairs(logfwdReuseExistVolume(), needVolumePaths)
	r.injectVolume(volumeNames)
	// First, add volumeMount to the main container.
	r.injectVolumeMount(volumeNames, volumePaths)

	// Then create a logfwd container, the container needs to be ReadOnly.
	envs := logfwdEnvObjects()
	envs = append(envs, corev1.EnvVar{Name: logfwdJSONConfigKey, Value: config})
	r.injectContainer(image, envs, volumeNames, volumePaths)
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
			paths = append(paths, getMountPaths(logging.LogFiles)...)
			paths = append(paths, getMountPaths(logging.Ignore)...)
		}
	}

	return configBuff.String(), unique(paths), true
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

func (r *logfwdResource) injectContainer(image string, envs []corev1.EnvVar, volumeNames, volumePaths []string) {
	container := corev1.Container{
		Name:            logfwdContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envs,
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
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

func (r *logfwdResource) getVolumePairs(reuse bool, needVolumePaths []string) (volumeNames, volumePaths []string) {
	volumePaths = needVolumePaths

	if !reuse {
		for idx := range needVolumePaths {
			volumeNames = append(volumeNames, fmt.Sprintf("datakit-logfwd-volume-%d", idx))
		}
		return
	}

	volumeManager := manager.NewVolumeManager(r.pod)
	volumeMountManager := manager.NewVolumeMountManager(r.pod)

	for idx := range needVolumePaths {
		exist, name := volumeMountManager.FindVolumeMountPathInContainer(needVolumePaths[idx])

		if exist && volumeManager.IsEmptyDirVolume(name) {
			l.Infof("Reuse volume %s for mountPath %s on %s", name, needVolumePaths[idx], r.parent)
		} else {
			name = fmt.Sprintf("datakit-logfwd-volume-%d", idx)
		}

		volumeNames = append(volumeNames, name)
	}

	return
}

func (r *logfwdResource) checkVolumeUnique(needVolumePaths []string) (bool, string, string) {
	volumeMountManager := manager.NewVolumeMountManager(r.pod)
	for idx := range needVolumePaths {
		_, name := volumeMountManager.FindVolumeMountPathInContainer(needVolumePaths[idx])
		if name != "" { // found equal path, not unique.
			return false, name, needVolumePaths[idx]
		}
	}
	return true, "", ""
}
