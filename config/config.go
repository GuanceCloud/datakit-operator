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

func initLog() {
	log = logger.SLogger("config")
}

type Configuration struct {
	ServerListen              string                 `json:"server_listen"`
	LogLevel                  string                 `json:"log_level"`
	AdmissionInject           AdmissionInjectConfig  `json:"admission_inject_v2"`
	AdmissionMutate           AdmissionMutateConfig  `json:"admission_mutate"`
	DeprecatedAdmissionInject DeprecatedInjectConfig `json:"admission_inject"`
}

func (c *Configuration) Setup() error {
	// 检查旧配置是否有效，如果有效则转换为新结构并替换
	if hasValidDeprecatedConfig(c) {
		log.Info("deprecated admission_inject config detected, converting to new structure and replacing admission_inject_v2")
		log.Info("priority: deprecated config takes precedence over new config, if deprecated config exists, it will be used")

		converted := convertDeprecatedToAdmissionInject(&c.DeprecatedAdmissionInject)

		// 替换 DDTraces 和 Logfwds（旧版优先）
		if isValidDeprecatedRule(&c.DeprecatedAdmissionInject.DDTraces) {
			c.AdmissionInject.DDTraces = converted.DDTraces
			log.Info("replaced admission_inject_v2.ddtrace with converted deprecated config")
		}
		if isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Logfwds) {
			c.AdmissionInject.Logfwds = converted.Logfwds
			log.Info("replaced admission_inject_v2.logfwd with converted deprecated config")
		}
	}

	if err := c.AdmissionInject.Setup(); err != nil {
		return err
	}
	if err := c.AdmissionMutate.Setup(); err != nil {
		return err
	}
	return nil
}

func hasValidDeprecatedConfig(c *Configuration) bool {
	return isValidDeprecatedRule(&c.DeprecatedAdmissionInject.DDTraces) ||
		isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Logfwds)
}

func isValidDeprecatedRule(rule *DeprecatedInjectRule) bool {
	if rule == nil {
		return false
	}

	hasImages := rule.Images != nil && len(rule.Images) > 0
	hasEnvironments := rule.Environments != nil && len(rule.Environments) > 0

	return hasImages || hasEnvironments
}

func initDefaultConfiguration() *Configuration {
	return &Configuration{
		ServerListen: ":9543",
		LogLevel:     "info",
	}
}

type AdmissionInjectConfig struct {
	DDTraces   InjectRules `json:"ddtraces"`
	Logfwds    InjectRules `json:"logfwds"`
	Flameshots InjectRules `json:"flameshots"`
}

func (c *AdmissionInjectConfig) Setup() error {
	c.DDTraces.Setup()
	c.Logfwds.Setup()
	c.Flameshots.Setup()
	return nil
}

type AdmissionMutateConfig struct {
	Loggings MutateRules `json:"loggings"`
}

func (c *AdmissionMutateConfig) Setup() error {
	c.Loggings.Setup()
	return nil
}
