// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package webhook

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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

func HandleInject(c *gin.Context) {
	// verify the content type is accurate
	contentType := c.GetHeader("Content-Type")
	if contentType != jsonContentType {
		log.Errorf("content_type=%s expect %s", contentType, jsonContentType)
		c.Status(http.StatusBadRequest)
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		c.Status(http.StatusBadRequest)
		return
	}

	log.Debugf("request_body=%s", body)

	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		log.Error(msg)
		c.String(http.StatusBadRequest, msg)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case admissionv1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*admissionv1.AdmissionReview)
		if !ok {
			log.Errorf("expect=AdmissionReview got=%T", obj)
			c.Status(http.StatusBadRequest)
			return
		}
		responseAdmissionReview := &admissionv1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)

		jsonPatch, err := mutateRequest(requestedAdmissionReview.Request)
		responseAdmissionReview.Response = mutationResponsev1(jsonPatch, err)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview

	default:
		msg := fmt.Sprintf("unsupported_gvk=%v", gvk)
		log.Error(msg)
		c.String(http.StatusBadRequest, msg)
		return
	}

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		log.Error(err)
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	log.Debugf("response_body=%s", respBytes)

	c.Data(http.StatusOK, jsonContentType, respBytes)
}

func mutateRequest(requ *admissionv1.AdmissionRequest) (jsonPatch, error) {
	var raw = requ.Object.Raw
	var resource interface{}
	var err error

	log.Debugf("request=%s", string(raw))

	switch requ.Resource {
	case podResource:
		var pod corev1.Pod
		err = json.Unmarshal(raw, &pod)
		if err != nil {
			break
		}
		err = mutatePod(requ.Namespace, getGenerateName(pod.GenerateName), &pod)
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

	log.Debugf("response=%s", string(newRaw))

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
