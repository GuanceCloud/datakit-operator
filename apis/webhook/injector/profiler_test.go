// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectProfiler(t *testing.T) {
	t.Run("inject java profiler", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DK_AGENT_PORT", "9529"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-java",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey:              "true",
					"admission.datakit/java-profiler.version": "v2.0.0",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Check profiler container
		assert.Len(t, pod.Spec.Containers, 2)
		profilerContainer := pod.Spec.Containers[1]
		assert.Equal(t, profilerContainerName, profilerContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-profiler-testing:v2.0.0", profilerContainer.Image)

		// Check volumes
		assert.Len(t, pod.Spec.Volumes, 3)
		assert.Equal(t, profilerVolumeName, pod.Spec.Volumes[0].Name)
		assert.Equal(t, profilerTmp, pod.Spec.Volumes[1].Name)
		assert.Equal(t, profilerTimezone, pod.Spec.Volumes[2].Name)

		// Check ShareProcessNamespace
		assert.NotNil(t, pod.Spec.ShareProcessNamespace)
		assert.True(t, *pod.Spec.ShareProcessNamespace)
		assert.Equal(t, corev1.RestartPolicyAlways, pod.Spec.RestartPolicy)
	})

	t.Run("inject python profiler", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerPythonImageKey: "pubrepo.guance.com/datakit-operator/python-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-python",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey:                "true",
					"admission.datakit/python-profiler.version": "latest",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "python:3.9"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Check profiler container
		assert.Len(t, pod.Spec.Containers, 2)
		profilerContainer := pod.Spec.Containers[1]
		assert.Equal(t, profilerContainerName, profilerContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/python-profiler-testing:latest", profilerContainer.Image)
	})

	t.Run("inject golang profiler", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerGolangImageKey: "pubrepo.guance.com/datakit-operator/golang-profiler-testing:v1.0.1",
				},
				Envs: []struct{ Key, Value string }{
					{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-golang",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey:                "true",
					"admission.datakit/golang-profiler.version": "v1.5.0",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "golang:1.19"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Check profiler container
		assert.Len(t, pod.Spec.Containers, 2)
		profilerContainer := pod.Spec.Containers[1]
		assert.Equal(t, profilerContainerName, profilerContainer.Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/golang-profiler-testing:v1.5.0", profilerContainer.Image)
	})

	t.Run("skip injection when profiler.enabled is false", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-disabled",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey: "false",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Should not inject
		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("skip injection when profiler version annotation missing", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: false,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-no-version",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey: "true",
					// Missing profiler version annotation
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Should not inject because no version annotation
		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("skip injection when CheckAnnotation=true but version annotation missing", func(t *testing.T) {
		originalFunc := profilerMatchNamespaceOrLabelsForConfig
		profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
			return true, &config.InjectRule{
				CheckAnnotation: true,
				Images: map[string]string{
					config.DeprecatedProfilerJavaImageKey: "pubrepo.guance.com/datakit-operator/java-profiler-testing:v1.0.1",
				},
			}
		}
		defer func() {
			profilerMatchNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-profiler-check-annotation",
				Annotations: map[string]string{
					profilerEnabledAnnotationKey: "true",
					// Missing profiler version annotation
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app", Image: "nginx:1.22"},
				},
			},
		}

		err := InjectProfilerToPod("", pod.Name, pod)
		assert.NoError(t, err)

		// Should not inject because CheckAnnotation=true requires version annotation
		assert.Len(t, pod.Spec.Containers, 1)
		assert.Len(t, pod.Spec.Volumes, 0)
	})
}
