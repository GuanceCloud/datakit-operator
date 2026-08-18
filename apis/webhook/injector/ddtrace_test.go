// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	corev1 "k8s.io/api/core/v1"
)

func newTestDDTraceRule(language, image string) *config.DDTraceRule {
	return &config.DDTraceRule{
		InjectRule: config.InjectRule{Image: image},
		Language:   language,
	}
}

func TestInjectDDTrace(t *testing.T) {
	t.Run("basic injection", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
			rule.Envs = []struct{ Key, Value string }{
				{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
				{"DD_TAGS", "host:node-02,system:linux"},
			}
			rule.Resources = config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			}
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
			rule.CheckAnnotation = true
			rule.Envs = []struct{ Key, Value string }{
				{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			}
			rule.Resources = config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			}
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("php", "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1")
			rule.CheckAnnotation = true
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("nodejs", "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2")
			rule.CheckAnnotation = true
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
			rule.CheckAnnotation = true
			rule.Envs = []struct{ Key, Value string }{
				{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			}
			rule.Resources = config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			}
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
			rule.Envs = []struct{ Key, Value string }{
				{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			}
			rule.Resources = config.ResourceRequirements{
				Requests: config.ResourceQuotaConfig{CPU: "100m", Memory: "64Mi"},
				Limits:   config.ResourceQuotaConfig{CPU: "200m", Memory: "128Mi"},
			}
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			return true, []*config.DDTraceRule{
				newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
			}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("php", "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1")
			rule.PHPLoaderFlavor = "linux-musl"
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("php", "pubrepo.guance.com/datakit-operator/php-lib-testing:v1.0.1")
			rule.PHPLoaderFlavor = "invalid"
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			rule := newTestDDTraceRule("nodejs", "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2")
			rule.Envs = []struct{ Key, Value string }{
				{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
			}
			return true, []*config.DDTraceRule{rule}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			return true, []*config.DDTraceRule{
				newTestDDTraceRule("nodejs", "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2"),
			}
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
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			return true, []*config.DDTraceRule{
				newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
			}
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
		originalPod := pod.DeepCopy()

		changed, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, originalPod, pod)
	})

	t.Run("nil pod error", func(t *testing.T) {
		_, err := InjectDDTraceToPod("", "test-pod", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot inject ddtrace-lib into nil pod")
	})

	t.Run("multiple rules: skip rule whose annotation does not match, try next", func(t *testing.T) {
		originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
		ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
			javaRule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
			javaRule.CheckAnnotation = true
			nodejsRule := newTestDDTraceRule("nodejs", "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v5.102.0")
			nodejsRule.CheckAnnotation = true
			return true, []*config.DDTraceRule{javaRule, nodejsRule}
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

func TestInjectDDTraceRepairsMissingJavaToolOptionsOnReinvocation(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-ddtrace-reinvocation", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})

	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assertCompleteJavaInjection(t, pod)

	// Simulate a later mutating webhook dropping the application container's env
	// while leaving the init container, volume, and volume mount intact.
	pod.Spec.Containers[0].Env = nil

	changed, err = InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assertCompleteJavaInjection(t, pod)

	afterRepair := pod.DeepCopy()
	changed, err = InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, afterRepair, pod)
	assertCompleteJavaInjection(t, pod)
}

func TestInjectDDTracePreservesJavaToolOptionsAndDoesNotDuplicateAgent(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	t.Run("preserves user options", func(t *testing.T) {
		pod := createTestPod("test-ddtrace-user-options", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "JAVA_TOOL_OPTIONS", Value: "-Xms128m -Duser.timezone=UTC"},
		}

		changed, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.True(t, changed)
		assertCompleteJavaInjection(t, pod)
		javaToolOptions, ok := findEnv(pod.Spec.Containers[0].Env, "JAVA_TOOL_OPTIONS")
		if assert.True(t, ok) {
			assert.Contains(t, javaToolOptions.Value, "-Xms128m -Duser.timezone=UTC")
		}
	})

	t.Run("keeps an existing Datakit agent option exactly once", func(t *testing.T) {
		pod := createTestPod("test-ddtrace-agent-exists", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "JAVA_TOOL_OPTIONS", Value: "-Xms128m -javaagent:/datadog-lib/dd-java-agent.jar -Duser.timezone=UTC"},
		}

		changed, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.True(t, changed)
		assertCompleteJavaInjection(t, pod)

		afterFirstInjection := pod.DeepCopy()
		changed, err = InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.False(t, changed)
		assert.Equal(t, afterFirstInjection, pod)
		assertCompleteJavaInjection(t, pod)
	})

	t.Run("recognizes an existing Datakit agent option with arguments", func(t *testing.T) {
		pod := createTestPod("test-ddtrace-agent-options-exist", map[string]string{
			ddtraceEnabledAnnotationKey: "true",
		})
		pod.Spec.Containers[0].Env = []corev1.EnvVar{
			{Name: "JAVA_TOOL_OPTIONS", Value: "-Xms128m -javaagent:/datadog-lib/dd-java-agent.jar=sample.rate=1 -Duser.timezone=UTC"},
		}

		changed, err := InjectDDTraceToPod("", pod.Name, pod)
		assert.NoError(t, err)
		assert.True(t, changed)
		javaToolOptions, ok := findEnv(pod.Spec.Containers[0].Env, "JAVA_TOOL_OPTIONS")
		if assert.True(t, ok) {
			assert.Equal(t, "-Xms128m -javaagent:/datadog-lib/dd-java-agent.jar=sample.rate=1 -Duser.timezone=UTC", javaToolOptions.Value)
			assert.Equal(t, 1, strings.Count(javaToolOptions.Value, javaAgentOption))
		}
	})
}

func TestInjectDDTraceRepairRequiresCompleteLibraryMounts(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	completePod := createTestPod("test-ddtrace-repair-prerequisites", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	changed, err := InjectDDTraceToPod("", completePod.Name, completePod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assertCompleteJavaInjection(t, completePod)

	tests := []struct {
		name   string
		remove func(*corev1.Pod)
	}{
		{
			name: "volume",
			remove: func(pod *corev1.Pod) {
				pod.Spec.Volumes = nil
			},
		},
		{
			name: "application volume mount",
			remove: func(pod *corev1.Pod) {
				pod.Spec.Containers[0].VolumeMounts = nil
			},
		},
		{
			name: "init container volume mount",
			remove: func(pod *corev1.Pod) {
				pod.Spec.InitContainers[0].VolumeMounts = nil
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pod := completePod.DeepCopy()
			pod.Spec.Containers[0].Env = nil
			tc.remove(pod)
			before := pod.DeepCopy()

			changed, err := InjectDDTraceToPod("", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
			_, exists := findEnv(pod.Spec.Containers[0].Env, "JAVA_TOOL_OPTIONS")
			assert.False(t, exists)
		})
	}
}

func TestInjectDDTraceRepairsOnlyMountedApplicationContainers(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-ddtrace-repair-mounted-containers", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	pod.Spec.Containers[0].Env = nil
	pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
		Name:  "later-webhook-sidecar",
		Image: "sidecar:latest",
		Env:   []corev1.EnvVar{{Name: "SIDECAR_ENV", Value: "keep"}},
	})

	changed, err = InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assertCompleteJavaInjection(t, pod)
	assert.Equal(t, []corev1.EnvVar{{Name: "SIDECAR_ENV", Value: "keep"}}, pod.Spec.Containers[1].Env)
	assert.Empty(t, pod.Spec.Containers[1].VolumeMounts)
}

func TestInjectDDTraceDoesNotRepairNonJavaPartialState(t *testing.T) {
	for _, lang := range []string{"python", "php", "nodejs"} {
		t.Run(lang, func(t *testing.T) {
			originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
			ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
				return true, []*config.DDTraceRule{
					newTestDDTraceRule(lang, "pubrepo.guance.com/datakit-operator/"+lang+"-lib-testing:v1.0.1"),
				}
			}
			defer func() {
				ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-ddtrace-non-java-repair", map[string]string{
				ddtraceEnabledAnnotationKey: "true",
			})
			changed, err := InjectDDTraceToPod("", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			pod.Spec.Containers[0].Env = nil
			before := pod.DeepCopy()

			changed, err = InjectDDTraceToPod("", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, before, pod)
		})
	}
}

func TestInjectDDTraceKeepsLanguageEnvPosition(t *testing.T) {
	tests := []struct {
		language      string
		envName       string
		envValue      string
		expectedValue string
	}{
		{
			language:      "java",
			envName:       "JAVA_TOOL_OPTIONS",
			envValue:      "-Xms128m",
			expectedValue: "-Xms128m -javaagent:/datadog-lib/dd-java-agent.jar",
		},
		{
			language:      "python",
			envName:       "PYTHONPATH",
			envValue:      "/app",
			expectedValue: "/datadog-lib/:/app",
		},
		{
			language:      "php",
			envName:       "PHP_INI_SCAN_DIR",
			envValue:      "/app/php",
			expectedValue: "/app/php:/datadog-lib",
		},
		{
			language:      "nodejs",
			envName:       "NODE_OPTIONS",
			envValue:      "--max-old-space-size=512",
			expectedValue: "--max-old-space-size=512 --require=/datadog-lib/node_modules/dd-trace/init",
		},
	}

	for _, tc := range tests {
		t.Run(tc.language, func(t *testing.T) {
			originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
			ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
				return true, []*config.DDTraceRule{
					newTestDDTraceRule(tc.language, "pubrepo.guance.com/datakit-operator/"+tc.language+"-lib-testing:v1.0.1"),
				}
			}
			defer func() {
				ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
			}()

			pod := createTestPod("test-ddtrace-language-env-position", map[string]string{
				ddtraceEnabledAnnotationKey: "true",
			})
			pod.Spec.Containers[0].Env = []corev1.EnvVar{
				{Name: "BEFORE", Value: "before"},
				{Name: tc.envName, Value: tc.envValue},
				{Name: "AFTER", Value: "after"},
			}

			changed, err := InjectDDTraceToPod("", pod.Name, pod)
			assert.NoError(t, err)
			assert.True(t, changed)
			assert.Equal(t, []string{"BEFORE", tc.envName, "AFTER"}, envNames(pod.Spec.Containers[0].Env[:3]))
			updated, exists := findEnv(pod.Spec.Containers[0].Env, tc.envName)
			if assert.True(t, exists) {
				assert.Equal(t, tc.expectedValue, updated.Value)
			}

			afterFirstInjection := pod.DeepCopy()
			changed, err = InjectDDTraceToPod("", pod.Name, pod)
			assert.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, afterFirstInjection, pod)
		})
	}
}

