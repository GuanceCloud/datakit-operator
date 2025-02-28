package injector

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	corev1 "k8s.io/api/core/v1"
)

type language string

const (
	null             language = ""
	java             language = "java"
	golang           language = "golang"
	nodejs           language = "nodejs"
	nodejsDeprecated language = "js"
	python           language = "python"
)

const enableEnvFieldRef = true

var (
	ddtraceEnabledNamespaces = func(ns string) string {
		return config.Cfg.AdmissionInject.DDTrace.MatchNamespace(ns)
	}

	ddtraceEnabledLabelSelectors = func(labels map[string]string) string {
		return config.Cfg.AdmissionInject.DDTrace.MatchLabelSelector(labels)
	}

	ddtraceJavaAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTraceJavaImageKey)
	}

	ddtracePythonAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTracePythonImageKey)
	}

	ddtraceNodejsAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTraceNodejsImageKey)
	}

	logfwdImage = func() string {
		return config.Cfg.AdmissionInject.Logfwd.Image(config.LogfwdImageKey)
	}

	logfwdResourceRequests = func() (cpu string, memory string) {
		return config.Cfg.AdmissionInject.Logfwd.ResourceRequests()
	}

	logfwdResourceLimits = func() (cpu string, memory string) {
		return config.Cfg.AdmissionInject.Logfwd.ResourceLimits()
	}

	profilerJavaImage = func() string {
		return config.Cfg.AdmissionInject.Profiler.Image(config.ProfilerJavaImageKey)
	}

	profilerPythonImage = func() string {
		return config.Cfg.AdmissionInject.Profiler.Image(config.ProfilerPythonImageKey)
	}

	profilerGolangImage = func() string {
		return config.Cfg.AdmissionInject.Profiler.Image(config.ProfilerGolangImageKey)
	}

	profilerResourceRequests = func() (cpu string, memory string) {
		return config.Cfg.AdmissionInject.Profiler.ResourceRequests()
	}

	profilerResourceLimits = func() (cpu string, memory string) {
		return config.Cfg.AdmissionInject.Profiler.ResourceLimits()
	}
)

var (
	ddtraceEnvs = func() []struct{ Key, Value string } {
		return config.Cfg.AdmissionInject.DDTrace.Envs()
	}

	ddtraceEnvObjects = func() []corev1.EnvVar {
		envs := ddtraceEnvs()
		return envbuilder.BuildEnvs(envs, enableEnvFieldRef)
	}

	profilerEnvs = func() []struct{ Key, Value string } {
		return config.Cfg.AdmissionInject.Profiler.Envs()
	}

	profilerEnvObjects = func() []corev1.EnvVar {
		envs := profilerEnvs()
		return envbuilder.BuildEnvs(envs, enableEnvFieldRef)
	}

	logfwdEnvs = func() []struct{ Key, Value string } {
		return config.Cfg.AdmissionInject.Logfwd.Envs()
	}

	logfwdEnvObjects = func() []corev1.EnvVar {
		envs := logfwdEnvs()
		return envbuilder.BuildEnvs(envs, enableEnvFieldRef)
	}
)
