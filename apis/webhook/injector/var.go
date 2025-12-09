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
	null language = ""
	java language = "java"
)

var (
	ddtraceMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
		found, rule := config.Cfg.AdmissionInject.DDTrace.Matches(ns, labels)
		if !found || rule == nil {
			return ""
		}
	}

	logfwdMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
	}

	flameshotMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) (bool, *config.InjectRule) {
	}
)
