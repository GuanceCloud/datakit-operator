package admission

import (
	"encoding/json"
	"fmt"

	"github.com/mattbaird/jsonpatch"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mutateFunc func(*corev1.Pod) error

func mutatePod(rawPod []byte, fn mutateFunc) ([]byte, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(rawPod, &pod); err != nil {
		return nil, fmt.Errorf("failed to decode raw object: %w", err)
	}

	if err := fn(&pod); err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("failed to encode the mutated Pod object: %v", err)
	}

	l.Debugf("mutating pod: %s", bytes)

	patchs, err := jsonpatch.CreatePatch(rawPod, bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the JSON patch: %w", err)
	}

	return json.Marshal(patchs)
}

func mutationResponsev1(jsonPatch []byte, err error) *admissionv1.AdmissionResponse {
	if err != nil {
		l.Warnf("Failed to v1.mutate: %v", err)
		return &admissionv1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
			Allowed: true,
		}
	}
	l.Debugf("jsonpatch: %s", jsonPatch)
	patchType := admissionv1.PatchTypeJSONPatch
	return &admissionv1.AdmissionResponse{
		Patch:     jsonPatch,
		PatchType: &patchType,
		Allowed:   true,
	}
}
