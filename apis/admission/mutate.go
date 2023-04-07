package admission

import (
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type jsonPatch []byte

func mutateDeployment(deployment *appsv1.Deployment) error {
	l.Debug("mutated deployment")
	return mutatePodTemplate(&deployment.Spec.Template)
}

func mutateDaemonSet(daemonset *appsv1.DaemonSet) error {
	l.Debug("mutated daemonset")
	return mutatePodTemplate(&daemonset.Spec.Template)
}

func mutateCronJob(cronjob *batchv1.CronJob) error {
	l.Debug("mutated cronjob")
	return mutatePodTemplate(&cronjob.Spec.JobTemplate.Spec.Template)
}

func mutateJob(job *batchv1.Job) error {
	l.Debug("mutated job")
	return mutatePodTemplate(&job.Spec.Template)
}

func mutateStatefulSet(statefulSet *appsv1.StatefulSet) error {
	l.Debug("mutated statefulSet")
	return mutatePodTemplate(&statefulSet.Spec.Template)
}

func mutatePodTemplate(podTemplate *corev1.PodTemplateSpec) error {
	l.Debug("mutated podTemplate")
	if err := injectLibToPodTemplate(podTemplate); err != nil {
		return err
	}
	if err := injectLogfwdToPodTemplate(podTemplate); err != nil {
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
