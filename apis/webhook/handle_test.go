// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ake-persson/mapslice-json"
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

func TestMutateRequest_ReinvocationRepairsMissingJavaToolOptions(t *testing.T) {
	originalConfig := config.Cfg
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{
			DDTraces: config.DDTraceRules{
				{
					InjectRule: config.InjectRule{
						Selector: config.Selector{Namespaces: []string{"default"}},
						Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
					},
					Language: "java",
				},
			},
		},
	}
	defer func() {
		config.Cfg = originalConfig
	}()
	assert.NoError(t, config.Cfg.Setup())

	rawPod := []byte(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {
    "name": "test-reinvocation",
    "namespace": "default",
    "annotations": {"admission.datakit/ddtrace.enabled": "true"}
  },
  "spec": {
    "containers": [{
      "name": "app",
      "image": "eclipse-temurin:17",
      "volumeMounts": [{"name": "datakit-auto-instrument", "mountPath": "/datadog-lib"}]
    }],
    "initContainers": [{
      "name": "datakit-lib-init",
      "image": "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
      "volumeMounts": [{"name": "datakit-auto-instrument", "mountPath": "/datadog-lib"}]
    }],
    "volumes": [{"name": "datakit-auto-instrument", "emptyDir": {}}]
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
	assert.NotEqual(t, []byte("[]"), patchBytes)

	patch, err := jsonpatch.DecodePatch(patchBytes)
	assert.NoError(t, err)
	mutatedRawPod, err := patch.Apply(rawPod)
	assert.NoError(t, err)

	var mutatedPod corev1.Pod
	assert.NoError(t, json.Unmarshal(mutatedRawPod, &mutatedPod))
	if assert.Len(t, mutatedPod.Spec.Containers, 1) {
		assert.Len(t, mutatedPod.Spec.Containers[0].Env, 1)
		assert.Equal(t, "JAVA_TOOL_OPTIONS", mutatedPod.Spec.Containers[0].Env[0].Name)
		assert.Equal(t, " -javaagent:/datadog-lib/dd-java-agent.jar", mutatedPod.Spec.Containers[0].Env[0].Value)
	}
	assert.Len(t, mutatedPod.Spec.InitContainers, 1)
	assert.Len(t, mutatedPod.Spec.Volumes, 1)
	assert.Len(t, mutatedPod.Spec.Containers[0].VolumeMounts, 1)

	// Kubernetes reinvokes a mutating webhook with the already-mutated object
	// and the same CREATE operation. A further invocation must be a no-op.
	req.Object.Raw = mutatedRawPod
	patchBytes, err = mutateRequest(req)
	assert.NoError(t, err)
	assert.JSONEq(t, `[]`, string(patchBytes))
}

func TestMutateRequest_ReinvocationDoesNotDuplicateAnyInjector(t *testing.T) {
	originalConfig := config.Cfg
	selector := config.Selector{Namespaces: []string{"default"}}
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{
			DDTraces: config.DDTraceRules{
				{
					InjectRule: config.InjectRule{
						Selector: selector,
						Image:    "dd-lib-java-init:test",
					},
					Language: "java",
				},
			},
			Logfwds: config.LogfwdRules{
				{
					InjectRule: config.InjectRule{
						Selector: selector,
						Image:    "logfwd:test",
					},
				},
			},
			Flameshots: config.FlameshotRules{
				{
					InjectRule: config.InjectRule{
						Selector: selector,
						Image:    "flameshot:test",
						Environments: mapslice.MapSlice{
							{Key: "FLAMESHOT_PROFILING_PATH", Value: "/flameshot-data"},
							{Key: "FLAMESHOT_HTTP_LOCAL_PORT", Value: "8089"},
						},
					},
					Processes: `[{"service":"app"}]`,
				},
			},
			Profilers: config.ProfilerRules{
				{
					InjectRule: config.InjectRule{
						Selector: selector,
						Image:    "profiler:test",
					},
					Language: "java",
				},
			},
		},
	}
	defer func() {
		config.Cfg = originalConfig
	}()
	assert.NoError(t, config.Cfg.Setup())

	rawPod := []byte(`{
  "apiVersion": "v1",
  "kind": "Pod",
  "metadata": {"name": "test-all-injectors", "namespace": "default"},
  "spec": {"containers": [{"name": "app", "image": "app:test"}]}
}`)
	req := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Namespace: "default",
		Resource:  podResource,
		Object:    runtime.RawExtension{Raw: rawPod},
	}

	patchBytes, err := mutateRequest(req)
	assert.NoError(t, err)
	patch, err := jsonpatch.DecodePatch(patchBytes)
	assert.NoError(t, err)
	mutatedRawPod, err := patch.Apply(rawPod)
	assert.NoError(t, err)

	var mutatedPod corev1.Pod
	assert.NoError(t, json.Unmarshal(mutatedRawPod, &mutatedPod))
	assert.Len(t, mutatedPod.Spec.InitContainers, 1)
	assert.Len(t, mutatedPod.Spec.Containers, 4)
	containerCounts := map[string]int{}
	for _, container := range mutatedPod.Spec.Containers {
		containerCounts[container.Name]++
	}
	assert.Equal(t, map[string]int{
		"app":               1,
		"datakit-logfwd":    1,
		"datakit-flameshot": 1,
		"datakit-profiler":  1,
	}, containerCounts)

	req.Object.Raw = mutatedRawPod
	patchBytes, err = mutateRequest(req)
	assert.NoError(t, err)
	assert.JSONEq(t, `[]`, string(patchBytes))
}

