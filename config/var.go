package config

import (
	"sort"
)

const (
	DDTraceJavaImageKey   = "java_agent_image"
	DDTracePythonImageKey = "python_agent_image"
	DDTraceJsImageKey     = "js_agent_image"

	LogfwdImageKey = "logfwd_image"

	ProfilerJavaImageKey   = "java_profiler_image"
	ProfilerPythonImageKey = "python_profiler_image"
	ProfilerGolangImageKey = "golang_profiler_image"
)

type Configuration struct {
	ServerListen    string                `json:"server_listen"`
	LogLevel        string                `json:"log_level"`
	AdmissionInject AdmissionInjectConfig `json:"admission_inject"`
}

type AdmissionInjectConfig struct {
	DDTrace  ContainerConfig `json:"ddtrace"`
	Logfwd   ContainerConfig `json:"logfwd"`
	Profiler ContainerConfig `json:"profiler"`
}

func (c *AdmissionInjectConfig) setup() {
	c.DDTrace.fillEnvs()
	c.Logfwd.fillEnvs()
	c.Profiler.fillEnvs()
}

type Envs []struct{ Key, Value string }

type ContainerConfig struct {
	Images       map[string]string `json:"images"`
	Environments map[string]string `json:"envs"`
	envs         Envs
}

func (c ContainerConfig) Image(name string) string { return c.Images[name] }
func (c ContainerConfig) Envs() Envs               { return c.envs }

func newContainerConfig() ContainerConfig {
	return ContainerConfig{
		Images:       make(map[string]string),
		Environments: make(map[string]string),
	}
}

func (c *ContainerConfig) fillEnvs() {
	if len(c.Environments) == 0 {
		return
	}

	var keys []string
	for key := range c.Environments {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		value := c.Environments[key]
		c.envs = append(c.envs, struct{ Key, Value string }{key, value})
	}
}
