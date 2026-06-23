// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func servePods(t *testing.T, pod *corev1.Pod, view string) []corev1.Pod {
	t.Helper()

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	if err := indexer.Add(pod); err != nil {
		t.Fatalf("add pod to indexer: %v", err)
	}

	router := gin.New()
	router.GET("/pods", (&Handler{PodLister: corev1listers.NewPodLister(indexer)}).ListAllPods)

	url := "/pods"
	if view != "" {
		url += "?view=" + view
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var pods []corev1.Pod
	if err := json.Unmarshal(w.Body.Bytes(), &pods); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("got %d pods, want 1", len(pods))
	}
	return pods
}

func TestListAllPodsEBPFV1ViewTrimsPodFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	controller := true
	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			UID:       "pod-uid",
			Name:      "app-0",
			Namespace: "default",
			Labels:    map[string]string{"app": "demo"},
			Annotations: map[string]string{
				"kubectl.kubernetes.io/last-applied-configuration": "...",
			},
			ManagedFields:  []v1.ManagedFieldsEntry{{Manager: "kcm"}},
			OwnerReferences: []v1.OwnerReference{{
				APIVersion: "apps/v1", Kind: "ReplicaSet",
				Name: "rs-0", UID: "owner-uid", Controller: &controller,
			}},
		},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			HostPID:     true,
			HostIPC:     true,
			Containers: []corev1.Container{{
				Name:  "app",
				Image: "nginx:1.25",
				Env:   []corev1.EnvVar{{Name: "SECRET", Value: "x"}},
				Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
			}},
			Volumes: []corev1.Volume{{Name: "config"}},
		},
		Status: corev1.PodStatus{
			HostIP: "10.0.0.1", PodIP: "10.244.0.10",
			PodIPs:            []corev1.PodIP{{IP: "10.244.0.10"}},
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app", Image: "nginx:1.25", Ready: true}},
		},
	}

	got := servePods(t, pod, "ebpf-v1")[0]

	// retained
	ok := got.UID == "pod-uid" && got.Name == "app-0" && got.Namespace == "default" &&
		got.Spec.HostNetwork && got.Spec.HostPID && got.Spec.HostIPC &&
		got.Status.HostIP == "10.0.0.1" && got.Status.PodIP == "10.244.0.10" &&
		got.Labels["app"] == "demo" && len(got.OwnerReferences) == 1 &&
		len(got.Spec.Containers) == 1 && got.Spec.Containers[0].Name == "app" &&
		len(got.Spec.Containers[0].Ports) == 1 && got.Spec.Containers[0].Ports[0].ContainerPort == 8080
	if !ok {
		t.Errorf("retained: %+v", got)
	}
	// stripped
	if got.Spec.Containers[0].Image != "" || len(got.Spec.Containers[0].Env) != 0 || len(got.Annotations) != 0 ||
		len(got.ManagedFields) != 0 || len(got.Spec.Volumes) != 0 ||
		got.Status.Phase != "" || len(got.Status.ContainerStatuses) != 0 {
		t.Errorf("stripped: %+v", got)
	}
}

func TestListAllPodsDefaultViewKeepsFullPod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:          "app-0",
			Namespace:     "default",
			ManagedFields: []v1.ManagedFieldsEntry{{Manager: "kcm"}},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "nginx:1.25"}},
		},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app", Image: "nginx:1.25"}},
		},
	}

	got := servePods(t, pod, "")[0]

	if got.Spec.Containers[0].Image != "nginx:1.25" ||
		got.Status.Phase != corev1.PodRunning ||
		len(got.Status.ContainerStatuses) != 1 ||
		len(got.ManagedFields) != 1 {
		t.Errorf("full pod not preserved: %+v", got)
	}
}
