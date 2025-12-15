// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package mutator

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

var (
	loggingMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) string {
		found, rule := config.Cfg.AdmissionMutate.Loggings.Matches(ns, labels)
		if !found || rule == nil {
			return ""
		}
		return rule.Config
	}
)
