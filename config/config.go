// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

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
		ServerListen: ":9543",
		LogLevel:     "info",
		AdmissionInject: AdmissionInjectConfig{
			DDTrace:  newContainerConfig(),
			Logfwd:   newContainerConfig(),
			Profiler: newContainerConfig(),
		},
	}
}

// Validate performs basic sanity checks after loading configuration.
func (c *Configuration) Validate() error {
	// ensure image maps are always initialized
	if c.AdmissionInject.DDTrace.Images == nil {
		c.AdmissionInject.DDTrace.Images = make(map[string]string)
	}
	if c.AdmissionInject.Logfwd.Images == nil {
		c.AdmissionInject.Logfwd.Images = make(map[string]string)
	}
	if c.AdmissionInject.Profiler.Images == nil {
		c.AdmissionInject.Profiler.Images = make(map[string]string)
	}
	return nil
}
