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

	log.Debugf("request=%s", string(raw))

	switch requ.Resource {
	case podResource:
		var pod corev1.Pod
		if err := json.Unmarshal(raw, &pod); err != nil {
			return nil, fmt.Errorf("failed to decode pod object: %v", err)
		}

		podName := pod.Name
		if podName == "" {
			podName = pod.GenerateName
		}
		log.Infof("received pod request: namespace=%s, name=%s", requ.Namespace, podName)

		if requ.Operation != admissionv1.Create {
			return marshalPatch(requ, nil)
		}

		mutatedPod := pod.DeepCopy()
		changed, err := mutatePod(requ.Namespace, getGenerateName(mutatedPod.GenerateName), mutatedPod)
		if err != nil {
			return nil, fmt.Errorf("failed to mutate type %#v resource: %v", requ.Resource, err)
		}

		patches := buildPodInjectionPatch(&pod, mutatedPod)
		if changed && len(patches) == 0 {
			log.Warnf("admission mutation produced no patch: uid=%s, namespace=%s, name=%s", requ.UID, requ.Namespace, podName)
		}
		return marshalPatch(requ, patches)

	default:
		return nil, fmt.Errorf("Unsupported resource: %#v", requ.Resource)
	}
}

func marshalPatch(requ *admissionv1.AdmissionRequest, patches []jsonpatch.JsonPatchOperation) (jsonPatch, error) {
	if patches == nil {
		patches = []jsonpatch.JsonPatchOperation{}
	}
	return json.Marshal(patches)
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
