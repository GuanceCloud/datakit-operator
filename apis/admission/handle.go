package admission

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mattbaird/jsonpatch"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	jsonContentType = "application/json"
)

var podResource = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

func HandleInject(w http.ResponseWriter, r *http.Request) {
	handle(w, r)
}

func handle(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
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
	var err error

	l.Debugf("request: %s", string(raw))

	switch requ.Resource {
	case podResource:
		var pod corev1.Pod
		err = json.Unmarshal(raw, &pod)
		if err != nil {
			break
		}
		err = mutatePod(getGenerateName(pod.GenerateName), &pod)
		resource = pod

	default:
		return nil, fmt.Errorf("Unsupported resource: %#v", requ.Resource)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to mutate type %#v resource: %v", requ.Resource, err)
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

func getGenerateName(name string) string {
	if len(name) == 0 {
		return "<no-one>"
	}
	if name[len(name)-1] == '-' {
		return name[:len(name)-1]
	}
	return name
}
