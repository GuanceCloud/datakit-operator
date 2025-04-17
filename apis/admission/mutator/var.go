package mutator

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/config"
)

var (
	loggingMatchNamespaceOrLabelsForConfig = func(ns string, labels map[string]string) string {
		return config.Cfg.AdmissionMutate.Loggings.Matches(ns, labels)
	}
)
