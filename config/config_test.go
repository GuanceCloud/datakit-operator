package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	err := os.Setenv("ENV_LOG_LEVEL", "debug")
	assert.NoError(t, err)
	defer func() {
		_ = os.Unsetenv("ENV_LOG_LEVEL")
	}()

	var testcase = struct {
		inCfgStr string
		inCfg    *Configuration
		outCfg   *Configuration
	}{
		inCfgStr: `
          		{
          		    "server_listen": "0.0.0.0:9543",
          		    "log_level":     "info",
          		    "admission_inject": {
          		        "ddtrace": {
          		            "images": {
          		                "java_agent_image":   "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.8.4-guance",
          		                "python_agent_image": "pubrepo.guance.com/datakit-operator/dd-lib-python-init:v1.6.2",
          		                "js_agent_image":     "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2"
          		            },
          		            "envs": {
          		                "DD_AGENT_HOST":           "datakit-service.datakit.svc",
          		                "DD_TRACE_AGENT_PORT":     "9529",
          		                "DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc",
          		                "DD_JMXFETCH_STATSD_PORT": "8125"
          		            }
          		        },
          		        "logfwd": {
          		            "images": {
          		                "logfwd_image": "pubrepo.guance.com/datakit/logfwd:1.5.8"
          		            }
          		        },
          		        "profiler": {
          		            "images": {
          		                "java_profiler_image":   "pubrepo.guance.com/dataflux/async-profiler:0.1.0",
          		                "python_profiler_image": "pubrepo.guance.com/dataflux/py-spy:0.1.0"
          		            },
          		            "envs": {
          		                "DK_AGENT_HOST":       "datakit-service.datakit.svc",
          		                "DK_AGENT_PORT":       "9529",
          		                "DK_PROFILE_VERSION":  "1.2.33",
          		                "DK_PROFILE_ENV":      "prod",
          		                "DK_PROFILE_DURATION": "240",
          		                "DK_PROFILE_SCHEDULE": "*/20 * * * *"
          		            }
          		        }
          		    }
          		}
       		`,
		inCfg: &Configuration{},
		outCfg: &Configuration{
			ServerListen: "0.0.0.0:9543",
			LogLevel:     "debug",
			AdmissionInject: AdmissionInjectConfig{
				DDTrace: ContainerConfig{
					Images: map[string]string{
						"java_agent_image":   "pubrepo.guance.com/datakit-operator/dd-lib-java-init:v1.8.4-guance",
						"python_agent_image": "pubrepo.guance.com/datakit-operator/dd-lib-python-init:v1.6.2",
						"js_agent_image":     "pubrepo.guance.com/datakit-operator/dd-lib-js-init:v3.9.2",
					},
					Environments: map[string]string{
						"DD_AGENT_HOST":           "datakit-service.datakit.svc",
						"DD_TRACE_AGENT_PORT":     "9529",
						"DD_JMXFETCH_STATSD_HOST": "datakit-service.datakit.svc",
						"DD_JMXFETCH_STATSD_PORT": "8125",
					},
					envs: Envs{
						{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
						{"DD_JMXFETCH_STATSD_HOST", "datakit-service.datakit.svc"},
						{"DD_JMXFETCH_STATSD_PORT", "8125"},
						{"DD_TRACE_AGENT_PORT", "9529"},
					},
				},
				Logfwd: ContainerConfig{
					Images: map[string]string{
						"logfwd_image": "pubrepo.guance.com/datakit/logfwd:1.5.8",
					},
				},
				Profiler: ContainerConfig{
					Images: map[string]string{
						"java_profiler_image":   "pubrepo.guance.com/dataflux/async-profiler:0.1.0",
						"python_profiler_image": "pubrepo.guance.com/dataflux/py-spy:0.1.0",
					},
					Environments: map[string]string{
						"DK_AGENT_HOST":       "datakit-service.datakit.svc",
						"DK_AGENT_PORT":       "9529",
						"DK_PROFILE_VERSION":  "1.2.33",
						"DK_PROFILE_ENV":      "prod",
						"DK_PROFILE_DURATION": "240",
						"DK_PROFILE_SCHEDULE": "*/20 * * * *",
					},
					envs: Envs{
						{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
						{"DK_AGENT_PORT", "9529"},
						{"DK_PROFILE_DURATION", "240"},
						{"DK_PROFILE_ENV", "prod"},
						{"DK_PROFILE_SCHEDULE", "*/20 * * * *"},
						{"DK_PROFILE_VERSION", "1.2.33"},
					},
				},
			},
		},
	}

	err = parseConfig(testcase.inCfgStr, testcase.inCfg)
	assert.NoError(t, err)

	t.Logf("result: %#v\n", testcase.inCfg)
	assert.Equal(t, testcase.outCfg, testcase.inCfg)
}
