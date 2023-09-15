package injector

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/envbuilder"
	corev1 "k8s.io/api/core/v1"
)

type language string

const (
	java   language = "java"
	golang language = "golang"
	js     language = "js"
	python language = "python"
)

const enableEnvFieldRef = true

var (
	ddtraceJavaAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTraceJavaImageKey)
	}

	ddtracePythonAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTracePythonImageKey)
	}

	ddtraceJsAgentImage = func() string {
		return config.Cfg.AdmissionInject.DDTrace.Image(config.DDTraceJsImageKey)
	}

	logfwdAppImage = func() string {
		return config.Cfg.AdmissionInject.Logfwd.Image(config.LogfwdImageKey)
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
)
