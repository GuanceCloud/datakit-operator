// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	logfwdContainerName = "datakit-logfwd"

	logfwdEnabledAnnotationKey   = "admission.datakit/logfwd.enabled"
	logfwdInstancesAnnotationKey = "admission.datakit/logfwd.instances"

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
	should, rule := r.getMatchingRule()
	if !should || rule == nil {
		return
	}

	log.Infof("logfwd injection started: pod=%s, namespace=%s", r.parent, r.pod.Namespace)

	// Then create a logfwd container, the container needs to be ReadOnly.
	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)

	// 提取配置：instances（Annotation）或 log_configs（rule）
	instancesConfig, instancesVolumePaths, hasInstancesConfig := r.extractInstancesConfig()
	logConfigsConfig, logConfigsVolumePaths, hasLogConfigsConfig := r.extractLogConfigsConfig(rule)

	// 即使没有 instances 和 log_configs 也要注入，因为 logfwd 还可以通过网络获取配置

	// 设置环境变量
	if hasInstancesConfig {
		envs = append(envs, corev1.EnvVar{Name: logfwdJSONConfigKey, Value: instancesConfig})
	}
	if hasLogConfigsConfig {
		envs = append(envs, corev1.EnvVar{Name: logfwdLogConfigsKey, Value: logConfigsConfig})
	}

	// 收集所有 volume paths
	allVolumePaths := append(instancesVolumePaths, logConfigsVolumePaths...)
	if len(rule.LogVolumePaths) > 0 {
		rulePaths := getMountPaths(rule.LogVolumePaths)
		allVolumePaths = append(allVolumePaths, rulePaths...)
	}

	volumeNames, volumeMountPaths := r.getVolumePairs(unique(allVolumePaths))
	log.Debugf("logfwd volumes prepared: pod=%s, volumes=%d, paths=%d", r.parent, len(volumeNames), len(allVolumePaths))

	r.injectVolume(volumeNames)
	r.injectVolumeMount(volumeNames, volumeMountPaths)

	r.injectContainer(rule.Image, envs, volumeNames, volumeMountPaths, rule.Resources)

	log.Infof("logfwd injection completed: pod=%s, image=%s", r.parent, rule.Image)
}

func (r *logfwdResource) getMatchingRule() (bool, *config.InjectRule) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), logfwdEnabledAnnotationKey) {
		log.Debugf("logfwd annotation disabled: pod=%s", r.parent)
		return false, nil
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(logfwdContainerName) {
		log.Debugf("logfwd container already exists: pod=%s", r.parent)
		return false, nil
	}

	matched, rule := logfwdMatchNamespaceOrLabelsForConfig(r.pod.Namespace, r.pod.GetLabels())
	if !matched || rule == nil {
		log.Debugf("logfwd no matching rule: pod=%s, namespace=%s", r.parent, r.pod.Namespace)
		return false, nil
	}

	// 兼容旧版配置：如果 rule.Legacy 为 true，需要验证是否存在 logfwd.instances Annotation
	if rule.Legacy {
		annotations := r.pod.GetAnnotations()
		if _, exists := annotations[logfwdInstancesAnnotationKey]; !exists {
			log.Debugf("logfwd legacy rule requires logfwd.instances annotation: pod=%s", r.parent)
			return false, nil
		}
	}

	return true, rule
}

// extractInstancesConfig 从 Annotation 中提取 instances config（兼容旧版）
func (r *logfwdResource) extractInstancesConfig() (string, []string, bool) {
	annotations := r.pod.GetAnnotations()
	configStr, found := annotations[logfwdInstancesAnnotationKey]
	if !found {
		return "", nil, false
	}

	// 如果注解存在但值为空，视为有效配置但内容为空
	if configStr == "" {
		return configStr, []string{}, true
	}

	var configBuff bytes.Buffer
	if err := json.Compact(&configBuff, []byte(configStr)); err != nil {
		log.Warnf("logfwd instances config invalid: pod=%s, error=%v", r.parent, err)
		return "", nil, false
	}

	// 处理 instances 配置，提取路径
	var configs []*logfwdConfig
	if err := json.Unmarshal(configBuff.Bytes(), &configs); err != nil {
		log.Warnf("logfwd instances config parse failed: pod=%s, error=%v", r.parent, err)
		return "", nil, false
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

	return configBuff.String(), unique(paths), true
}

// extractLogConfigsConfig 从 rule.LogConfigs 提取 log_configs config
func (r *logfwdResource) extractLogConfigsConfig(rule *config.InjectRule) (string, []string, bool) {
	if rule == nil || rule.LogConfigs == "" {
		return "", nil, false
	}

	configStr := rule.LogConfigs
	var configBuff bytes.Buffer
	if err := json.Compact(&configBuff, []byte(configStr)); err != nil {
		log.Warnf("logfwd log_configs config invalid: pod=%s, error=%v", r.parent, err)
		return "", nil, false
	}

	// 处理 log_configs 配置，提取路径
	var configs []*logConfig
	if err := json.Unmarshal(configBuff.Bytes(), &configs); err != nil {
		log.Warnf("logfwd log_configs config parse failed: pod=%s, error=%v", r.parent, err)
		return "", nil, false
	}

	var paths []string
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		paths = append(paths, getMountPath(cfg.Path))
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

	// pod-info volume 必须始终创建，用于将 Pod labels 注入到容器中
	// labels 将被挂载到 /etc/podinfo/labels，供 logfwd 识别和标记日志
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
}

func (r *logfwdResource) injectContainer(image string, envs []corev1.EnvVar, volumeNames, volumePaths []string, resources config.ResourceRequirements) {
	container := corev1.Container{
		Name:            logfwdContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Env:             envs,
	}

	setContainerResources(&container, resources.Requests.CPU, resources.Requests.Memory, resources.Limits.CPU, resources.Limits.Memory)

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
	log.Debugf("logfwd container created: pod=%s, envs=%d, volume_mounts=%d", r.parent, len(envs), len(container.VolumeMounts))
}

func (r *logfwdResource) getVolumePairs(needVolumePaths []string) (volumeNames, volumePaths []string) {
	volumeManager := manager.NewVolumeManager(r.pod)
	volumeMountManager := manager.NewVolumeMountManager(r.pod)

	for idx := range needVolumePaths {
		exists, name := volumeMountManager.FindVolumeMountPathInContainer(needVolumePaths[idx])

		if exists {
			if !volumeManager.IsEmptyDirVolume(name) {
				log.Warnf("logfwd mount conflict: path=%s pod=%s", needVolumePaths[idx], r.parent)
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
