package mutator

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	loggingAnnotationKey = "datakit/logs"
)

func MutateLoggingToPod(namespace, parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject ddtrace-lib into nil pod")
	}

	r := newLoggingResource(namespace, parent, pod)
	r.process()
	return nil
}

type loggingResource struct {
	namespace string
	parent    string
	pod       *corev1.Pod
}

func newLoggingResource(namespace, parent string, pod *corev1.Pod) *loggingResource {
	return &loggingResource{
		namespace: namespace,
		parent:    parent,
		pod:       pod,
	}
}

func (r *loggingResource) process() {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	should, configStr := r.shouldInject()
	if !should {
		return
	}

	var cfgs logConfigs
	if err := json.Unmarshal([]byte(configStr), &cfgs); err != nil {
		l.Warnf("logging config '%s' parse failed for Pod %s, err %s", configStr, r.parent, err)
		return
	}

	volumeNames, volumePaths := r.getVolumePairs(parseMountPaths(cfgs))
	r.injectEmptyDirVolume(volumeNames)
	r.injectVolumeMount(volumeNames, volumePaths)

	r.injectAnnotationWithLog(configStr)
}

func (r *loggingResource) shouldInject() (bool, string) {
	if _, exist := r.pod.Annotations[loggingAnnotationKey]; exist {
		return false, ""
	}
	if configStr := loggingMatchNamespaceOrLabelsForConfig(r.namespace, r.pod.Labels); configStr != "" {
		l.Debugf("logging config '%s' found from Pod %s", configStr, r.parent)
		return true, configStr
	}
	return false, ""
}

func (r *loggingResource) injectAnnotationWithLog(configStr string) {
	if r.pod.Annotations == nil {
		r.pod.Annotations = make(map[string]string)
	}
	r.pod.Annotations[loggingAnnotationKey] = configStr
}

func (r *loggingResource) injectEmptyDirVolume(volumeNames []string) {
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

func (r *loggingResource) injectVolumeMount(volumeNames, volumePaths []string) {
	manager := manager.NewVolumeMountManager(r.pod)
	for idx := range volumeNames {
		volumeMount := corev1.VolumeMount{
			Name:      volumeNames[idx],
			MountPath: volumePaths[idx],
		}
		manager.AddVolumeMount(&volumeMount)
	}
}

func (r *loggingResource) getVolumePairs(needVolumePaths []string) (volumeNames, volumePaths []string) {
	l.Debugf("logging volume paths %s", needVolumePaths)

	volumeManager := manager.NewVolumeManager(r.pod)
	volumeMountManager := manager.NewVolumeMountManager(r.pod)

	for idx := range needVolumePaths {
		exists, name := volumeMountManager.FindVolumeMountPathInContainer(needVolumePaths[idx])

		if exists {
			if volumeManager.IsEmptyDirVolume(name) {
				l.Infof("logging reuse volume %s for mountPath %s on %s", name, needVolumePaths[idx], r.parent)
			} else {
				l.Warnf("logging found same mountPath %s from %s, it is not EmptyDir", needVolumePaths[idx], r.parent)
				continue
			}
		} else {
			name = fmt.Sprintf("datakit-logs-volume-%d", idx)
		}

		volumeNames = append(volumeNames, name)
		volumePaths = append(volumePaths, needVolumePaths[idx])
	}

	return
}

type logConfigs []struct {
	Disable bool   `json:"disable"`
	Type    string `json:"type"` // enums "stdout" or "file"
	Path    string `json:"path"`
}

func parseMountPaths(cfgs logConfigs) []string {
	var paths []string
	for _, cfg := range cfgs {
		if cfg.Disable {
			continue
		}
		if cfg.Type != "file" {
			continue
		}

		if starIdx := strings.Index(cfg.Path, "*"); starIdx != -1 {
			paths = append(paths, filepath.Dir(cfg.Path[:starIdx]))
		} else {
			paths = append(paths, filepath.Dir(cfg.Path))
		}
	}
	return unique(paths)
}

func unique(slice []string) []string {
	var res []string
	keys := make(map[string]interface{})
	for _, str := range slice {
		if str == "" {
			continue
		}
		if _, ok := keys[str]; !ok {
			keys[str] = nil
			res = append(res, str)
		}
	}
	return res
}
