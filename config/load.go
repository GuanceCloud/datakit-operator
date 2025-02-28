package config

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadConfigWithEnv() error {
	initLog()
	log.Info("loading config..")
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

	if err := c.AdmissionInject.Setup(); err != nil {
		return err
	}
	if err := c.AdmissionMutate.Setup(); err != nil {
		return err
	}
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
		for idx, item := range c.AdmissionInject.DDTrace.Environments {
			key, ok := item.Key.(string)
			if !ok {
				continue
			}
			if key == "DD_AGENT_HOST" {
				c.AdmissionInject.DDTrace.Environments[idx].Value = v
			}
		}
	}

	if v := os.Getenv("ENV_DD_TRACE_AGENT_PORT"); v != "" {
		for idx, item := range c.AdmissionInject.DDTrace.Environments {
			key, ok := item.Key.(string)
			if !ok {
				continue
			}
			if key == "DD_TRACE_AGENT_PORT" {
				c.AdmissionInject.DDTrace.Environments[idx].Value = v
			}
		}
	}

	if v := os.Getenv("ENV_DD_JAVA_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTraceJavaImageKey] = v
	}

	if v := os.Getenv("ENV_DD_PYTHON_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTracePythonImageKey] = v
	}

	if v := os.Getenv("ENV_DD_JS_AGENT_IMAGE"); v != "" {
		c.AdmissionInject.DDTrace.Images[DDTraceNodejsImageKey] = v
	}

	if v := os.Getenv("ENV_LOGFWD_IMAGE"); v != "" {
		c.AdmissionInject.Logfwd.Images[LogfwdImageKey] = v
	}
}
