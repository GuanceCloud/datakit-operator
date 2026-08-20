// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetGenerateName(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{
			in:  "",
			out: "<no-one>",
		},
		{
			in:  "deployment",
			out: "deployment",
		},
		{
			in:  "deployment-",
			out: "deployment",
		},
		{
			in:  "deployment-replicaset",
			out: "deployment-replicaset",
		},
	}

	for _, tc := range cases {
		res := getGenerateName(tc.in)
		assert.Equal(t, tc.out, res)
	}

}

func TestMutateRequest_NoPatchWhenPodIsNotMutated(t *testing.T) {
	rawPod := []byte(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "test-pod",
    "namespace": "default",
    "annotations": {
      "admission.datakit/enabled": "false"
    }
  },
  "spec": {
    "containers": [
      {
        "name": "app",
        "image": "nginx:1.25",
        "securityContext": {
          "appArmorProfile": {
            "type": "Unconfined"
          }
        }
      }
    ],
    "initContainers": [
      {
        "name": "istio-init",
        "image": "istio/proxyv2:1.20.0",
        "restartPolicy": "Always"
      }
    ],
    "securityContext": {},
    "nodeName": "worker-1"
  },
  "status": {
    "hostIPs": [{"ip": "10.0.0.1"}],
    "containerStatuses": [
      {
        "name": "app",
        "image": "nginx:1.25",
        "imageID": "",
        "ready": false,
        "restartCount": 0,
        "state": {"waiting": {"reason": "ContainerCreating"}},
        "volumeMounts": [{"name": "kube-api-access", "mountPath": "/var/run/secrets/kubernetes.io/serviceaccount", "recursiveReadOnly": "Disabled"}]
      }
    ]
  }
}`)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Namespace: "default",
		Resource:  podResource,
		Object: runtime.RawExtension{
			Raw: rawPod,
		},
	}

	patchBytes, err := mutateRequest(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, patchBytes)

	var patches []map[string]interface{}
	err = json.Unmarshal(patchBytes, &patches)
	assert.NoError(t, err)
	assert.Empty(t, patches)
}

func TestMutateRequest_UpdateReturnsEmptyPatch(t *testing.T) {
	rawPod := []byte(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "test-pod",
    "namespace": "default"
  },
  "spec": {
    "containers": [{"name": "app", "image": "nginx:1.25"}]
  }
}`)

	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Update,
		Namespace: "default",
		Resource:  podResource,
		Object: runtime.RawExtension{
			Raw: rawPod,
		},
	}

	patchBytes, err := mutateRequest(req)
	assert.NoError(t, err)

	var patches []map[string]interface{}
	err = json.Unmarshal(patchBytes, &patches)
	assert.NoError(t, err)
	assert.Empty(t, patches)
}

func TestBuildPodInjectionPatchOnlyAddsDatakitChanges(t *testing.T) {
	oldPod := &corev1.Pod{}
	oldPod.Name = "test-pod"
	oldPod.Namespace = "default"
	oldPod.Annotations = map[string]string{"existing": "true"}
	oldPod.Spec.Containers = []corev1.Container{
		{
			Name:  "app",
			Image: "busybox:1.36",
			Env: []corev1.EnvVar{
				{Name: "DD_TAGS", Value: "env:dev"},
			},
		},
	}
	oldPod.Spec.InitContainers = []corev1.Container{
		{Name: "istio-init", Image: "istio/proxyv2:1.20.0"},
	}

	newPod := oldPod.DeepCopy()
	newPod.Annotations["datakit/logs"] = `[{"type":"stdout"}]`
	newPod.Spec.InitContainers = append(newPod.Spec.InitContainers, corev1.Container{
		Name:  "datakit-lib-init",
		Image: "pubrepo.guance.com/datakit-operator/dd-lib-java-init:latest",
	})
	newPod.Spec.Volumes = append(newPod.Spec.Volumes, corev1.Volume{Name: "datakit-auto-instrument"})
	newPod.Spec.Containers[0].Env[0].Value = "env:dev,pod_name:test-pod"
	newPod.Spec.Containers[0].Env = append(newPod.Spec.Containers[0].Env, corev1.EnvVar{Name: "DD_AGENT_HOST", Value: "datakit-service.datakit.svc"})
	newPod.Spec.Containers[0].VolumeMounts = append(newPod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
		Name:      "datakit-auto-instrument",
		MountPath: "/datadog-lib",
	})

	patches := buildPodInjectionPatch(oldPod, newPod)
	assert.NotEmpty(t, patches)

	for _, patch := range patches {
		assert.NotEqual(t, "remove", patch.Operation)
		assert.NotContains(t, patch.Path, "/status")
		assert.NotContains(t, patch.Path, "appArmorProfile")
	}
}

