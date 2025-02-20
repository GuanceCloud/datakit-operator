package mutator

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

var (
	loggingMatchNamespaceForConfig = func(ns string) string {
		return config.Cfg.AdmissionMutate.Loggings.MatchNamespace(ns)
	}

	loggingMatchLabelsForConfig = func(labels map[string]string) string {
		return config.Cfg.AdmissionMutate.Loggings.MatchLabels(labels)
	}
)
