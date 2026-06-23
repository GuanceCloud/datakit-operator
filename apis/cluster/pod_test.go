package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestListAllPodsEBPFV1ViewTrimsPodFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			UID:       types.UID("pod-uid"),
			Name:      "app-0",
			Namespace: "default",
			Labels: map[string]string{
				"app":        "demo",
				"project_id": "p1",
			},
			Annotations: map[string]string{
				"large": "annotation",
			},
			ManagedFields: []v1.ManagedFieldsEntry{
				{Manager: "kube-controller-manager"},
			},
			OwnerReferences: []v1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       "app-7f9d7f",
					UID:        types.UID("owner-uid"),
					Controller: boolPtr(true),
				},
			},
		},
		Spec: corev1.PodSpec{
			HostNetwork: true,
			HostPID:     true,
			HostIPC:     true,
			Volumes: []corev1.Volume{
				{Name: "config"},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx:1.25",
					Env: []corev1.EnvVar{
						{Name: "SECRET", Value: "value"},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "config", MountPath: "/etc/config"},
					},
					Ports: []corev1.ContainerPort{
						{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			HostIP: "10.0.0.1",
			PodIP:  "10.244.0.10",
			PodIPs: []corev1.PodIP{
				{IP: "10.244.0.10"},
			},
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Image: "nginx:1.25", Ready: true},
			},
		},
	}

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	if err := indexer.Add(pod); err != nil {
		t.Fatalf("add pod to indexer: %v", err)
	}

	h := &Handler{PodLister: corev1listers.NewPodLister(indexer)}

	router := gin.New()
	router.GET("/pods", h.ListAllPods)

	req := httptest.NewRequest(http.MethodGet, "/pods?view=ebpf-v1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var pods []corev1.Pod
	if err := json.Unmarshal(w.Body.Bytes(), &pods); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("pods len = %d, want 1", len(pods))
	}

	got := pods[0]
	if got.UID != types.UID("pod-uid") ||
		got.Name != "app-0" ||
		got.Namespace != "default" ||
		got.Labels["app"] != "demo" ||
		len(got.OwnerReferences) != 1 {
		t.Fatalf("trimmed metadata mismatch: %#v", got.ObjectMeta)
	}

	if !got.Spec.HostNetwork || !got.Spec.HostPID || !got.Spec.HostIPC {
		t.Fatalf("host namespace flags were not preserved: %#v", got.Spec)
	}
	if len(got.Spec.Containers) != 1 {
		t.Fatalf("containers len = %d, want 1", len(got.Spec.Containers))
	}
	if got.Spec.Containers[0].Name != "app" ||
		len(got.Spec.Containers[0].Ports) != 1 ||
		got.Spec.Containers[0].Ports[0].ContainerPort != 8080 {
		t.Fatalf("trimmed container mismatch: %#v", got.Spec.Containers[0])
	}

	if got.Status.HostIP != "10.0.0.1" ||
		got.Status.PodIP != "10.244.0.10" ||
		len(got.Status.PodIPs) != 1 ||
		got.Status.PodIPs[0].IP != "10.244.0.10" {
		t.Fatalf("trimmed status mismatch: %#v", got.Status)
	}

	if len(got.Annotations) != 0 ||
		len(got.ManagedFields) != 0 ||
		len(got.Spec.Volumes) != 0 ||
		got.Spec.Containers[0].Image != "" ||
		len(got.Spec.Containers[0].Env) != 0 ||
		len(got.Spec.Containers[0].VolumeMounts) != 0 ||
		got.Status.Phase != "" ||
		len(got.Status.ContainerStatuses) != 0 {
		t.Fatalf("large fields should be trimmed: %#v", got)
	}
}

func TestListAllPodsDefaultViewKeepsFullPod(t *testing.T) {
	gin.SetMode(gin.TestMode)

	pod := &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      "app-0",
			Namespace: "default",
			ManagedFields: []v1.ManagedFieldsEntry{
				{Manager: "test"},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx:1.25"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "app", Image: "nginx:1.25"},
			},
		},
	}

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})
	if err := indexer.Add(pod); err != nil {
		t.Fatalf("add pod to indexer: %v", err)
	}

	h := &Handler{PodLister: corev1listers.NewPodLister(indexer)}

	router := gin.New()
	router.GET("/pods", h.ListAllPods)

	req := httptest.NewRequest(http.MethodGet, "/pods", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var pods []corev1.Pod
	if err := json.Unmarshal(w.Body.Bytes(), &pods); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("pods len = %d, want 1", len(pods))
	}
	if pods[0].Spec.Containers[0].Image != "nginx:1.25" ||
		pods[0].Status.Phase != corev1.PodRunning ||
		len(pods[0].Status.ContainerStatuses) != 1 ||
		len(pods[0].ManagedFields) != 1 {
		t.Fatalf("default view should keep full pod: %#v", pods[0])
	}
}

func boolPtr(v bool) *bool {
	return &v
}