func TestMutateRequestInjectsOTelJavaPatch(t *testing.T) {
	originalCfg := config.Cfg
	defer func() {
		config.Cfg = originalCfg
	}()

	var cfg config.Configuration
	err := json.Unmarshal([]byte(`{
  "admission_inject_v2": {
    "otels": [{
      "name": "otel-java",
      "language": "java",
      "namespace_selectors": ["default"],
      "label_selectors": [],
      "check_annotation": false,
      "image": "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0",
      "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "POD_NAMESPACE": "{fieldRef:metadata.namespace}",
        "NODE_NAME": "{fieldRef:spec.nodeName}",
        "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
        "OTEL_RESOURCE_ATTRIBUTES": "k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(POD_NAMESPACE),k8s.node.name=$(NODE_NAME)",
        "OTEL_TRACES_EXPORTER": "otlp",
        "OTEL_LOGS_EXPORTER": "none",
        "OTEL_METRICS_EXPORTER": "none",
        "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
        "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces",
        "OTEL_EXPORTER_OTLP_LOGS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/logs",
        "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/metrics"
      }
    }]
  }
}`), &cfg)
	if !assert.NoError(t, err) || !assert.NoError(t, cfg.Setup()) {
		return
	}
	config.Cfg = &cfg

	rawPod := []byte(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "test-otel-java",
    "namespace": "default",
    "labels": {"app": "checkout"},
    "annotations": {"admission.datakit/ddtrace.enabled": "false"}
  },
  "spec": {
    "nodeName": "worker-1",
    "containers": [
      {"name": "app", "image": "example.com/checkout:1.0.0"},
      {"name": "sidecar", "image": "busybox:1.36"}
    ]
  }
}`)
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Namespace: "default",
		Resource:  podResource,
		Object: runtime.RawExtension{
			Raw: rawPod,
		},
	}

	patchBytes, err := mutateRequest(req)
	if !assert.NoError(t, err) {
		return
	}
	patch, err := jsonpatch.DecodePatch(patchBytes)
	if !assert.NoError(t, err) {
		return
	}
	mutatedRawPod, err := patch.Apply(rawPod)
	if !assert.NoError(t, err) {
		return
	}

	var pod corev1.Pod
	if !assert.NoError(t, json.Unmarshal(mutatedRawPod, &pod)) {
		return
	}
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, "datakit-otel-lib-init", pod.Spec.InitContainers[0].Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.29.0", pod.Spec.InitContainers[0].Image)
		assert.Equal(t, "100m", pod.Spec.InitContainers[0].Resources.Requests.Cpu().String())
		assert.Equal(t, "64Mi", pod.Spec.InitContainers[0].Resources.Requests.Memory().String())
		assert.Equal(t, "500m", pod.Spec.InitContainers[0].Resources.Limits.Cpu().String())
		assert.Equal(t, "512Mi", pod.Spec.InitContainers[0].Resources.Limits.Memory().String())
	}
	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, "datakit-otel-auto-instrument", pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
	}

	expectedEnvNames := []string{
		"JAVA_TOOL_OPTIONS",
		"POD_NAME",
		"POD_NAMESPACE",
		"NODE_NAME",
		"OTEL_SERVICE_NAME",
		"OTEL_RESOURCE_ATTRIBUTES",
		"OTEL_TRACES_EXPORTER",
		"OTEL_LOGS_EXPORTER",
		"OTEL_METRICS_EXPORTER",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
		"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	}
	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, "datakit-otel-auto-instrument", container.VolumeMounts[0].Name)
			assert.Equal(t, "/otel-auto-instrumentation-java", container.VolumeMounts[0].MountPath)
		}
		envNames := make([]string, 0, len(container.Env))
		for _, env := range container.Env {
			envNames = append(envNames, env.Name)
		}
		if !assert.Equal(t, expectedEnvNames, envNames, container.Name) {
			continue
		}
		assert.Equal(t, "-javaagent:/otel-auto-instrumentation-java/javaagent.jar", container.Env[0].Value)
		if assert.NotNil(t, container.Env[1].ValueFrom) && assert.NotNil(t, container.Env[1].ValueFrom.FieldRef) {
			assert.Equal(t, "metadata.name", container.Env[1].ValueFrom.FieldRef.FieldPath)
		}
		assert.Equal(t, "none", container.Env[7].Value)
		assert.Equal(t, "none", container.Env[8].Value)
	}
}
