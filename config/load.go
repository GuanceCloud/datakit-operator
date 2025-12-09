// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadConfigWithEnv 从环境变量加载配置
// 优先级：
// 1) ENV_CONFIG_FILE: JSON 配置文件路径
// 2) ENV_JSON_CONFIG: 原始 JSON 字符串
func LoadConfigWithEnv() error {
	initLog()
	log.Info("loading config..")

	if path := os.Getenv("ENV_CONFIG_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read config file failed: %w", err)
		}
		if err := parseConfig(string(b), Cfg); err != nil {
			return err
		}
		return nil
	}

	cfgStr := os.Getenv("ENV_JSON_CONFIG")
	if err := parseConfig(cfgStr, Cfg); err != nil {
		return err
	}
	return nil
}

func parseConfig(cfgStr string, c *Configuration) error {
	if cfgStr != "" {
		if err := json.Unmarshal([]byte(cfgStr), c); err != nil {
			return fmt.Errorf("unable to unmarshal config: %w", err)
		}
	}

	// 调用 Configuration 的 Setup 方法进行初始化和转换
	return c.Setup()
}