package admission

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
)

const jsonContentType = "application/json"

type admissionFunc func([]byte) ([]byte, error)

func handleInjectLib(w http.ResponseWriter, r *http.Request) {
	handle(w, r, injectLib)
}

func handle(w http.ResponseWriter, r *http.Request, mutateFunc admissionFunc) {
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
	case admissionv1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*admissionv1beta1.AdmissionReview)
		if !ok {
			l.Errorf("Expected v1beta1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &admissionv1beta1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		l.Debugf("request object: %v", requestedAdmissionReview.Request)
		jsonPatch, err := mutateFunc(requestedAdmissionReview.Request.Object.Raw)
		responseAdmissionReview.Response = mutationResponsev1beta1(jsonPatch, err)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview

	case admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
		if !ok {
			l.Errorf("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &admissionv1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		jsonPatch, err := mutateFunc(requestedAdmissionReview.Request.Object.Raw)
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
