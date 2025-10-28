// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

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
	logfwdContainerName = "datakit-logfwd"

	logfwdEnabledAnnotationKey     = "admission.datakit/logfwd.enabled"
	logfwdInstancesAnnotationKey   = "admission.datakit/logfwd.instances"
	logfwdLogConfigsAnnotationKey  = "admission.datakit/logfwd.log_configs"
	logfwdVolumePathsAnnotationKey = "admission.datakit/logfwd.volume_paths"

	logfwdJSONConfigKey = "LOGFWD_JSON_CONFIG"
	logfwdLogConfigsKey = "LOGFWD_LOG_CONFIGS"
)

type logfwdConfig struct {
	DataKitAddr string `json:"datakit_addr"`
	Loggings    []struct {
		LogFiles []string `json:"logfiles"`
		Ignore   []string `json:"ignore"`
	} `json:"loggings"`
}

type logConfig struct {
	Path string `json:"path"`
}

func InjectLogfwdToPod(_, parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject logfwd into nil pod")
	}

	log.Debugf("logfwd injection start: pod=%s", parent)
	r := newLogfwdResource(parent, pod)
	r.process()
	log.Debugf("logfwd injection completed: pod=%s", parent)
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
	if !r.shouldInject() {
		log.Debugf("logfwd injection skipped: pod=%s, reason=should_inject_false", r.parent)
		return
	}

	log.Infof("logfwd injection processing: pod=%s", r.parent)
	image := logfwdImage()
	// Then create a logfwd container, the container needs to be ReadOnly.
	envs := logfwdEnvObjects()

	instancesConfig, instancesVolumePaths, hasInstancesConfig := r.extractInstancesConfig()
	if hasInstancesConfig {
		envs = append(envs, corev1.EnvVar{Name: logfwdJSONConfigKey, Value: instancesConfig})
		log.Debugf("logfwd instances config added: pod=%s", r.parent)
	}

	logConfigsConfig, logConfigsVolumePaths, hasLogConfigsConfig := r.extractLogConfigsConfig()
	if hasLogConfigsConfig {
		envs = append(envs, corev1.EnvVar{Name: logfwdLogConfigsKey, Value: logConfigsConfig})
		log.Debugf("logfwd log_configs config added: pod=%s", r.parent)
	}

	if !(hasInstancesConfig || hasLogConfigsConfig) {
		log.Debugf("logfwd injection skipped: pod=%s, reason=no_valid_config", r.parent)
		return
	}

	allVolumePaths := append(instancesVolumePaths, logConfigsVolumePaths...)

	// 处理 volume_paths 注解
	volumePathsFromAnnotation := r.extractVolumePathsAnnotation()
	allVolumePaths = append(allVolumePaths, volumePathsFromAnnotation...)

	volumeNames, volumeMountPaths := r.getVolumePairs(unique(allVolumePaths))
	r.injectVolume(volumeNames)
	r.injectVolumeMount(volumeNames, volumeMountPaths)

	r.injectContainer(image, envs, volumeNames, volumeMountPaths)
	log.Infof("logfwd container injected successfully: pod=%s", r.parent)
}

func (r *logfwdResource) shouldInject() bool {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), logfwdEnabledAnnotationKey) {
		return false
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(logfwdContainerName) {
		return false
	}

	annotations := r.pod.GetAnnotations()
	_, foundInstances := annotations[logfwdInstancesAnnotationKey]
	_, foundLogConfigs := annotations[logfwdLogConfigsAnnotationKey]
	return foundInstances || foundLogConfigs
}

func extractAndProcessConfig(
	pod *corev1.Pod,
	parent, annotationKey, logType string,
	processConfigs func([]byte) ([]string, error),
) (string, []string, bool) {

	annotations := pod.GetAnnotations()
	configStr, found := annotations[annotationKey]
	if !found {
		return "", nil, false
	}

	// 如果注解存在但值为空，视为有效配置但内容为空
	if configStr == "" {
		log.Infof("logfwd use_%s pod=%s config=%s", logType, parent, configStr)
		return configStr, []string{}, true
	}

	var configBuff bytes.Buffer
	if err := json.Compact(&configBuff, []byte(configStr)); err != nil {
		log.Warnf("logfwd config compact failed: pod=%s, type=%s, error=%v, config_length=%d", parent, logType, err, len(configStr))
		return "", nil, false
	}

	log.Infof("logfwd use_%s pod=%s config=%s", logType, parent, configBuff.String())

	paths, err := processConfigs(configBuff.Bytes())
	if err != nil {
		log.Warnf("logfwd config processing failed: pod=%s, type=%s, error=%v, config_length=%d", parent, logType, err, len(configBuff.String()))
		return "", nil, false
	}

	return configBuff.String(), unique(paths), true
}

func processInstancesConfig(data []byte) ([]string, error) {
	var configs []*logfwdConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instances config: %w", err)
	}

	var paths []string
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		for _, logging := range cfg.Loggings {
			paths = append(paths, getMountPaths(logging.LogFiles)...)
			paths = append(paths, getMountPaths(logging.Ignore)...)
		}
	}
	return paths, nil
}

