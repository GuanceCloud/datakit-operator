// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"fmt"
	"testing"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestAdmissionImagePullPolicy(t *testing.T) {
	for _, tc := range []struct {
		ruleset, container string
		want               corev1.PullPolicy
		init               bool
	}{
		{"ddtraces", "datakit-lib-init", corev1.PullIfNotPresent, true},
		{"otels", "datakit-otel-lib-init", corev1.PullNever, true},
		{"logfwds", "datakit-logfwd", corev1.PullIfNotPresent, false},
		{"flameshots", "datakit-flameshot", corev1.PullNever, false},
		{"profilers", "datakit-profiler", corev1.PullIfNotPresent, false},
	} {
		t.Run(tc.ruleset, func(t *testing.T) {
			original := config.Cfg
			config.Cfg = &config.Configuration{}
			t.Cleanup(func() { config.Cfg = original })
			rawConfig := fmt.Sprintf(`{"admission_inject_v2":{"%s":[{
				"namespace_selectors":["default"],"image":"example.com/sdk:1","image_pull_policy":%q,
				"language":"java","processes":"app","envs":{
					"FLAMESHOT_PROFILING_PATH":"/tmp/profiling","FLAMESHOT_HTTP_LOCAL_PORT":"9090"
				}
			}]}}`, tc.ruleset, tc.want)
			if !assert.NoError(t, json.Unmarshal([]byte(rawConfig), config.Cfg)) || !assert.NoError(t, config.Cfg.Setup()) {
				return
			}
			rawPod := []byte(`{
				"apiVersion":"v1","kind":"Pod","metadata":{"name":"app","namespace":"default"},
				"spec":{
					"containers":[{"name":"app","image":"app:1","imagePullPolicy":"Never"}],
					"initContainers":[{"name":"existing","image":"init:1","imagePullPolicy":"IfNotPresent"}]
				}
			}`)
			patchBytes, err := mutateRequest(podAdmissionRequest(rawPod, admissionv1.Create))
			if !assert.NoError(t, err) {
				return
			}
			patch, err := jsonpatch.DecodePatch(patchBytes)
			if !assert.NoError(t, err) {
				return
			}
			mutatedRaw, err := patch.Apply(rawPod)
			if !assert.NoError(t, err) {
				return
			}
			var pod corev1.Pod
			if !assert.NoError(t, json.Unmarshal(mutatedRaw, &pod)) {
				return
			}
			assert.Equal(t, corev1.PullNever, pod.Spec.Containers[0].ImagePullPolicy)
			assert.Equal(t, corev1.PullIfNotPresent, pod.Spec.InitContainers[0].ImagePullPolicy)
			containers := pod.Spec.Containers
			if tc.init {
				containers = pod.Spec.InitContainers
			}
			if !assert.Len(t, containers, 2) {
				return
			}
			assert.Equal(t, tc.container, containers[1].Name)
			assert.Equal(t, tc.want, containers[1].ImagePullPolicy)

			// A different policy on an existing injected container must be preserved.
			containers[1].ImagePullPolicy = corev1.PullAlways
			mutatedRaw, err = json.Marshal(&pod)
			if !assert.NoError(t, err) {
				return
			}
			patchBytes, err = mutateRequest(podAdmissionRequest(mutatedRaw, admissionv1.Create))
			assert.NoError(t, err)
			assert.JSONEq(t, `[]`, string(patchBytes))
		})
	}
}
