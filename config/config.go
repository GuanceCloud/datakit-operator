package config

import "gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"

var (
	Cfg = initDefaultConfiguration()
	log = logger.DefaultSLogger("config")
)

const (
	DDTraceJavaImageKey   = "java_agent_image"
	DDTracePythonImageKey = "python_agent_image"
	DDTraceNodejsImageKey = "js_agent_image"

	LogfwdImageKey            = "logfwd_image"
	LogfwdReuseExistVolumeOpt = "reuse_exist_volume"

	ProfilerJavaImageKey   = "java_profiler_image"
	ProfilerPythonImageKey = "python_profiler_image"
	ProfilerGolangImageKey = "golang_profiler_image"
)

type Configuration struct {
	ServerListen    string                `json:"server_listen"`
	LogLevel        string                `json:"log_level"`
	AdmissionInject AdmissionInjectConfig `json:"admission_inject"`
	AdmissionMutate AdmissionMutateConfig `json:"admission_mutate"`
}

func initLog() {
	log = logger.SLogger("config")
}

func initDefaultConfiguration() *Configuration {
	return &Configuration{
		AdmissionInject: AdmissionInjectConfig{
			DDTrace:  newContainerConfig(),
			Logfwd:   newContainerConfig(),
			Profiler: newContainerConfig(),
		},
	}
}
