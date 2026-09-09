// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	authorizationv1 "k8s.io/api/authorization/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	typedauthorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	typedcoordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	clocktesting "k8s.io/utils/clock/testing"
)

type fakeAccessReviews struct {
	typedauthorizationv1.SelfSubjectAccessReviewInterface
	denied string
	err    error
}

func (r *fakeAccessReviews) Create(_ context.Context, review *authorizationv1.SelfSubjectAccessReview, _ metav1.CreateOptions) (*authorizationv1.SelfSubjectAccessReview, error) {
	attributes := review.Spec.ResourceAttributes
	review.Status.Allowed = attributes.Namespace == "operator-system" && attributes.Group == "coordination.k8s.io" &&
		attributes.Resource == "leases" && attributes.Verb != r.denied
	return review, r.err
}

func TestElectionStatusChecksPermissionsAndCache(t *testing.T) {
	for _, failure := range []string{"", "get", "list", "watch", "create", "update", "api", "cache"} {
		t.Run(failure, func(t *testing.T) {
			c, store, _, router := newElectionTest(t)
			reviews := &fakeAccessReviews{denied: failure}
			c.accessReviews = reviews
			wantCode, wantBody := 503, `{"content":{"status":"error","error_code":"rbac_forbidden"}}`
			switch failure {
			case "":
				wantCode, wantBody = 200, `{"content":{"status":"ready"}}`
			case "api":
				reviews.err = context.DeadlineExceeded
				wantBody = `{"content":{"status":"error","error_code":"storage_unavailable"}}`
			case "cache":
				c.cache.(*fakeLeaseCache).ready = false
				wantBody = `{"content":{"status":"error","error_code":"cache_not_ready"}}`
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/dk-election/status", nil))
			assert.Equal(t, wantCode, recorder.Code)
			assert.JSONEq(t, wantBody, recorder.Body.String())
			assert.Zero(t, store.createCalls.Load()+store.updateCalls.Load(), "probe must not acquire or renew a Lease")
		})
	}
}

func TestElectionWithoutOperatorNamespaceStaysUnavailable(t *testing.T) {
	// Nil clients ensure disabled election never accesses Kubernetes, even for its status probe.
	c := NewCoordinator(nil, nil, "")
	c.Start(t.Context())
	router := newContractRouter(t, c)
	for _, path := range []string{"/v1/dk-election", "/v1/dk-election/heartbeat"} {
		got := requestElection(t, router, path+"?token=secret&id=a", 503)
		assert.Equal(t, StatusError, got.Status)
		assert.Equal(t, "cache_not_ready", got.ErrorCode)
	}
	status := httptest.NewRecorder()
	router.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/v1/dk-election/status", nil))
	assert.Equal(t, 503, status.Code)
	assert.JSONEq(t, `{"content":{"status":"error","error_code":"cache_not_ready"}}`, status.Body.String())
	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, 200, metrics.Code)
	assert.Contains(t, metrics.Body.String(), "datakit_operator_election_active_scopes 0\n")
}

type fakeKubernetesLeaseClient struct {
	typedcoordinationv1.LeaseInterface
	backend      *fakeLeaseBackend
	store        *fakeLeaseStore
	broadcaster  *watch.Broadcaster
	unavailable  atomic.Bool
	listFailures atomic.Int64
}

func newFakeKubernetesLeaseClient() *fakeKubernetesLeaseClient {
	backend := newFakeLeaseBackend()
	return &fakeKubernetesLeaseClient{
		backend:     backend,
		store:       &fakeLeaseStore{backend: backend},
		broadcaster: watch.NewBroadcaster(100, watch.WaitIfChannelFull),
	}
}

func (c *fakeKubernetesLeaseClient) Get(ctx context.Context, name string, _ metav1.GetOptions) (*coordinationv1.Lease, error) {
	return c.store.Get(ctx, name, metav1.GetOptions{})
}

func (c *fakeKubernetesLeaseClient) Create(ctx context.Context, lease *coordinationv1.Lease, _ metav1.CreateOptions) (*coordinationv1.Lease, error) {
	created, err := c.store.Create(ctx, lease, metav1.CreateOptions{})
	if err == nil {
		_ = c.broadcaster.Action(watch.Added, created.DeepCopy())
	}
	return created, err
}

