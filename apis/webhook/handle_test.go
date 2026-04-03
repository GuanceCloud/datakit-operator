// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
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

func TestMutateRequest_PreservesInitContainerRestartPolicy(t *testing.T) {
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
        "image": "nginx:1.25"
      }
    ],
    "initContainers": [
      {
        "name": "istio-init",
        "image": "istio/proxyv2:1.20.0",
        "restartPolicy": "Always"
      }
    ]
  }
}`)

	req := &admissionv1.AdmissionRequest{
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

	found := false
	for _, p := range patches {
		if p["op"] == "remove" && p["path"] == "/spec/initContainers/0/restartPolicy" {
			found = true
			break
		}
	}
	assert.False(t, found, "restartPolicy should not be removed in generated patch")
}

func TestPreserveInitContainerRestartPolicy_DoesNotSetForInjectedInitContainer(t *testing.T) {
	oldRaw := []byte(`{
  "apiVersion":"v1",
  "kind":"Pod",
  "spec":{
    "containers":[{"name":"app","image":"busybox:1.36"}],
    "initContainers":[{"name":"istio-init","image":"busybox:1.36","restartPolicy":"Always"}]
  }
}`)

	newRaw := []byte(`{
  "apiVersion":"v1",
  "kind":"Pod",
  "spec":{
    "containers":[{"name":"app","image":"busybox:1.36"}],
    "initContainers":[
      {"name":"istio-init","image":"busybox:1.36"},
      {"name":"datakit-lib-init","image":"pubrepo.jiagouyun.com/datakit-operator/dd-lib-python-init:latest"}
    ]
  }
}`)

	got, err := preserveInitContainerRestartPolicy(oldRaw, newRaw)
	assert.NoError(t, err)

	var pod map[string]interface{}
	err = json.Unmarshal(got, &pod)
	assert.NoError(t, err)

	spec, ok := pod["spec"].(map[string]interface{})
	assert.True(t, ok)
	initContainers, ok := spec["initContainers"].([]interface{})
	assert.True(t, ok)

	policies := map[string]string{}
	for _, c := range initContainers {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		policy, _ := m["restartPolicy"].(string)
		policies[name] = policy
	}

	assert.Equal(t, "Always", policies["istio-init"])
	assert.Equal(t, "", policies["datakit-lib-init"])
}