func TestBuildPodInjectionPatchReplacesOnlyChangedEnvEntry(t *testing.T) {
	oldPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Env: []corev1.EnvVar{
					{Name: "JAVA_TOOL_OPTIONS", Value: "-Xms128m"},
					{Name: "USER_ENV", Value: "keep-me"},
				},
			}},
		},
	}
	newPod := oldPod.DeepCopy()
	newPod.Spec.Containers[0].Env[0].Value += " -javaagent:/datadog-lib/dd-java-agent.jar"

	patches := buildPodInjectionPatch(oldPod, newPod)
	if assert.Len(t, patches, 1) {
		assert.Equal(t, "replace", patches[0].Operation)
		assert.Equal(t, "/spec/containers/0/env/0", patches[0].Path)
	}
}

func TestBuildPodInjectionPatchAppendsNewEnvEntry(t *testing.T) {
	oldPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Env:  []corev1.EnvVar{{Name: "USER_ENV", Value: "keep-me"}},
			}},
		},
	}
	newPod := oldPod.DeepCopy()
	newPod.Spec.Containers[0].Env = append(newPod.Spec.Containers[0].Env, corev1.EnvVar{
		Name:  "JAVA_TOOL_OPTIONS",
		Value: " -javaagent:/datadog-lib/dd-java-agent.jar",
	})

	patches := buildPodInjectionPatch(oldPod, newPod)
	if assert.Len(t, patches, 1) {
		assert.Equal(t, "add", patches[0].Operation)
		assert.Equal(t, "/spec/containers/0/env/-", patches[0].Path)
	}
}

func TestBuildPodInjectionPatchFallsBackWhenEnvOrderChanges(t *testing.T) {
	oldPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "app",
				Env: []corev1.EnvVar{
					{Name: "FIRST", Value: "one"},
					{Name: "SECOND", Value: "two"},
				},
			}},
		},
	}
	newPod := oldPod.DeepCopy()
	newPod.Spec.Containers[0].Env[0], newPod.Spec.Containers[0].Env[1] =
		newPod.Spec.Containers[0].Env[1], newPod.Spec.Containers[0].Env[0]

	patches := buildPodInjectionPatch(oldPod, newPod)
	if assert.Len(t, patches, 1) {
		assert.Equal(t, "replace", patches[0].Operation)
		assert.Equal(t, "/spec/containers/0/env", patches[0].Path)
	}
}

