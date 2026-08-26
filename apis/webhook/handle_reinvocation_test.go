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
)

func useWebhookDDTraceConfig(t *testing.T) {
	t.Helper()
	original := config.Cfg
	config.Cfg = &config.Configuration{AdmissionInject: config.AdmissionInjectConfig{
		DDTraces: config.DDTraceRules{&config.DDTraceRule{
			InjectRule: config.InjectRule{
				Selector: config.Selector{Namespaces: []string{"default"}},
				Image:    "example.com/dd-java:1",
			},
			Language: "java",
		}},
	}}
	if err := config.Cfg.Setup(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.Cfg = original })
}

func TestMutateRequestRepairsDDTraceReinvocationWithNarrowPatch(t *testing.T) {
	useWebhookDDTraceConfig(t)
	raw := []byte(`{
  "apiVersion":"v1","kind":"Pod","metadata":{"name":"reinvoked","namespace":"default"},
  "spec":{
    "containers":[{"name":"app","image":"java:17","volumeMounts":[{"name":"datakit-auto-instrument","mountPath":"/datadog-lib"}]}],
    "initContainers":[{"name":"datakit-lib-init","image":"example.com/dd-java:1","volumeMounts":[{"name":"datakit-auto-instrument","mountPath":"/datadog-lib"}]}],
    "volumes":[{"name":"datakit-auto-instrument","emptyDir":{}}]
  }
}`)
	req := podAdmissionRequest(raw, admissionv1.Create)

	patchBytes, err := mutateRequest(req)
	if !assert.NoError(t, err) {
		return
	}
	var operations []struct {
		Operation string `json:"op"`
		Path      string `json:"path"`
	}
	if !assert.NoError(t, json.Unmarshal(patchBytes, &operations)) || !assert.Len(t, operations, 1) {
		return
	}
	assert.Equal(t, "add", operations[0].Operation)
	assert.Equal(t, "/spec/containers/0/env", operations[0].Path)

	patch, err := jsonpatch.DecodePatch(patchBytes)
	if !assert.NoError(t, err) {
		return
	}
	mutatedRaw, err := patch.Apply(raw)
	if !assert.NoError(t, err) {
		return
	}
	var pod corev1.Pod
	if !assert.NoError(t, json.Unmarshal(mutatedRaw, &pod)) || !assert.Len(t, pod.Spec.Containers, 1) {
		return
	}
	if !assert.Len(t, pod.Spec.Containers[0].Env, 1) {
		return
	}
	assert.Equal(t, "JAVA_TOOL_OPTIONS", pod.Spec.Containers[0].Env[0].Name)

	req.Object.Raw = mutatedRaw
	patchBytes, err = mutateRequest(req)
	if !assert.NoError(t, err) {
		return
	}
	assert.JSONEq(t, `[]`, string(patchBytes))
}

func TestMutateRequestRepairsOnlyEligibleDDTraceContainer(t *testing.T) {
	useWebhookDDTraceConfig(t)
	raw := []byte(`{
  "apiVersion":"v1","kind":"Pod","metadata":{"name":"reinvoked","namespace":"default"},
  "spec":{
    "containers":[
      {"name":"value-from","image":"java:17","env":[{"name":"JAVA_TOOL_OPTIONS","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}}],"volumeMounts":[{"name":"datakit-auto-instrument","mountPath":"/datadog-lib"}]},
      {"name":"missing-env","image":"java:17","volumeMounts":[{"name":"datakit-auto-instrument","mountPath":"/datadog-lib"}]}
    ],
    "initContainers":[{"name":"datakit-lib-init","image":"example.com/dd-java:1","volumeMounts":[{"name":"datakit-auto-instrument","mountPath":"/datadog-lib"}]}],
    "volumes":[{"name":"datakit-auto-instrument","emptyDir":{}}]
  }
}`)
	req := podAdmissionRequest(raw, admissionv1.Create)

	patchBytes, err := mutateRequest(req)
	if !assert.NoError(t, err) {
		return
	}
	var operations []struct {
		Operation string `json:"op"`
		Path      string `json:"path"`
	}
	if !assert.NoError(t, json.Unmarshal(patchBytes, &operations)) || !assert.Len(t, operations, 1) {
		return
	}
	assert.Equal(t, "add", operations[0].Operation)
	assert.Equal(t, "/spec/containers/1/env", operations[0].Path)

	patch, err := jsonpatch.DecodePatch(patchBytes)
	if !assert.NoError(t, err) {
		return
	}
	mutatedRaw, err := patch.Apply(raw)
	if !assert.NoError(t, err) {
		return
	}
	var pod corev1.Pod
	if !assert.NoError(t, json.Unmarshal(mutatedRaw, &pod)) || !assert.Len(t, pod.Spec.Containers, 2) {
		return
	}
	assert.NotNil(t, pod.Spec.Containers[0].Env[0].ValueFrom)
	if assert.Len(t, pod.Spec.Containers[1].Env, 1) {
		assert.Equal(t, "JAVA_TOOL_OPTIONS", pod.Spec.Containers[1].Env[0].Name)
	}

	req.Object.Raw = mutatedRaw
	patchBytes, err = mutateRequest(req)
	if !assert.NoError(t, err) {
		return
	}
	assert.JSONEq(t, `[]`, string(patchBytes))
}
