// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

type language string

const (
	java   language = "java"
	python language = "python"
	php    language = "php"
	nodejs language = "nodejs"
	golang language = "golang"

	enableEnvFieldRef = true
)

var (
	ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
		return config.Cfg.AdmissionInject.DDTraces.Matches(ns, labels, "")
	}

	ddtraceMatchAllNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, []*config.InjectRule) {
		return config.Cfg.AdmissionInject.DDTraces.MatchesAll(ns, labels)
	}

	logfwdMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
		return config.Cfg.AdmissionInject.Logfwds.Matches(ns, labels, "")
	}

	flameshotMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
		return config.Cfg.AdmissionInject.Flameshots.Matches(ns, labels, "")
	}

	profilerMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
		return config.Cfg.AdmissionInject.Profilers.Matches(ns, labels, "")
	}
)
