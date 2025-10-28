// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package config

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"

	loggingv1alpha1 "gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/pkg/apis/datakits/v1alpha1"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/labels"
)

type loggingConfigCache struct {
	mu      sync.RWMutex
	configs []*cachedConfig
}

type cachedConfig struct {
	name            string
	podTargetLabels []string
	configs         string

	namespaceRegex   *regexp.Regexp
	podRegex         *regexp.Regexp
	podLabelSelector labels.Selector
}

var globalCache = &loggingConfigCache{
	configs: []*cachedConfig{},
}

func (c *loggingConfigCache) addOrUpdateConfig(name string, config *loggingv1alpha1.ClusterLoggingConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	configsJSON, err := json.Marshal(config.Spec.Configs)
	if err != nil {
		log.Errorf("failed to marshal configs to JSON for config %s: %v", name, err)
		return
	}

	cached := &cachedConfig{
		name:            name,
		podTargetLabels: config.Spec.PodTargetLabels,
		configs:         string(configsJSON),
	}

	sel := config.Spec.Selector
	if sel.NamespaceRegex != "" {
		if r, err := regexp.Compile(sel.NamespaceRegex); err == nil {
			cached.namespaceRegex = r
		} else {
			log.Errorf("failed to compile namespace regex %s for config %s: %v", sel.NamespaceRegex, name, err)
		}
	}

	if sel.PodRegex != "" {
		if r, err := regexp.Compile(sel.PodRegex); err == nil {
			cached.podRegex = r
		} else {
			log.Errorf("failed to compile pod regex %s for config %s: %v", sel.PodRegex, name, err)
		}
	}

	if sel.PodLabelSelector != "" {
		if s, err := labels.Parse(sel.PodLabelSelector); err == nil {
			cached.podLabelSelector = s
		} else {
			log.Errorf("failed to compile pod labelSelector %s for config %s: %v", sel.PodLabelSelector, name, err)
		}
	}

	found := false
	for i, existing := range c.configs {
		if existing.name == name {
			c.configs[i] = cached
			found = true
			break
		}
	}
	if !found {
		c.configs = append(c.configs, cached)
	}

	log.Infof("added/updated logging config: %s", name)
}

func (c *loggingConfigCache) removeConfig(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i, cached := range c.configs {
		if cached.name == name {
			c.configs = append(c.configs[:i], c.configs[i+1:]...)
			log.Infof("removed logging config: %s", name)
			break
		}
	}
}

func (c *loggingConfigCache) findMatchingConfigs(namespace, podName, podLabels string) (name string, configs string, podTargetLabels []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, cached := range c.configs {
		if c.matchesSelector(cached, namespace, podName, podLabels) {
			return cached.name, cached.configs, cached.podTargetLabels
		}
	}
	return "", "", nil
}

func (c *loggingConfigCache) matchesSelector(cached *cachedConfig, namespace, podName, podLabels string) bool {
	if cached.namespaceRegex != nil && !cached.namespaceRegex.MatchString(namespace) {
		return false
	}
	if cached.podRegex != nil && !cached.podRegex.MatchString(podName) {
		return false
	}
	if cached.podLabelSelector != nil && podLabels != "" {
		podLabelsMap := parseLabels(podLabels)
		if !cached.podLabelSelector.Matches(labels.Set(podLabelsMap)) {
			return false
		}
	}
	return true
}

func AddOrUpdateConfig(name string, config *loggingv1alpha1.ClusterLoggingConfig) {
	globalCache.addOrUpdateConfig(name, config)
}

func RemoveConfig(name string) {
	globalCache.removeConfig(name)
}

func FindMatchingConfigs(namespace, podName, podLabels string) (name string, configs string, podTargetLabels []string) {
	return globalCache.findMatchingConfigs(namespace, podName, podLabels)
}

func parseLabels(s string) map[string]string {
	if s == "" {
		return map[string]string{}
	}

	tags := map[string]string{}

	parts := strings.Split(s, ",")
	for _, p := range parts {
		arr := strings.Split(p, "=")
		if len(arr) != 2 {
			continue
		}

		tags[arr[0]] = arr[1]
	}

	return tags
}