func TestInjectDDTraceKeepsConfiguredEnvOrderAndExistingPositions(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
		rule.Envs = []struct{ Key, Value string }{
			{Key: "NEW_FIRST", Value: "one"},
			{Key: "DD_TAGS", Value: "cluster:prod"},
			{Key: "EXISTING", Value: "replace-me"},
			{Key: "NEW_LAST", Value: "$(NEW_FIRST)"},
		}
		return true, []*config.DDTraceRule{rule}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-ddtrace-config-env-order", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "BEFORE", Value: "before"},
		{Name: "DD_TAGS", Value: "user:value"},
		{Name: "EXISTING", Value: "keep-me"},
		{Name: "AFTER", Value: "after"},
	}

	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []string{
		"BEFORE",
		"DD_TAGS",
		"EXISTING",
		"AFTER",
		"JAVA_TOOL_OPTIONS",
		"NEW_FIRST",
		"NEW_LAST",
	}, envNames(pod.Spec.Containers[0].Env))
	ddTags, exists := findEnv(pod.Spec.Containers[0].Env, "DD_TAGS")
	if assert.True(t, exists) {
		assert.Equal(t, "user:value,cluster:prod", ddTags.Value)
	}
	existing, exists := findEnv(pod.Spec.Containers[0].Env, "EXISTING")
	if assert.True(t, exists) {
		assert.Equal(t, "keep-me", existing.Value)
	}
}

