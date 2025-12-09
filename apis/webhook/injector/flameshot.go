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

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	flameshotContainerName = "datakit-flameshot"

	flameshotProfilingVolumeName = "datakit-flameshot-volume"
	flameshotProfilingPathKey    = "FLAMESHOT_PROFILING_PATH"

	flameshotHTTPPortName        = "datakit-flameshot-http-port"
	flameshotHTTPLocalAddressKey = "FLAMESHOT_HTTP_LOCAL_ADDRESS"

	flameshotEnabledAnnotationKey   = "admission.datakit/flameshot.enabled"
	flameshotProcessesAnnotationKey = "admission.datakit/flameshot.processes"
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

	should, processes := r.shouldInject()
	if !should || processes == "" {
		return
	}

	image := flameshotImage()
	log.Infof("flameshot use_image=%s pod=%s", image, r.parent)

	envs, profilingPath, port := buildFlameshotEnvs()
	if profilingPath == "" {
		log.Warnf("flameshot missing required env key=%s pod=%s", flameshotProfilingPathKey, r.parent)
		return
	}
	if port == 0 {
		log.Warnf("flameshot missing required env key=%s pod=%s", flameshotHTTPLocalAddressKey, r.parent)
		return
	}
	log.Infof("flameshot profilingPath=%s port=%d pod=%s", profilingPath, port, r.parent)

	r.resetSpec()
	r.injectContainer(image, envs, port)
	r.injectVolume()
	r.injectVolumeMount(profilingPath)
}

func (r *flameshotResource) shouldInject() (bool, string) {
	if !CheckAnnotationIsTrue(r.pod.GetAnnotations(), flameshotEnabledAnnotationKey) {
		return false, ""
	}

	if manager.NewContainerManager(r.pod).ContainsContainer(flameshotContainerName) {
		return false, ""
	}

	annotations := r.pod.GetAnnotations()
	if processes, found := annotations[flameshotProcessesAnnotationKey]; found {
		if processes == "" {
			log.Warnf("flameshot processes is empty for pod=%s", r.parent)
		} else {
			log.Debugf("flameshot_find_annotation processes=%s pod=%s", processes, r.parent)
			return true, processes
		}
	}

	return false, ""
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

func (r *flameshotResource) injectContainer(image string, envs []corev1.EnvVar, port int32) {
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
		Resources: corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{},
			Limits:   map[corev1.ResourceName]resource.Quantity{},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          flameshotHTTPPortName,
				ContainerPort: port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
	}

	// set requests
	cpuRequest, memoryRequest := flameshotResourceRequests()
	if cpuRequest != "" {
		container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpuRequest)
	}
	if memoryRequest != "" {
		container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memoryRequest)
	}

	// set limits
	cpuLimit, memoryLimit := flameshotResourceLimits()
	if cpuLimit != "" {
		container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpuLimit)
	}
	if memoryLimit != "" {
		container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(memoryLimit)
	}

	container.Env = append(container.Env, envs...)

	manager.NewContainerManager(r.pod).AddContainer(&container)
}

func buildFlameshotEnvs() (envs []corev1.EnvVar, path string, port int32) {
	envs = envbuilder.BuildEnvs(flameshotEnvs(), enableEnvFieldRef)

	var profilingPath string
	var httpPort int32

	for _, env := range envs {
		if env.Name == flameshotProfilingPathKey {
			profilingPath = filepath.Clean(env.Value)
		}
		if env.Name == flameshotHTTPLocalAddressKey {
			parsedPort, err := parsePortFromAddress(env.Value)
			if err == nil {
				httpPort = parsedPort
			}
		}
	}

	return envs, profilingPath, httpPort
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
