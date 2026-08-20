// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package injector

import (
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/manager"
	corev1 "k8s.io/api/core/v1"
)

// InjectTracingToPod selects at most one configured tracing auto-injection
// provider. DDTrace has priority when both providers select the same Pod; a
// failed DDTrace injection does not fall back to OTel.
func InjectTracingToPod(namespace, parent string, pod *corev1.Pod) (bool, error) {
	if pod != nil && manager.NewContainerManager(pod).ContainsInitContainer(ddtraceInitContainerName) {
		return InjectDDTraceToPod(namespace, parent, pod)
	}

	ddtraceSelected, changed, err := injectDDTraceToPod(namespace, parent, pod)
	if err != nil || ddtraceSelected {
		return changed, err
	}

	otelChanged, err := InjectOTelToPod(namespace, parent, pod)
	return changed || otelChanged, err
}
