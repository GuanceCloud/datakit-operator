// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"sync"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
)

type observationSource int

const (
	fromCache observationSource = iota
	fromStore
)

type leaseObservation struct {
	resourceVersion string
	observedAt      time.Duration
	known           bool
}

type leaseObservations struct {
	mu     sync.Mutex
	leases map[string][2]leaseObservation
}

// remaining measures a Lease version's lifetime using local monotonic elapsed
// time. A version first seen after restart receives a full observation period;
// its persisted RenewTime may have been written by a node with a different clock.
//
// Cache and API observations are kept separately: a lagging cache must not keep
// replacing the newer API observation and restarting its expiry timer. When
// both sources see the same version, they share its first observation time.
func (o *leaseObservations) remaining(lease *coordinationv1.Lease, source observationSource, elapsed time.Duration) time.Duration {
	if holder(lease) == "" || lease.Spec.RenewTime == nil ||
		lease.Spec.LeaseDurationSeconds == nil || *lease.Spec.LeaseDurationSeconds <= 0 {
		return 0
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if o.leases == nil {
		o.leases = make(map[string][2]leaseObservation)
	}
	observations := o.leases[lease.Name]
	current, other := &observations[source], observations[1-source]
	if !current.known || current.resourceVersion != lease.ResourceVersion {
		if other.known && other.resourceVersion == lease.ResourceVersion {
			*current = other
		} else {
			*current = leaseObservation{resourceVersion: lease.ResourceVersion, observedAt: elapsed, known: true}
		}
		o.leases[lease.Name] = observations
	}
	duration := time.Duration(*lease.Spec.LeaseDurationSeconds) * time.Second
	return max(0, duration-(elapsed-current.observedAt))
}

func (c *Coordinator) remaining(lease *coordinationv1.Lease, source observationSource) time.Duration {
	return c.observations.remaining(lease, source, c.elapsed())
}