func (c *fakeKubernetesLeaseClient) Update(ctx context.Context, lease *coordinationv1.Lease, _ metav1.UpdateOptions) (*coordinationv1.Lease, error) {
	updated, err := c.store.Update(ctx, lease, metav1.UpdateOptions{})
	if err == nil {
		_ = c.broadcaster.Action(watch.Modified, updated.DeepCopy())
	}
	return updated, err
}

func (c *fakeKubernetesLeaseClient) List(_ context.Context, _ metav1.ListOptions) (*coordinationv1.LeaseList, error) {
	if c.unavailable.Load() {
		c.listFailures.Add(1)
		return nil, context.DeadlineExceeded
	}
	c.backend.mu.Lock()
	defer c.backend.mu.Unlock()
	list := &coordinationv1.LeaseList{TypeMeta: metav1.TypeMeta{APIVersion: "coordination.k8s.io/v1", Kind: "LeaseList"}}
	for _, lease := range c.backend.leases {
		if lease.Labels[managedLeaseLabel] == "true" {
			list.Items = append(list.Items, *lease.DeepCopy())
		}
	}
	return list, nil
}

func (c *fakeKubernetesLeaseClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	if c.unavailable.Load() {
		return nil, context.DeadlineExceeded
	}
	return c.broadcaster.Watch()
}

func startTestCoordinator(t *testing.T, client *fakeKubernetesLeaseClient, clock *clocktesting.FakeClock) (*Coordinator, context.CancelFunc) {
	t.Helper()
	started := clock.Now()
	c := NewCoordinator(client, &fakeAccessReviews{}, "operator-system")
	c.now = clock.Now
	c.elapsed = func() time.Duration { return clock.Since(started) }
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	c.Start(ctx)
	if !assert.Eventually(t, c.cache.Ready, 5*time.Second, time.Millisecond) {
		t.FailNow()
	}
	return c, cancel
}

func TestCoordinatorRestartRetainsLease(t *testing.T) {
	client := newFakeKubernetesLeaseClient()
	defer client.broadcaster.Shutdown()
	clock := clocktesting.NewFakeClock(time.Now())
	first, stop := startTestCoordinator(t, client, clock)
	got := requestElection(t, newContractRouter(t, first), "/v1/dk-election?token=secret&id=a", 200)
	assert.Equal(t, StatusSuccess, got.Status)
	stop()
	assert.Eventually(t, func() bool { return !first.cache.Ready() }, time.Second, 10*time.Millisecond)

	clock.Step(20 * time.Second)
	second, _ := startTestCoordinator(t, client, clock)
	got = requestElection(t, newContractRouter(t, second), "/v1/dk-election/heartbeat?token=secret&id=a", 200)
	assert.Equal(t, Result{Status: StatusSuccess, ID: "a", IncumbencyID: "a", Interval: 3, LeaseDuration: 30, Epoch: 1}, got)
	leases, err := client.List(context.Background(), metav1.ListOptions{})
	assert.NoError(t, err)
	if !assert.Len(t, leases.Items, 1) {
		return
	}
	assert.Equal(t, "operator-system", leases.Items[0].Namespace)
	persisted, err := json.Marshal(leases.Items)
	assert.NoError(t, err)
	assert.NotContains(t, string(persisted), "secret")
}

func TestWatchFailureMakesCoordinatorUnavailable(t *testing.T) {
	client := newFakeKubernetesLeaseClient()
	defer client.broadcaster.Shutdown()
	c := NewCoordinator(client, &fakeAccessReviews{}, "operator-system")
	router := newContractRouter(t, c)
	readyStatus := func() int {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/dk-election/status", nil))
		return recorder.Code
	}
	assert.Equal(t, 503, readyStatus())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)
	assert.Eventually(t, func() bool { return readyStatus() == 200 }, 5*time.Second, 10*time.Millisecond)
	requestElection(t, router, "/v1/dk-election?token=secret&id=a", 200)

	client.unavailable.Store(true)
	assert.NoError(t, client.broadcaster.Action(watch.Error, &metav1.Status{
		Status: metav1.StatusFailure, Reason: metav1.StatusReasonExpired, Code: 410,
	}))
	assert.Eventually(t, func() bool { return client.listFailures.Load() > 0 }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, 503, readyStatus())
	got := requestElection(t, router, "/v1/dk-election?token=secret&id=b", 503)
	assert.Equal(t, StatusError, got.Status)
}
