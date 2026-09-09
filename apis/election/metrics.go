// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
)

type electionOperation int

const (
	operationCampaign electionOperation = iota
	operationHeartbeat
	operationInformer
)

const (
	transitionFirstAcquire = iota
	transitionLeaseExpired
)

var (
	operationNames  = [...]string{"campaign", "heartbeat", "informer"}
	transitionNames = [...]string{"first_acquire", "lease_expired"}
	resultNames     = [...]string{StatusSuccess, StatusDefeat, StatusError}
)

type electionCounters struct {
	requests      [2][3]uint64
	transitions   [2]uint64
	storageErrors [3]uint64
	casConflicts  [2]uint64
}

type electionMetrics struct {
	mu sync.Mutex
	electionCounters
}

func (m *electionMetrics) recordRequest(operation electionOperation, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for index, name := range resultNames {
		if result == name {
			m.requests[operation][index]++
			return
		}
	}
}

func (m *electionMetrics) recordTransition(reason int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transitions[reason]++
}

func (m *electionMetrics) recordStorageError(operation electionOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.storageErrors[operation]++
}

func (m *electionMetrics) recordCASConflict(operation electionOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.casConflicts[operation]++
}

func (c *Coordinator) renderMetrics() string {
	leases, err := c.cache.List(labels.Everything())
	if err != nil {
		_ = c.storageError(operationInformer, "metrics_cache", c.namespace, err)
	}
	active := 0
	var remaining, age time.Duration
	now := c.now()
	for _, lease := range leases {
		lifetime := c.remaining(lease, fromCache)
		if lifetime <= 0 {
			continue
		}
		if active == 0 || lifetime < remaining {
			remaining = lifetime
		}
		if lease.Spec.AcquireTime != nil {
			age = max(age, now.Sub(lease.Spec.AcquireTime.Time))
		}
		active++
	}

	c.metrics.mu.Lock()
	counts := c.metrics.electionCounters
	c.metrics.mu.Unlock()

	var out strings.Builder
	out.WriteString("# HELP datakit_operator_election_requests_total Election HTTP requests by operation and result.\n# TYPE datakit_operator_election_requests_total counter\n")
	for operation, values := range counts.requests {
		for result, count := range values {
			fmt.Fprintf(&out, "datakit_operator_election_requests_total{operation=%q,result=%q} %d\n", operationNames[operation], resultNames[result], count)
		}
	}
	out.WriteString("# HELP datakit_operator_election_transitions_total Collection Leader transitions by reason.\n# TYPE datakit_operator_election_transitions_total counter\n")
	for reason, count := range counts.transitions {
		fmt.Fprintf(&out, "datakit_operator_election_transitions_total{reason=%q} %d\n", transitionNames[reason], count)
	}
	out.WriteString("# HELP datakit_operator_election_storage_errors_total Kubernetes Lease storage errors by operation.\n# TYPE datakit_operator_election_storage_errors_total counter\n")
	for operation, count := range counts.storageErrors {
		fmt.Fprintf(&out, "datakit_operator_election_storage_errors_total{operation=%q} %d\n", operationNames[operation], count)
	}
	out.WriteString("# HELP datakit_operator_election_cas_conflicts_total Kubernetes Lease resourceVersion conflicts by operation.\n# TYPE datakit_operator_election_cas_conflicts_total counter\n")
	for operation, count := range counts.casConflicts {
		fmt.Fprintf(&out, "datakit_operator_election_cas_conflicts_total{operation=%q} %d\n", operationNames[operation], count)
	}
	out.WriteString("# HELP datakit_operator_election_active_scopes Number of active election scopes in the Lease cache.\n# TYPE datakit_operator_election_active_scopes gauge\n")
	fmt.Fprintf(&out, "datakit_operator_election_active_scopes %d\n", active)
	out.WriteString("# HELP datakit_operator_election_lease_remaining_seconds Minimum remaining lifetime among active election Leases.\n# TYPE datakit_operator_election_lease_remaining_seconds gauge\n")
	fmt.Fprintf(&out, "datakit_operator_election_lease_remaining_seconds %.0f\n", remaining.Seconds())
	out.WriteString("# HELP datakit_operator_election_lease_age_seconds Age of the oldest active election Lease.\n# TYPE datakit_operator_election_lease_age_seconds gauge\n")
	fmt.Fprintf(&out, "datakit_operator_election_lease_age_seconds %.0f\n", age.Seconds())
	return out.String()
}
