package admission

import (
	admissionv1 "k8s.io/api/admission/v1"
	_ "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/admission/injector"
)

type jsonPatch []byte

const (
	injectEnabled = "admission.datakit/enabled"
)

func shouldNotInject(annotations map[string]string) bool {
	s, found := annotations[injectEnabled]
	return found && s == "false"
}

func mutatePod(parent string, pod *corev1.Pod) error {
	if shouldNotInject(pod.GetAnnotations()) {
		return nil
	}

	l.Debug("mutated pod")

	if err := injector.InjectDDTraceToPod(parent, pod); err != nil {
		return err
	}
	if err := injector.InjectLogfwdToPod(parent, pod); err != nil {
		return err
	}
	if err := injector.InjectProfilerToPod(parent, pod); err != nil {
		return err
	}
	return nil
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