func processLogConfigsConfig(data []byte) ([]string, error) {
	var configs []*logConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log_configs config: %w", err)
	}

	var paths []string
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		paths = append(paths, getMountPath(cfg.Path))
	}
	return paths, nil
}

func (r *logfwdResource) extractInstancesConfig() (string, []string, bool) {
	return extractAndProcessConfig(r.pod, r.parent, logfwdInstancesAnnotationKey, "instances", processInstancesConfig)
}

func (r *logfwdResource) extractLogConfigsConfig() (string, []string, bool) {
	return extractAndProcessConfig(r.pod, r.parent, logfwdLogConfigsAnnotationKey, "log_configs", processLogConfigsConfig)
}

func (r *logfwdResource) extractVolumePathsAnnotation() []string {
	annotations := r.pod.GetAnnotations()
	if annotations == nil {
		return nil
	}

	volumePathsStr, exists := annotations[logfwdVolumePathsAnnotationKey]
	if !exists || volumePathsStr == "" {
		return nil
	}

	var volumePaths []string
	if err := json.Unmarshal([]byte(volumePathsStr), &volumePaths); err != nil {
		log.Warnf("logfwd volume paths unmarshal failed: pod=%s, error=%v, raw_length=%d", r.parent, err, len(volumePathsStr))
		return nil
	}

	log.Infof("logfwd use_volume_paths pod=%s paths=%v", r.parent, volumePaths)
	// 处理路径，转换为挂载路径
	return getMountPaths(volumePaths)
}

func (r *logfwdResource) injectVolume(volumeNames []string) {
	if len(volumeNames) == 0 {
		return
	}
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

	podInfoVolume := corev1.Volume{
		Name: "datakit-pod-info",
		VolumeSource: corev1.VolumeSource{
			DownwardAPI: &corev1.DownwardAPIVolumeSource{
				Items: []corev1.DownwardAPIVolumeFile{
					{
						Path: "labels",
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.labels",
						},
					},
				},
			},
		},
	}
	manager.AddVolume(&podInfoVolume)
	log.Debugf("logfwd injecting volumes: pod=%s, volumes=%v", r.parent, volumeNames)
}

func (r *logfwdResource) injectVolumeMount(volumeNames, volumePaths []string) {
	if len(volumeNames) == 0 {
		return
	}
	manager := manager.NewVolumeMountManager(r.pod)
	for idx := range volumeNames {
		volumeMount := corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
		}
		manager.AddVolumeMount(&volumeMount)
	}
	log.Debugf("logfwd injecting volumeMounts: pod=%s, mounts=%v", r.parent, volumePaths)
}

func (r *logfwdResource) injectContainer(image string, envs []corev1.EnvVar, volumeNames, volumePaths []string) {
	log.Debugf("logfwd injecting container: pod=%s, image=%s", r.parent, image)
	container := corev1.Container{
		Name:            logfwdContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Env:             envs,
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{},
			Limits:   map[corev1.ResourceName]resource.Quantity{},
		},
	}

	// set requests
	cpuRequest, memoryRequest := logfwdResourceRequests()
	if cpuRequest != "" {
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpuRequest)
	}
	if memoryRequest != "" {
		container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memoryRequest)
	}

	// set limits
	cpuLimit, memoryLimit := logfwdResourceLimits()
	if cpuLimit != "" {
		container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLimit)
	}
	if memoryLimit != "" {
		container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(memoryLimit)
	}

	for idx := range volumeNames {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
			ReadOnly:  true,
		})
	}
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name:      "datakit-pod-info",
		MountPath: "/etc/podinfo",
		ReadOnly:  true,
	})

	manager.NewContainerManager(r.pod).AddContainer(&container)
	log.Debugf("logfwd container added: pod=%s, env_count=%d, volume_mounts=%d", r.parent, len(envs), len(volumeNames))
}

func (r *logfwdResource) getVolumePairs(needVolumePaths []string) (volumeNames, volumePaths []string) {
	log.Debugf("logfwd_volume_paths=%s", needVolumePaths)

	volumeManager := manager.NewVolumeManager(r.pod)
	volumeMountManager := manager.NewVolumeMountManager(r.pod)

	for idx := range needVolumePaths {
		exists, name := volumeMountManager.FindVolumeMountPathInContainer(needVolumePaths[idx])

		if exists {
			if volumeManager.IsEmptyDirVolume(name) {
				log.Infof("logfwd_reuse_volume name=%s path=%s pod=%s", name, needVolumePaths[idx], r.parent)
			} else {
				log.Warnf("logfwd_mount_conflict path=%s pod=%s not_emptydir=true", needVolumePaths[idx], r.parent)
				continue
			}
		} else {
			name = fmt.Sprintf("datakit-logfwd-volume-%d", idx)
		}

		volumeNames = append(volumeNames, name)
		volumePaths = append(volumePaths, needVolumePaths[idx])
	}

	return
}
