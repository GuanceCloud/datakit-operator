// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	admissionv1 "k8s.io/api/admission/v1"
	_ "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook/injector"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook/mutator"
)

type jsonPatch []byte

func shouldInject(annotations map[string]string) bool {
	const injectEnabled = "admission.datakit/enabled"
	return injector.CheckAnnotationIsTrue(annotations, injectEnabled)
}

func mutatePod(namespace, parent string, pod *corev1.Pod) (bool, error) {
	if !shouldInject(pod.GetAnnotations()) {
		return false, nil
	}

	log.Debug("mutated pod")

	changed := false
	if ok, err := injector.InjectDDTraceToPod(namespace, parent, pod); err != nil {
		return changed, err
	} else {
		changed = changed || ok
	}
	if ok, err := injector.InjectLogfwdToPod(namespace, parent, pod); err != nil {
		return changed, err
	} else {
		changed = changed || ok
	}
	if ok, err := injector.InjectFlameshotToPod(namespace, parent, pod); err != nil {
		return changed, err
	} else {
		changed = changed || ok
	}
	if ok, err := injector.InjectProfilerToPod(namespace, parent, pod); err != nil {
		return changed, err
	} else {
		changed = changed || ok
	}
	if ok, err := mutator.MutateLoggingToPod(namespace, parent, pod); err != nil {
		return changed, err
	} else {
		changed = changed || ok
	}

	return changed, nil
}

func mutationResponsev1(jsonPatch []byte, err error) *admissionv1.AdmissionResponse {
	if err != nil {
		log.Warnf("mutate_failed=%v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
			Allowed: true,
		}
	}
	log.Debugf("json_patch=%s", jsonPatch)
	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Patch:     jsonPatch,
		PatchType: &patchType,
		Allowed:   true,
	}
}
