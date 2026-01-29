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
		log.Info("Deprecated admission_inject configuration detected, converting to admission_inject_v2 structure")
		log.Info("Conversion policy: deprecated configuration takes precedence over new configuration")

		converted := convertDeprecatedToAdmissionInject(&c.DeprecatedAdmissionInject)

		// 替换 DDTraces、Profilers 和 Logfwds（旧版优先）
		if isValidDeprecatedRule(&c.DeprecatedAdmissionInject.DDTrace) {
			c.AdmissionInject.DDTraces = converted.DDTraces
			log.Info("Converted deprecated ddtrace configuration to admission_inject_v2.ddtraces")
		}
		if isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Logfwd) {
			c.AdmissionInject.Logfwds = converted.Logfwds
			log.Info("Converted deprecated logfwd configuration to admission_inject_v2.logfwds")
		}
		if isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Profiler) {
			c.AdmissionInject.Profilers = converted.Profilers
			log.Info("Converted deprecated profiler configuration to admission_inject_v2.profilers")
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
	return isValidDeprecatedRule(&c.DeprecatedAdmissionInject.DDTrace) ||
		isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Logfwd) ||
		isValidDeprecatedRule(&c.DeprecatedAdmissionInject.Profiler)
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
	Profilers  InjectRules `json:"profilers"`
}

func (c *AdmissionInjectConfig) Setup() error {
	c.DDTraces.Setup()
	c.Logfwds.Setup()
	c.Flameshots.Setup()
	c.Profilers.Setup()
	return nil
}

type AdmissionMutateConfig struct {
	Loggings MutateRules `json:"loggings"`
}

func (c *AdmissionMutateConfig) Setup() error {
	c.Loggings.Setup()
	return nil
}