func TestInjectDDTraceKeepsValueFromDDTagsInPlace(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		rule := newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1")
		rule.Envs = []struct{ Key, Value string }{
			{Key: "NEW_FIRST", Value: "one"},
			{Key: ddtraceDDTagsKey, Value: "cluster:prod"},
			{Key: "NEW_LAST", Value: "two"},
		}
		return true, []*config.DDTraceRule{rule}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	valueFrom := &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.annotations['tags']"},
	}
	pod := createTestPod("test-ddtrace-value-from-tags-order", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	pod.Spec.Containers[0].Env = []corev1.EnvVar{
		{Name: "BEFORE", Value: "before"},
		{Name: ddtraceDDTagsKey, ValueFrom: valueFrom.DeepCopy()},
		{Name: "AFTER", Value: "after"},
	}

	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []string{
		"BEFORE",
		ddtraceDDTagsKey,
		"AFTER",
		javaToolOptionsKey,
		"NEW_FIRST",
		"NEW_LAST",
	}, envNames(pod.Spec.Containers[0].Env))
	ddTags, exists := findEnv(pod.Spec.Containers[0].Env, ddtraceDDTagsKey)
	if assert.True(t, exists) {
		assert.Equal(t, valueFrom, ddTags.ValueFrom)
		assert.Empty(t, ddTags.Value)
	}
}

