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
