package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mattbaird/jsonpatch"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	jsonContentType = "application/json"
)

var (
	deploymentResource  = metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	daemonsetResource   = metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	cronjobResource     = metav1.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"}
	jobResource         = metav1.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	statefulsetResource = metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
)

func HandleInject(w http.ResponseWriter, r *http.Request) {
	handle(w, r)
}

func handle(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != jsonContentType {
		l.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	l.Debugf("handling request: %s", body)

	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		l.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
		if !ok {
			l.Errorf("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &admissionv1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)

		jsonPatch, err := mutateRequest(requestedAdmissionReview.Request)
		responseAdmissionReview.Response = mutationResponsev1(jsonPatch, err)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview

	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		l.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		l.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	l.Debugf("sending response: %s", respBytes)

	w.Header().Set("Content-Type", jsonContentType)
	if _, err := w.Write(respBytes); err != nil {
		l.Error(err)
	}
}

func mutateRequest(requ *admissionv1.AdmissionRequest) (jsonPatch, error) {
	var raw = requ.Object.Raw
	var resource interface{}

	l.Debugf("request: %s", string(raw))

	switch requ.Resource {
	case deploymentResource:
		var deployment appsv1.Deployment
		if err := json.Unmarshal(raw, &deployment); err != nil {
			return nil, fmt.Errorf("failed to decode raw deployment: %w", err)
		}
		mutateDeployment(&deployment)
		resource = deployment

	case daemonsetResource:
		var daemonset appsv1.DaemonSet
		if err := json.Unmarshal(raw, &daemonset); err != nil {
			return nil, fmt.Errorf("failed to decode raw daemonset: %w", err)
		}
		mutateDaemonSet(&daemonset)
		resource = daemonset

	case cronjobResource:
		var cronjob batchv1.CronJob
		if err := json.Unmarshal(raw, &cronjob); err != nil {
			return nil, fmt.Errorf("failed to decode raw cronjob: %w", err)
		}
		mutateCronJob(&cronjob)
		resource = cronjob

	case jobResource:
		var job batchv1.Job
		if err := json.Unmarshal(raw, &job); err != nil {
			return nil, fmt.Errorf("failed to decode raw job: %w", err)
		}
		mutateJob(&job)
		resource = job

	case statefulsetResource:
		var statefulset appsv1.StatefulSet
		if err := json.Unmarshal(raw, &statefulset); err != nil {
			return nil, fmt.Errorf("failed to decode raw statefulset: %w", err)
		}
		mutateStatefulSet(&statefulset)
		resource = statefulset

	default:
		return nil, fmt.Errorf("Unsupported resource: %#v", requ.Resource)
	}

	newRaw, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to encode the mutated object: %v", err)
	}

	l.Debugf("response: %s", string(newRaw))

	patchs, err := jsonpatch.CreatePatch(raw, newRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the JSON patch: %w", err)
	}

	return json.Marshal(patchs)
}
