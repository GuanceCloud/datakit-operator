package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var Cfg = initDefaultConfiguration()

func initDefaultConfiguration() *Configuration {
	return &Configuration{
		AdmissionInject: AdmissionInjectConfig{
			DDTrace:  newContainerConfig(),
			Logfwd:   newContainerConfig(),
			Profiler: newContainerConfig(),
		},
	}
}

func LoadConfigWithEnv() error {
	cfgStr := os.Getenv("ENV_JSON_CONFIG")
	return parseConfig(cfgStr, Cfg)
}

func parseConfig(cfgStr string, c *Configuration) error {
	if cfgStr != "" {
		if err := json.Unmarshal([]byte(cfgStr), c); err != nil {
			return fmt.Errorf("unable to unmarshal config: %w", err)
		}
	}

	loadEnvs(c)
	c.AdmissionInject.setup()
	return nil
}

// loadEnvs
// Deprecated: No longer used; kept for compatibility.
func loadEnvs(c *Configuration) {
	if v := os.Getenv("ENV_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}

	if v := os.Getenv("ENV_SERVER_LISTEN"); v != "" {
		c.ServerListen = v
	}

	if v := os.Getenv("ENV_DD_AGENT_HOST"); v != "" {
		c.AdmissionInject.DDTrace.Environments["DD_AGENT_HOST"] = v
	}

	if v := os.Getenv("ENV_DD_TRACE_AGENT_PORT"); v != "" {
		c.AdmissionInject.DDTrace.Environments["DD_TRACE_AGENT_PORT"] = v
	}

	if v := os.Getenv("ENV_DD_JAVA_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTraceJavaImageKey] = v
	}

	if v := os.Getenv("ENV_DD_PYTHON_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTracePythonImageKey] = v
	}

	if v := os.Getenv("ENV_DD_JS_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTraceJsImageKey] = v
	}

	if v := os.Getenv("ENV_LOGFWD_IMAGE"); v != "" {
		c.AdmissionInject.Logfwd.Images[LogfwdImageKey] = v
	}
}
