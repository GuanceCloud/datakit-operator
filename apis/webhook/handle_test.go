// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"os"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func podAdmissionRequest(raw []byte, operation admissionv1.Operation) *admissionv1.AdmissionRequest {
	return &admissionv1.AdmissionRequest{
		Operation: operation,
		Namespace: "default",
		Resource:  podResource,
		Object:    runtime.RawExtension{Raw: raw},
	}
}

func useWebhookOTelConfig(t *testing.T) {
	t.Helper()
	original := config.Cfg
	config.Cfg = &config.Configuration{
		AdmissionInject: config.AdmissionInjectConfig{
			OTels: config.OTelRules{&config.OTelRule{
				InjectRule: config.InjectRule{
					Name:     "otel-java",
					Selector: config.Selector{Namespaces: []string{"default"}},
					Image:    "example.com/otel-java:1",
				},
				Language: "java",
			}},
		},
	}
	assert.NoError(t, config.Cfg.Setup())
	t.Cleanup(func() { config.Cfg = original })
}

func TestMutateRequestSkipsDisabledCreateAndEnabledUpdate(t *testing.T) {
	useWebhookOTelConfig(t)
	disabled := []byte(`{
  "apiVersion":"v1","kind":"Pod",
  "metadata":{"name":"app","namespace":"default","annotations":{"admission.datakit/enabled":"false"}},
  "spec":{"containers":[{"name":"app","image":"nginx","securityContext":{"appArmorProfile":{"type":"Unconfined"}}}]},
  "status":{"hostIPs":[{"ip":"10.0.0.1"}]}
}`)
	enabled := []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"app","namespace":"default","annotations":{"admission.datakit/ddtrace.enabled":"false"}},"spec":{"containers":[{"name":"app","image":"nginx"}]}}`)

	for _, tc := range []struct {
		name string
		raw  []byte
		op   admissionv1.Operation
	}{{"disabled create", disabled, admissionv1.Create}, {"enabled update", enabled, admissionv1.Update}} {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := mutateRequest(podAdmissionRequest(tc.raw, tc.op))
			assert.NoError(t, err)
			assert.JSONEq(t, `[]`, string(patch))
		})
	}
}

func TestBuildPodInjectionPatchOnlyAddsOwnedFields(t *testing.T) {
	oldPod := &corev1.Pod{}
	oldPod.Annotations = map[string]string{"existing": "true"}
	oldPod.Spec.Containers = []corev1.Container{{
		Name:  "app",
		Image: "busybox",
		Env:   []corev1.EnvVar{{Name: "DD_TAGS", Value: "env:dev"}},
	}}
	oldPod.Spec.InitContainers = []corev1.Container{{Name: "istio-init", Image: "istio"}}

	newPod := oldPod.DeepCopy()
	newPod.Annotations["datakit/logs"] = `[{"type":"stdout"}]`
	newPod.Spec.InitContainers = append(newPod.Spec.InitContainers, corev1.Container{Name: "datakit-lib-init"})
	newPod.Spec.Volumes = []corev1.Volume{{Name: "datakit-auto-instrument"}}
	newPod.Spec.Containers[0].Env[0].Value += ",pod_name:app"
	newPod.Spec.Containers[0].Env = append(newPod.Spec.Containers[0].Env, corev1.EnvVar{Name: "DD_AGENT_HOST"})
	newPod.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{
		Name: "datakit-auto-instrument", MountPath: "/datadog-lib",
	}}

	patches := buildPodInjectionPatch(oldPod, newPod)
	assert.NotEmpty(t, patches)
	operations := map[string]string{}
	for _, patch := range patches {
		operations[patch.Path] = patch.Operation
		assert.NotEqual(t, "remove", patch.Operation)
		assert.NotContains(t, patch.Path, "/status")
		assert.NotContains(t, patch.Path, "appArmorProfile")
	}
	assert.Equal(t, "replace", operations["/spec/containers/0/env/0"])
	assert.Equal(t, "add", operations["/spec/containers/0/env/-"])
}

func TestMutateRequestProducesApplicableOTelPatch(t *testing.T) {
	useWebhookOTelConfig(t)
	raw := []byte(`{
  "apiVersion":"v1","kind":"Pod",
  "metadata":{"name":"app","namespace":"default","annotations":{"admission.datakit/ddtrace.enabled":"false"}},
  "spec":{"containers":[{"name":"app","image":"example.com/app:1","securityContext":{"appArmorProfile":{"type":"Unconfined"}}}]},
  "status":{"hostIPs":[{"ip":"10.0.0.1"}]}
}`)

	patchBytes, err := mutateRequest(podAdmissionRequest(raw, admissionv1.Create))
	if !assert.NoError(t, err) {
		return
	}
	patch, err := jsonpatch.DecodePatch(patchBytes)
	if !assert.NoError(t, err) {
		return
	}
	mutatedRaw, err := patch.Apply(raw)
	if !assert.NoError(t, err) {
		return
	}

	var pod corev1.Pod
	if !assert.NoError(t, json.Unmarshal(mutatedRaw, &pod)) {
		return
	}
	if !assert.Len(t, pod.Spec.Containers, 1) {
		return
	}
	if assert.Len(t, pod.Spec.InitContainers, 1) {
		assert.Equal(t, "datakit-otel-lib-init", pod.Spec.InitContainers[0].Name)
	}
	if assert.Len(t, pod.Spec.Volumes, 1) {
		assert.Equal(t, "datakit-otel-auto-instrument", pod.Spec.Volumes[0].Name)
	}
	if assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1) {
		assert.Equal(t, "/otel-auto-instrumentation-java", pod.Spec.Containers[0].VolumeMounts[0].MountPath)
	}
	if assert.Len(t, pod.Spec.Containers[0].Env, 1) {
		assert.Equal(t, "JAVA_TOOL_OPTIONS", pod.Spec.Containers[0].Env[0].Name)
	}

	assert.Contains(t, string(mutatedRaw), `"appArmorProfile"`)
	assert.Contains(t, string(mutatedRaw), `"hostIPs"`)
}

func TestWebhookDeploymentSourcesEnableReinvocation(t *testing.T) {
	for _, path := range []string{
		"../../charts/datakit-operator/templates/mutatingwebhook.yaml",
		"../../templates/datakit-operator.template.yaml",
	} {
		contents, err := os.ReadFile(path)
		assert.NoError(t, err)
		assert.Contains(t, string(contents), "reinvocationPolicy: IfNeeded")
		assert.NotContains(t, string(contents), "reinvocationPolicy: Never")
	}
}
