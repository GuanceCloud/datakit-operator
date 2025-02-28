package config

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

func defaultResourceRequirements() *ResourceRequirements {
	return &ResourceRequirements{
		Requests: ResourceQuotaConfig{"100m", "64Mi"},
		Limits:   ResourceQuotaConfig{"500m", "512Mi"},
	}
}

type ResourceRequirements struct {
	Requests ResourceQuotaConfig
	Limits   ResourceQuotaConfig
}

type ResourceQuotaConfig struct {
	CPU    string
	Memory string
}

func (r ResourceRequirements) Verify() error {
	if r.Requests.CPU != "" {
		if _, err := resource.ParseQuantity(r.Requests.CPU); err != nil {
			return fmt.Errorf("cannot parse '%s' cpu request, err %w", r.Requests.CPU, err)
		}
	}

	if r.Requests.Memory != "" {
		if _, err := resource.ParseQuantity(r.Requests.Memory); err != nil {
			return fmt.Errorf("cannot parse '%s' memory request, err %w", r.Requests.Memory, err)
		}
	}

	if r.Limits.CPU != "" {
		if _, err := resource.ParseQuantity(r.Limits.CPU); err != nil {
			return fmt.Errorf("cannot parse '%s' cpu limit, err %w", r.Limits.CPU, err)
		}
	}

	if r.Limits.Memory != "" {
		if _, err := resource.ParseQuantity(r.Limits.Memory); err != nil {
			return fmt.Errorf("cannot parse '%s' memory limit, err %w", r.Limits.Memory, err)
		}
	}

	return nil
}
