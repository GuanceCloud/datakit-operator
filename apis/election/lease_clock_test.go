// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clocktesting "k8s.io/utils/clock/testing"
)

func TestElectionWallClockJumpDoesNotChangeLeaseLifetime(t *testing.T) {
	for _, jump := range []time.Duration{5 * time.Second, -time.Hour} {
		t.Run(jump.String(), func(t *testing.T) {
			c, _, clock, router := newElectionTest(t)
			first := requestElection(t, router, "/v1/dk-election?token=secret&id=a", 200)
			assert.Equal(t, StatusSuccess, first.Status)

			// Wall time jumps, while elapsed time and DataKit's safe lease do not.
			c.now = func() time.Time { return clock.Now().Add(jump) }
			clock.Step(25 * time.Second)
			assert.Less(t, 25, first.LeaseDuration-first.Interval)
			got := requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
			assert.Equal(t, StatusDefeat, got.Status, "a still has two seconds of safe collection time")
			metrics := httptest.NewRecorder()
			router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			assert.Contains(t, metrics.Body.String(), "datakit_operator_election_active_scopes 1\n")
			assert.Contains(t, metrics.Body.String(), "datakit_operator_election_lease_remaining_seconds 5\n")

			clock.Step(5 * time.Second)
			metrics = httptest.NewRecorder()
			router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			assert.Contains(t, metrics.Body.String(), "datakit_operator_election_active_scopes 0\n")
			got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
			assert.Equal(t, StatusSuccess, got.Status, "wall time must not delay takeover either")
			assert.EqualValues(t, 2, got.Epoch)
		})
	}
}

func TestElectionRestartObservesPersistedLeaseBeforeTakeover(t *testing.T) {
	client := newFakeKubernetesLeaseClient()
	defer client.broadcaster.Shutdown()
	lastSuccess := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	oldClock := clocktesting.NewFakeClock(lastSuccess)
	old, stop := startTestCoordinator(t, client, oldClock)
	first := requestElection(t, newContractRouter(t, old), "/v1/dk-election?token=secret&id=a", 200)
	assert.Equal(t, StatusSuccess, first.Status)
	stop()
	assert.Eventually(t, func() bool { return !old.cache.Ready() }, time.Second, time.Millisecond)

	// Only the replacement is running. Its node is five seconds ahead, but
	// the original holder's local safety deadline is still two seconds away.
	replacementClock := clocktesting.NewFakeClock(lastSuccess.Add(30 * time.Second))
	replacement, _ := startTestCoordinator(t, client, replacementClock)
	router := newContractRouter(t, replacement)
	got := requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
	assert.Equal(t, StatusDefeat, got.Status)
	persisted, err := client.Get(context.Background(), leaseName("secret", ""), metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, "a", holder(persisted))
	assert.EqualValues(t, 1, epoch(persisted))

	reads := client.store.getCalls.Load()
	for range 29 {
		replacementClock.Step(time.Second)
		got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
		assert.Equal(t, StatusDefeat, got.Status)
	}
	assert.Equal(t, reads, client.store.getCalls.Load(), "observing an existing Lease must retain the cache fast path")
	replacementClock.Step(time.Second)
	got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
	assert.Equal(t, StatusSuccess, got.Status, "the authoritative reread must not restart the observation period")
	assert.EqualValues(t, 2, got.Epoch)
}

func TestElectionRenewalAfterWallClockJump(t *testing.T) {
	for _, jump := range []time.Duration{5 * time.Second, -time.Hour} {
		t.Run(jump.String(), func(t *testing.T) {
			c, _, clock, router := newElectionTest(t)
			requestElection(t, router, "/v1/dk-election?token=secret&id=a", 200)
			c.now = func() time.Time { return clock.Now().Add(jump) }
			clock.Step(25 * time.Second)
			got := requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=a", 200)
			assert.Equal(t, StatusSuccess, got.Status)
			assert.EqualValues(t, 1, got.Epoch)

			clock.Step(29 * time.Second)
			got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
			assert.Equal(t, StatusDefeat, got.Status)
			clock.Step(time.Second)
			got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
			assert.Equal(t, StatusSuccess, got.Status)
			assert.EqualValues(t, 2, got.Epoch)
			got = requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=a", 200)
			assert.Equal(t, StatusDefeat, got.Status)
		})
	}
}

func TestElectionStaleCacheDoesNotRestartLeaseObservation(t *testing.T) {
	c, store, clock, router := newElectionTest(t)
	requestElection(t, router, "/v1/dk-election?token=secret&id=a", 200)
	name := leaseName("secret", "")
	lease, err := store.Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)
	stale := newFakeLeaseBackend()
	stale.leases[name] = lease.DeepCopy()
	c.cache = &fakeLeaseCache{backend: stale, ready: true}

	clock.Step(20 * time.Second)
	got := requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=a", 200)
	assert.Equal(t, StatusSuccess, got.Status)
	clock.Step(11 * time.Second)
	for range 19 {
		// Each request sees the original cached RV, followed by the renewed
		// authoritative RV. Neither repeated observation may refresh its timer.
		got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
		assert.Equal(t, StatusDefeat, got.Status)
		clock.Step(time.Second)
	}
	got = requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
	assert.Equal(t, StatusSuccess, got.Status)
	assert.EqualValues(t, 2, got.Epoch)
	got = requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=b", 200)
	assert.Equal(t, StatusSuccess, got.Status, "the stale holder must not reject the new holder's heartbeat")
}
