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
