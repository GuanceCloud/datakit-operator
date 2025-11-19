// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package logging

import (
	"encoding/json"
	"net/http"

	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/logging/config"
)

type Response struct {
	Name            string   `json:"name,omitempty"`
	PodTargetLabels []string `json:"pod_target_labels,omitempty"`
	Configs         string   `json:"configs,omitempty"`
	Error           string   `json:"error,omitempty"`
}

func HandleConfigs(w http.ResponseWriter, r *http.Request) {
	log.Debugf("config request received: method=%s, namespace=%s, pod_name=%s", r.Method, r.URL.Query().Get("namespace"), r.URL.Query().Get("pod_name"))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if err := json.NewEncoder(w).Encode(Response{Error: "method not allowed"}); err != nil {
			log.Warnf("failed to encode response: %v", err)
		}
		return
	}

	namespace := r.URL.Query().Get("namespace")
	podName := r.URL.Query().Get("pod_name")
	podLabelsStr := r.URL.Query().Get("pod_labels")

	if namespace == "" || podName == "" {
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(Response{Error: "namespace, pod_name parameters are required"}); err != nil {
			log.Warnf("failed to encode response: %v", err)
		}
		return
	}

	// 从 cache 中查找匹配的配置
	name, configs, podTargetLabels := config.FindMatchingConfigs(namespace, podName, podLabelsStr)

	if configs == "" {
		log.Infof("no matching config found: namespace=%s, pod_name=%s", namespace, podName)
		w.WriteHeader(http.StatusNotFound)
		if err := json.NewEncoder(w).Encode(Response{Error: "no matching config found"}); err != nil {
			log.Warnf("failed to encode response: %v", err)
		}
		return
	}

	log.Infof("matching config found: name=%s", name)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(Response{Name: name, PodTargetLabels: podTargetLabels, Configs: configs}); err != nil {
		log.Warnf("failed to encode response: %v", err)
	}
}
