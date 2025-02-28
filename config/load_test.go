package config

import (
	"os"
	"testing"

	"github.com/ake-persson/mapslice-json"
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
          		                "DD_TRACE_AGENT_PORT":     "9529"
                                    }
                                },
                                "logfwd": {
          		            "images": {
          		                "logfwd_image": "pubrepo.guance.com/datakit/logfwd:1.5.8"
                                    },
                                    "resources": {
				        "requests": {
					    "cpu":    "100m",
					    "memory": "64Mi"
					},
				        "limits": {
					    "cpu":    "200m",
					    "memory": "128Mi"
					}
				    }
          		        },
          		        "profiler": {
          		            "images": {
          		                "java_profiler_image":   "pubrepo.guance.com/dataflux/async-profiler:0.1.0",
          		                "python_profiler_image": "pubrepo.guance.com/dataflux/py-spy:0.1.0"
          		            },
          		            "envs": {
          		                "DK_AGENT_PORT":       "9529",
          		                "DK_AGENT_HOST":       "datakit-service.datakit.svc"
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
					Environments: mapslice.MapSlice{
						{Key: "DD_AGENT_HOST", Value: "datakit-service.datakit.svc"},
						{Key: "DD_TRACE_AGENT_PORT", Value: "9529"},
					},
					envs: Envs{
						{"DD_AGENT_HOST", "datakit-service.datakit.svc"},
						{"DD_TRACE_AGENT_PORT", "9529"},
					},
				},
				Logfwd: ContainerConfig{
					Images: map[string]string{
						"logfwd_image": "pubrepo.guance.com/datakit/logfwd:1.5.8",
					},
					Resources: &ResourceRequirements{
						Requests: ResourceQuotaConfig{"100m", "64Mi"},
						Limits:   ResourceQuotaConfig{"200m", "128Mi"},
					},
				},
				Profiler: ContainerConfig{
					Images: map[string]string{
						"java_profiler_image":   "pubrepo.guance.com/dataflux/async-profiler:0.1.0",
						"python_profiler_image": "pubrepo.guance.com/dataflux/py-spy:0.1.0",
					},
					Environments: mapslice.MapSlice{
						{Key: "DK_AGENT_PORT", Value: "9529"},
						{Key: "DK_AGENT_HOST", Value: "datakit-service.datakit.svc"},
					},
					Resources: &ResourceRequirements{
						Requests: ResourceQuotaConfig{"100m", "64Mi"},
						Limits:   ResourceQuotaConfig{"500m", "512Mi"},
					},
					envs: Envs{
						{"DK_AGENT_PORT", "9529"},
						{"DK_AGENT_HOST", "datakit-service.datakit.svc"},
					},
				},
			},
		},
	}

	err = parseConfig(testcase.inCfgStr, testcase.inCfg)
	assert.NoError(t, err)

	t.Logf("result: %#v\n", testcase.inCfg)

	// mapslice.MapSlice has private variable 'index', skip Environments

	// global
	assert.Equal(t, testcase.outCfg.ServerListen, testcase.inCfg.ServerListen)
	assert.Equal(t, testcase.outCfg.LogLevel, testcase.inCfg.LogLevel)

	// ddtrace
	assert.Equal(t, testcase.outCfg.AdmissionInject.DDTrace.Images, testcase.inCfg.AdmissionInject.DDTrace.Images)
	assert.Equal(t, testcase.outCfg.AdmissionInject.DDTrace.envs, testcase.inCfg.AdmissionInject.DDTrace.envs)

	// logfwd
	assert.Equal(t, testcase.outCfg.AdmissionInject.Logfwd.Images, testcase.inCfg.AdmissionInject.Logfwd.Images)
	assert.Equal(t, testcase.outCfg.AdmissionInject.Logfwd.Resources, testcase.inCfg.AdmissionInject.Logfwd.Resources)

	// profiler
	assert.Equal(t, testcase.outCfg.AdmissionInject.Profiler.Images, testcase.inCfg.AdmissionInject.Profiler.Images)
	assert.Equal(t, testcase.outCfg.AdmissionInject.Profiler.Resources, testcase.inCfg.AdmissionInject.Profiler.Resources)
	assert.Equal(t, testcase.outCfg.AdmissionInject.Profiler.envs, testcase.inCfg.AdmissionInject.Profiler.envs)
}