func TestInjectDDTraceDoesNotLeavePartialMutationOnConfigError(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-ddtrace-value-from", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name: "JAVA_TOOL_OPTIONS",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
		},
	}}
	originalPod := pod.DeepCopy()

	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, originalPod, pod)
}

func TestInjectDDTraceReinvocationKeepsValueFromUnchanged(t *testing.T) {
	originalFunc := ddtraceMatchAllNamespaceOrLabelsForConfig
	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.DDTraceRule) {
		return true, []*config.DDTraceRule{
			newTestDDTraceRule("java", "pubrepo.guance.com/datakit-operator/java-lib-testing:v1.0.1"),
		}
	}
	defer func() {
		ddtraceMatchAllNamespaceOrLabelsForConfig = originalFunc
	}()

	pod := createTestPod("test-ddtrace-reinvocation-value-from", map[string]string{
		ddtraceEnabledAnnotationKey: "true",
	})
	changed, err := InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.True(t, changed)
	pod.Spec.Containers[0].Env = []corev1.EnvVar{{
		Name: javaToolOptionsKey,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
		},
	}}
	before := pod.DeepCopy()

	changed, err = InjectDDTraceToPod("", pod.Name, pod)
	assert.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, before, pod)
}

func assertCompleteJavaInjection(t *testing.T, pod *corev1.Pod) {
	t.Helper()

	assert.Equal(t, 1, countInitContainers(pod, ddtraceInitContainerName))
	assert.Equal(t, 1, countVolumes(pod, ddtraceVolumeName))
	if !assert.NotEmpty(t, pod.Spec.InitContainers) || !assert.NotEmpty(t, pod.Spec.Containers) {
		return
	}
	assert.Equal(t, 1, countVolumeMounts(pod.Spec.InitContainers[0].VolumeMounts, ddtraceVolumeName))
	assert.Equal(t, 1, countVolumeMounts(pod.Spec.Containers[0].VolumeMounts, ddtraceVolumeName))
	assert.Equal(t, 1, countEnvs(pod.Spec.Containers[0].Env, "JAVA_TOOL_OPTIONS"))

	javaToolOptions, ok := findEnv(pod.Spec.Containers[0].Env, "JAVA_TOOL_OPTIONS")
	if assert.True(t, ok) {
		assert.Equal(t, 1, strings.Count(javaToolOptions.Value, "-javaagent:/datadog-lib/dd-java-agent.jar"))
	}
}

func findEnv(envs []corev1.EnvVar, name string) (corev1.EnvVar, bool) {
	for _, env := range envs {
		if env.Name == name {
			return env, true
		}
	}
	return corev1.EnvVar{}, false
}

func countEnvs(envs []corev1.EnvVar, name string) int {
	count := 0
	for _, env := range envs {
		if env.Name == name {
			count++
		}
	}
	return count
}

func countInitContainers(pod *corev1.Pod, name string) int {
	count := 0
	for _, container := range pod.Spec.InitContainers {
		if container.Name == name {
			count++
		}
	}
	return count
}

func countVolumes(pod *corev1.Pod, name string) int {
	count := 0
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == name {
			count++
		}
	}
	return count
}

func countVolumeMounts(volumeMounts []corev1.VolumeMount, name string) int {
	count := 0
	for _, volumeMount := range volumeMounts {
		if volumeMount.Name == name {
			count++
		}
	}
	return count
}

func envNames(envs []corev1.EnvVar) []string {
	names := make([]string, 0, len(envs))
	for _, env := range envs {
		names = append(names, env.Name)
	}
	return names
}