func TestWebhookDeploymentSourcesEnableReinvocation(t *testing.T) {
	paths := []string{
		"../../charts/datakit-operator/templates/mutatingwebhook.yaml",
		"../../templates/datakit-operator.template.yaml",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			contents, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Contains(t, string(contents), "reinvocationPolicy: IfNeeded")
			assert.NotContains(t, string(contents), "reinvocationPolicy: Never")
		})
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

func TestMutateRequestInjectsOTelPythonPatch(t *testing.T) {
	originalCfg := config.Cfg
	defer func() {
		config.Cfg = originalCfg
	}()

	var cfg config.Configuration
	err := json.Unmarshal([]byte(`{
  "admission_inject_v2": {
    "otels": [{
      "name": "otel-python",
      "language": "python",
      "namespace_selectors": ["default"],
      "label_selectors": [],
      "check_annotation": false,
      "image": "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0",
      "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
        "OTEL_TRACES_EXPORTER": "otlp",
        "OTEL_LOGS_EXPORTER": "none",
        "OTEL_METRICS_EXPORTER": "none",
        "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
        "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces"
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
    "name": "test-otel-python",
    "namespace": "default",
    "labels": {"app": "checkout-python"},
    "annotations": {"admission.datakit/ddtrace.enabled": "false"}
  },
  "spec": {
    "containers": [
      {
        "name": "app",
        "image": "python:3.10-slim",
        "env": [{"name": "PYTHONPATH", "value": "/app"}]
      },
      {"name": "sidecar", "image": "python:3.14-slim"}
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
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, "datakit-otel-lib-init", initContainer.Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-python:0.64b0", initContainer.Image)
		assert.Equal(t, []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-python"}, initContainer.Command)
		assert.Equal(t, "100m", initContainer.Resources.Requests.Cpu().String())
		assert.Equal(t, "64Mi", initContainer.Resources.Requests.Memory().String())
		assert.Equal(t, "500m", initContainer.Resources.Limits.Cpu().String())
		assert.Equal(t, "512Mi", initContainer.Resources.Limits.Memory().String())
	}
	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, "datakit-otel-auto-instrument", pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
	}

	expectedPythonPaths := map[string]string{
		"app":     "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/app:/otel-auto-instrumentation-python",
		"sidecar": "/otel-auto-instrumentation-python/opentelemetry/instrumentation/auto_instrumentation:/otel-auto-instrumentation-python",
	}
	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, "datakit-otel-auto-instrument", container.VolumeMounts[0].Name)
			assert.Equal(t, "/otel-auto-instrumentation-python", container.VolumeMounts[0].MountPath)
		}
		envs := make(map[string]corev1.EnvVar, len(container.Env))
		for _, env := range container.Env {
			envs[env.Name] = env
		}
		assert.Equal(t, expectedPythonPaths[container.Name], envs["PYTHONPATH"].Value, container.Name)
		assert.Equal(t, "http/protobuf", envs["OTEL_EXPORTER_OTLP_PROTOCOL"].Value, container.Name)
		assert.Equal(t, "none", envs["OTEL_LOGS_EXPORTER"].Value, container.Name)
		assert.Equal(t, "none", envs["OTEL_METRICS_EXPORTER"].Value, container.Name)
		if assert.NotNil(t, envs["POD_NAME"].ValueFrom) && assert.NotNil(t, envs["POD_NAME"].ValueFrom.FieldRef) {
			assert.Equal(t, "metadata.name", envs["POD_NAME"].ValueFrom.FieldRef.FieldPath)
		}
		if assert.NotNil(t, envs["OTEL_SERVICE_NAME"].ValueFrom) && assert.NotNil(t, envs["OTEL_SERVICE_NAME"].ValueFrom.FieldRef) {
			assert.Equal(t, "metadata.labels['app']", envs["OTEL_SERVICE_NAME"].ValueFrom.FieldRef.FieldPath)
		}
	}
}

func TestMutateRequestInjectsOTelNodeJSPatch(t *testing.T) {
	originalCfg := config.Cfg
	defer func() {
		config.Cfg = originalCfg
	}()

	var cfg config.Configuration
	err := json.Unmarshal([]byte(`{
  "admission_inject_v2": {
    "otels": [{
      "name": "otel-nodejs",
      "language": "nodejs",
      "namespace_selectors": ["default"],
      "label_selectors": [],
      "check_annotation": false,
      "image": "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0",
      "envs": {
        "POD_NAME": "{fieldRef:metadata.name}",
        "OTEL_SERVICE_NAME": "{fieldRef:metadata.labels['app']}",
        "OTEL_TRACES_EXPORTER": "otlp",
        "OTEL_LOGS_EXPORTER": "none",
        "OTEL_METRICS_EXPORTER": "none",
        "OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
        "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://datakit-service.datakit.svc.cluster.local:9529/otel/v1/traces"
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
    "name": "test-otel-nodejs",
    "namespace": "default",
    "labels": {"app": "checkout-nodejs"},
    "annotations": {"admission.datakit/ddtrace.enabled": "false"}
  },
  "spec": {
    "containers": [
      {
        "name": "app",
        "image": "node:20-slim",
        "env": [{"name": "NODE_OPTIONS", "value": "--max-old-space-size=512"}]
      },
      {"name": "sidecar", "image": "node:24-alpine"}
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
		initContainer := pod.Spec.InitContainers[0]
		assert.Equal(t, "datakit-otel-lib-init", initContainer.Name)
		assert.Equal(t, "ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-nodejs:0.78.0", initContainer.Image)
		assert.Equal(t, []string{"cp", "-r", "/autoinstrumentation/.", "/otel-auto-instrumentation-nodejs"}, initContainer.Command)
	}
	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, "datakit-otel-auto-instrument", pod.Spec.Volumes[0].Name)
		assert.NotNil(t, pod.Spec.Volumes[0].EmptyDir)
	}

	expectedNodeOptions := map[string]string{
		"app":     "--max-old-space-size=512 --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
		"sidecar": " --require /otel-auto-instrumentation-nodejs/autoinstrumentation.js",
	}
	for _, container := range pod.Spec.Containers {
		if assert.Len(t, container.VolumeMounts, 1, container.Name) {
			assert.Equal(t, "datakit-otel-auto-instrument", container.VolumeMounts[0].Name)
			assert.Equal(t, "/otel-auto-instrumentation-nodejs", container.VolumeMounts[0].MountPath)
		}
		envs := make(map[string]corev1.EnvVar, len(container.Env))
		for _, env := range container.Env {
			envs[env.Name] = env
		}
		assert.Equal(t, expectedNodeOptions[container.Name], envs["NODE_OPTIONS"].Value, container.Name)
		assert.Equal(t, "http/protobuf", envs["OTEL_EXPORTER_OTLP_PROTOCOL"].Value, container.Name)
		assert.Equal(t, "none", envs["OTEL_LOGS_EXPORTER"].Value, container.Name)
		assert.Equal(t, "none", envs["OTEL_METRICS_EXPORTER"].Value, container.Name)
		if assert.NotNil(t, envs["POD_NAME"].ValueFrom) && assert.NotNil(t, envs["POD_NAME"].ValueFrom.FieldRef) {
			assert.Equal(t, "metadata.name", envs["POD_NAME"].ValueFrom.FieldRef.FieldPath)
		}
	}

	req.Object.Raw = mutatedRawPod
	secondPatchBytes, err := mutateRequest(req)
	assert.NoError(t, err)
	var secondPatch []map[string]interface{}
	assert.NoError(t, json.Unmarshal(secondPatchBytes, &secondPatch))
	assert.Empty(t, secondPatch)
}
