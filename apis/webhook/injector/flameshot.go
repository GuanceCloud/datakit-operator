// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

const (
	flameshotContainerName        = "datakit-flameshot"
	flameshotEnabledAnnotationKey = "admission.datakit/flameshot.enabled"

	flameshotProfilingVolumeName = "datakit-flameshot-volume"
	flameshotProfilingPathKey    = "FLAMESHOT_PROFILING_PATH"

	flameshotHTTPPortName        = "datakit-flameshot-http-port"
	flameshotHTTPLocalAddressKey = "FLAMESHOT_HTTP_LOCAL_ADDR"
	flameshotProcessesKey        = "FLAMESHOT_PROCESSES"
)

func InjectFlameshotToPod(namespace, parent string, pod *corev1.Pod) error {
	if pod == nil {
		return fmt.Errorf("cannot inject flameshot into nil pod")
	}

	r := newFlameshotResource(namespace, parent, pod)
	r.process()
	return nil
}

type flameshotResource struct {
	namespace string
	parent    string
	pod       *corev1.Pod
}

func newFlameshotResource(namespace, parent string, pod *corev1.Pod) *flameshotResource {
	return &flameshotResource{
		namespace: namespace,
		parent:    parent,
		pod:       pod,
	}
}

func (r *flameshotResource) process() {
	if r.pod.Namespace != "" {
		r.namespace = r.pod.Namespace
	}

	should, rule := r.getMatchingRule()
	if !should || rule == nil {
		return
	}

	// 如果 Processes 为空，跳过注入
	if rule.Processes == "" {
		log.Warnf("flameshot injection skipped: pod=%s, reason=flameshot_processes_empty", r.parent)
		return
	}

	log.Infof("flameshot injection started: pod=%s, namespace=%s", r.parent, r.namespace)

	envs := envbuilder.BuildEnvs(rule.Envs, enableEnvFieldRef)

	// 添加 FLAMESHOT_PROCESSES 环境变量（如果已存在会被后面的值覆盖）
	envs = append(envs, corev1.EnvVar{Name: flameshotProcessesKey, Value: rule.Processes})

	profilingPath := getFlameshotProfilingPath(envs)
	if profilingPath == "" {
		log.Warnf("flameshot missing required env: key=%s pod=%s", flameshotProfilingPathKey, r.parent)
		return
	}

	port := getFlameshotPort(envs)
	if port == 0 {
		log.Warnf("flameshot missing required env: key=%s pod=%s", flameshotHTTPLocalAddressKey, r.parent)
		return
	}

	r.resetSpec()
	r.injectContainer(rule.Image, envs, port, rule.Resources)
	r.injectVolume()
	r.injectVolumeMount(profilingPath)
	log.Debugf("flameshot container created: pod=%s, image=%s, port=%d", r.parent, rule.Image, port)

	log.Infof("flameshot injection completed: pod=%s, image=%s", r.parent, rule.Image)
}

func (r *flameshotResource) getMatchingRule() (bool, *config.InjectRule) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), flameshotEnabledAnnotationKey) {
		log.Debugf("flameshot annotation disabled: pod=%s", r.parent)
		return false, nil
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(flameshotContainerName) {
		log.Debugf("flameshot container already exists: pod=%s", r.parent)
		return false, nil
	}

	matched, rule := flameshotMatchNamespaceOrLabelsForConfig(r.namespace, r.pod.GetLabels())
	if matched && rule != nil {
		log.Infof("flameshot rule matched: pod=%s, image=%s", r.parent, rule.Image)
	} else {
		log.Debugf("flameshot no matching rule: pod=%s, namespace=%s", r.parent, r.namespace)
	}
	return matched, rule
}

func (r *flameshotResource) resetSpec() {
	var b = true
	r.pod.Spec.ShareProcessNamespace = &b
	r.pod.Spec.RestartPolicy = corev1.RestartPolicyAlways
}

func (r *flameshotResource) injectVolume() {
	profilingDir := corev1.Volume{
		Name: flameshotProfilingVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	manager := manager.NewVolumeManager(r.pod)
	manager.AddVolume(&profilingDir)
}

func (r *flameshotResource) injectVolumeMount(path string) {
	profilingDir := corev1.VolumeMount{
		Name:      flameshotProfilingVolumeName,
		MountPath: path,
	}

	manager := manager.NewVolumeMountManager(r.pod)
	manager.AddVolumeMount(&profilingDir)
}

func (r *flameshotResource) injectContainer(image string, envs []corev1.EnvVar, port int32, resources config.ResourceRequirements) {
	container := corev1.Container{
		Name:            flameshotContainerName,
		Image:           image,
		Command:         []string{"/flameshot/flameshot"},
		ImagePullPolicy: corev1.PullAlways,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"SYS_PTRACE"},
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          flameshotHTTPPortName,
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	setContainerResources(&container, resources.Requests.CPU, resources.Requests.Memory, resources.Limits.CPU, resources.Limits.Memory)

	container.Env = append(container.Env, envs...)
	manager.NewContainerManager(r.pod).AddContainer(&container)
}

func getFlameshotProfilingPath(envs []corev1.EnvVar) string {
	for _, env := range envs {
		if env.Name == flameshotProfilingPathKey {
			return filepath.Clean(env.Value)
		}
	}
	return ""
}

func getFlameshotPort(envs []corev1.EnvVar) int32 {
	for _, env := range envs {
		if env.Name == flameshotHTTPLocalAddressKey {
			parsedPort, err := parsePortFromAddress(env.Value)
			if err == nil {
				return parsedPort
			}
		}
	}
	return 0
}

func parsePortFromAddress(address string) (int32, error) {
	_, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return 0, err
	}

	port, err := strconv.ParseInt(portStr, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(port), nil
}
