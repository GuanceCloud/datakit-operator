package injector

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

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

	ddtraceEnvs = func() []struct{ Key, Value string } {
		return config.Cfg.AdmissionInject.DDTrace.Envs()
	}

	profilerEnvs = func() []struct{ Key, Value string } {
		return config.Cfg.AdmissionInject.Profiler.Envs()
	}
)
