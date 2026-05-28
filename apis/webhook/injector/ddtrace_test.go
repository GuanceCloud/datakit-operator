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
)

func TestInjectDDTrace(t *testing.T) {
	t.Run("basic injection", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
					{"DD_TAGS", "host:node-02,system:linux"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "datakit-lib-init", pod.Spec.InitContainers[0].Name)
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)
	})

	t.Run("CheckAnnotation=true with annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "java",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-check-annotation", map[string]string{
			ddtraceEnabledAnnotationKey:          "true",
			"admission.datakit/java-lib.version": "v2.0.0",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-lib-testing:v2.0.0", pod.Spec.InitContainers[0].Image)
	})

	t.Run("CheckAnnotation=true with php annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "php",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-check-annotation-php", map[string]string{
			ddtraceEnabledAnnotationKey:         "true",
			"admission.datakit/php-lib.version": "v2.0.0",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/php-lib-testing:v2.0.0", pod.Spec.InitContainers[0].Image)
	})

	t.Run("CheckAnnotation=true with nodejs annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "nodejs",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-check-annotation-nodejs", map[string]string{
			ddtraceEnabledAnnotationKey:            "true",
			"admission.datakit/nodejs-lib.version": "v4.0.0",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v4.0.0", pod.Spec.InitContainers[0].Image)
	})

	t.Run("CheckAnnotation=true without annotation", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "java",
				CheckAnnotation: true,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-no-annotation", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 0)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("CheckAnnotation=false", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "java",
				CheckAnnotation: false,
				Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
				Resources: config.ResourceRequirements{
					Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
					Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
				},
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-no-check", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1", pod.Spec.InitContainers[0].Image)
	})

	t.Run("skip when annotation is false", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-disabled", map[string]string{
			ddtraceEnabledAnnotationKey: "false",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 0)
		assert.Len(t, pod.Spec.Volumes, 0)
	})

	t.Run("php basic injection", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "php",
				PHPLoaderFlavor: "linux-musl",
				Image:           "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-php", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "datakit-lib-init", pod.Spec.InitContainers[0].Name)
		assert.Equal(t, []string{"sh", "-c"}, pod.Spec.InitContainers[0].Command[:2])
		assert.Contains(t, pod.Spec.InitContainers[0].Command[2], "/datadog-lib/linux-musl/loader/dd_library_loader.ini")

		envMap := map[string]string{}
		for _, env := range pod.Spec.Containers[0].Env {
			envMap[env.Name] = env.Value
		}
		assert.Equal(t, "/datadog-lib", envMap["DD_LOADER_PACKAGE_PATH"])
		assert.Equal(t, "/usr/local/etc/php/conf.d:/datadog-lib", envMap["PHP_INI_SCAN_DIR"])
	})

	t.Run("php invalid flavor fallback to linux-gnu", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language:        "php",
				PHPLoaderFlavor: "invalid",
				Image:           "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-php-loader-default", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Contains(t, pod.Spec.InitContainers[0].Command[2], "/datadog-lib/linux-gnu/loader/dd_library_loader.ini")
	})

	t.Run("nodejs basic injection", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language: "nodejs",
				Image:    "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2",
				Envs: []struct{ Key, Value string }{
					{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				},
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-nodejs", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "NODE_OPTIONS", Value: "--max-old-space-size=512"},
		}

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "datakit-lib-init", pod.Spec.InitContainers[0].Name)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2", pod.Spec.InitContainers[0].Image)
		assert.Len(t, pod.Spec.Volumes, 1)
		assert.Len(t, pod.Spec.Containers[0].VolumeMounts, 1)

		envMap := map[string]string{}
		for _, env := range pod.Spec.Containers[0].Env {
			envMap[env.Name] = env.Value
		}
		assert.Equal(t, "--max-old-space-size=512 --require=/datadog-lib/node_modules/dd-trace/init", envMap["NODE_OPTIONS"])
		assert.Equal(t, "datakit-service.datakit.svc", envMap["DD_AGENT_HOST"])
	})

	t.Run("nodejs injection without pre-existing env", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language: "nodejs",
				Image:    "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-nodejs-no-preexist-env", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, " --require=/datadog-lib/node_modules/dd-trace/init", pod.Spec.Containers[0].Env[0].Value)
	})

	t.Run("skip when init container already exists", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{{
				Language: "java",
				Image:    "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
			}}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		pod := createTestPod("test-ddtrace-exists", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.InitContainers = []corev1.Container{
			{Name: ddtraceInitContainerName, Image: "existing-image"},
		}

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "existing-image", pod.Spec.InitContainers[0].Image)
	})

	t.Run("nil pod error", func(t *testing.T) {
		_, err := InjectDDTraceToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject ddtrace-lib into nil pod")
	})

	t.Run("multiple rules: skip rule whose annotation does not match, try next", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
			return true, []*config.InjectRule{
				{
					Language:        "java",
					CheckAnnotation: true,
					Image:           "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1",
				},
				{
					Language:        "nodejs",
					CheckAnnotation: true,
					Image:           "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v5.102.0",
				},
			}
		}
		defer func() {
			ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
		}()

		// Pod only has nodejs annotation, no java annotation.
		// The first rule (java) should be skipped, the second (nodejs) should match.
		pod := createTestPod("test-ddtrace-multi-rule", map[string]string{
			ddtraceEnabledAnnotationKey:            "true",
			"admission.datakit/nodejs-lib.version": "",
		})

		_, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)

		assert.Len(t, pod.Spec.InitContainers, 1)
		assert.Equal(t, "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v5.102.0", pod.Spec.InitContainers[0].Image)
	})
}
