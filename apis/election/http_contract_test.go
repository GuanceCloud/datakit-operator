// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clocktesting "k8s.io/utils/clock/testing"
)

type fakeLeaseBackend struct {
	mu     sync.Mutex
	leases map[string]*coordinationv1.Lease
	nextRV int64
}

func newFakeLeaseBackend() *fakeLeaseBackend {
	return &fakeLeaseBackend{leases: make(map[string]*coordinationv1.Lease)}
}

type fakeLeaseStore struct {
	backend      *fakeLeaseBackend
	getErr       error
	createErr    error
	updateErr    error
	beforeUpdate func(*coordinationv1.Lease)
	getCalls     atomic.Int64
	createCalls  atomic.Int64
	updateCalls  atomic.Int64
}

func (s *fakeLeaseStore) Get(_ context.Context, name string, _ metav1.GetOptions) (*coordinationv1.Lease, error) {
	s.getCalls.Add(1)
	if s.getErr != nil {
		return nil, s.getErr
	}
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()
	lease, ok := s.backend.leases[name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, name)
	}
	return lease.DeepCopy(), nil
}

func (s *fakeLeaseStore) Create(_ context.Context, lease *coordinationv1.Lease, _ metav1.CreateOptions) (*coordinationv1.Lease, error) {
	s.createCalls.Add(1)
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()
	if _, ok := s.backend.leases[lease.Name]; ok {
		return nil, apierrors.NewAlreadyExists(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, lease.Name)
	}
	s.backend.nextRV++
	created := lease.DeepCopy()
	created.ResourceVersion = strconv.FormatInt(s.backend.nextRV, 10)
	s.backend.leases[lease.Name] = created
	return created.DeepCopy(), nil
}

func (s *fakeLeaseStore) Update(_ context.Context, lease *coordinationv1.Lease, _ metav1.UpdateOptions) (*coordinationv1.Lease, error) {
	s.updateCalls.Add(1)
	if s.beforeUpdate != nil {
		s.beforeUpdate(lease.DeepCopy())
	}
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	s.backend.mu.Lock()
	defer s.backend.mu.Unlock()
	current, ok := s.backend.leases[lease.Name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, lease.Name)
	}
	if current.ResourceVersion != lease.ResourceVersion {
		return nil, apierrors.NewConflict(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, lease.Name, nil)
	}
	s.backend.nextRV++
	updated := lease.DeepCopy()
	updated.ResourceVersion = strconv.FormatInt(s.backend.nextRV, 10)
	s.backend.leases[lease.Name] = updated
	return updated.DeepCopy(), nil
}

type fakeLeaseCache struct {
	backend *fakeLeaseBackend
	ready   bool
	err     error
}

func (c *fakeLeaseCache) Ready() bool { return c.ready }

func (c *fakeLeaseCache) Start(context.Context) {}

func (c *fakeLeaseCache) Get(name string) (*coordinationv1.Lease, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.backend.mu.Lock()
	defer c.backend.mu.Unlock()
	lease, ok := c.backend.leases[name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "coordination.k8s.io", Resource: "leases"}, name)
	}
	return lease.DeepCopy(), nil
}

func (c *fakeLeaseCache) List(labels.Selector) ([]*coordinationv1.Lease, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.backend.mu.Lock()
	defer c.backend.mu.Unlock()
	result := make([]*coordinationv1.Lease, 0, len(c.backend.leases))
	for _, lease := range c.backend.leases {
		result = append(result, lease.DeepCopy())
	}
	return result, nil
}

func newTestCoordinator(store leaseStore, cache leaseCache, clock *clocktesting.FakeClock) *Coordinator {
	started := clock.Now()
	return &Coordinator{
		store: store, cache: cache, now: clock.Now, namespace: "operator-system",
		elapsed: func() time.Duration { return clock.Since(started) },
	}
}

func newContractRouter(t *testing.T, coordinator *Coordinator) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	coordinator.RegisterRoutes(router)
	return router
}

func newElectionTest(t *testing.T) (*Coordinator, *fakeLeaseStore, *clocktesting.FakeClock, *gin.Engine) {
	t.Helper()
	backend := newFakeLeaseBackend()
	store := &fakeLeaseStore{backend: backend}
	clock := clocktesting.NewFakeClock(time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC))
	coordinator := newTestCoordinator(store, &fakeLeaseCache{backend: backend, ready: true}, clock)
	return coordinator, store, clock, newContractRouter(t, coordinator)
}

func requestElection(t *testing.T, router http.Handler, path string, wantCode int) Result {
	t.Helper()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))
	assert.Equal(t, wantCode, recorder.Code, path)
	var response ResponseEnvelope
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Content
}

func TestElectionLifecycle(t *testing.T) {
	_, _, clock, router := newElectionTest(t)
	tests := []struct {
		step                         time.Duration
		endpoint, id, status, holder string
		epoch                        int64
	}{
		{0, "", "a", StatusSuccess, "a", 1},
		{0, "", "b", StatusDefeat, "a", 1},
		{0, "", "a", StatusDefeat, "a", 1},
		{20 * time.Second, "/heartbeat", "a", StatusSuccess, "a", 1},
		{11 * time.Second, "", "b", StatusDefeat, "a", 1}, // Renewal extended the original deadline.
		{0, "/heartbeat", "b", StatusDefeat, "a", 1},
		{20 * time.Second, "/heartbeat", "a", StatusDefeat, "", 1},
		{0, "", "b", StatusSuccess, "b", 2},
		{0, "/heartbeat", "a", StatusDefeat, "b", 2},
	}
	for _, test := range tests {
		clock.Step(test.step)
		got := requestElection(t, router, "/v1/dk-election"+test.endpoint+"?token=secret&namespace=prod&id="+test.id, 200)
		assert.Equal(t, Result{
			Status: test.status, Namespace: "prod", ID: test.id, IncumbencyID: test.holder,
			Interval: 3, LeaseDuration: 30, Epoch: test.epoch,
		}, got)
	}
	for _, query := range []string{"token=other&namespace=prod", "token=secret&namespace=other"} {
		assert.Equal(t, StatusSuccess, requestElection(t, router, "/v1/dk-election?"+query+"&id=c", 200).Status)
	}
}

func TestElectionFailuresAreNotDefeat(t *testing.T) {
	conflict := apierrors.NewConflict(schema.GroupResource{Resource: "leases"}, "test", errors.New("conflict"))
	for _, failure := range []string{"cache", "get", "create", "update", "conflict"} {
		t.Run(failure, func(t *testing.T) {
			c, store, _, router := newElectionTest(t)
			path := "/v1/dk-election?token=secret&id=a"
			if failure == "update" || failure == "conflict" {
				requestElection(t, router, path, 200)
				path = "/v1/dk-election/heartbeat?token=secret&id=a"
			}
			switch failure {
			case "cache":
				c.cache.(*fakeLeaseCache).ready = false
			case "get":
				store.getErr = context.DeadlineExceeded
			case "create":
				store.createErr = context.DeadlineExceeded
			case "update":
				store.updateErr = context.DeadlineExceeded
			case "conflict":
				store.updateErr = conflict
			}
			got := requestElection(t, router, path, 503)
			assert.Equal(t, StatusError, got.Status)
			assert.NotEmpty(t, got.ErrorMsg)
		})
	}
	_, _, _, router := newElectionTest(t)
	for _, query := range []string{"id=a", "token=secret"} {
		assert.Equal(t, StatusError, requestElection(t, router, "/v1/dk-election?"+query, 400).Status)
	}
}

func TestConcurrentCampaignsHaveOneWinner(t *testing.T) {
	c, store, clock, first := newElectionTest(t)
	second := newContractRouter(t, newTestCoordinator(store, c.cache, clock))
	routers := []*gin.Engine{first, second}
	var wg sync.WaitGroup
	var winners atomic.Int64
	for i := range 64 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := requestElection(t, routers[i%2], "/v1/dk-election?token=secret&id="+strconv.Itoa(i), 200)
			if result.Status == StatusSuccess {
				winners.Add(1)
			} else {
				assert.Equal(t, StatusDefeat, result.Status)
			}
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, winners.Load())
}

func TestLateHeartbeatCannotOverwriteNewHolder(t *testing.T) {
	_, store, clock, router := newElectionTest(t)
	requestElection(t, router, "/v1/dk-election?token=secret&id=a", 200)
	atUpdate, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	store.beforeUpdate = func(lease *coordinationv1.Lease) {
		if holder(lease) == "a" {
			once.Do(func() { close(atUpdate); <-release })
		}
	}
	clock.Step(29 * time.Second)
	done := make(chan Result, 1)
	go func() {
		done <- requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=a", 200)
	}()
	<-atUpdate
	clock.Step(2 * time.Second)
	winner := requestElection(t, router, "/v1/dk-election?token=secret&id=b", 200)
	close(release)
	assert.Equal(t, StatusSuccess, winner.Status)
	late := <-done
	assert.Equal(t, StatusDefeat, late.Status)
	assert.Equal(t, "b", late.IncumbencyID)
	assert.EqualValues(t, 2, late.Epoch)
	assert.Equal(t, StatusSuccess, requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&id=b", 200).Status)
}

func TestFollowerTrafficAndMetricsUseCache(t *testing.T) {
	_, store, clock, router := newElectionTest(t)
	requestElection(t, router, "/v1/dk-election?token=secret&namespace=prod&id=leader", 200)
	clock.Step(3 * time.Second)
	requestElection(t, router, "/v1/dk-election/heartbeat?token=secret&namespace=prod&id=leader", 200)
	reads := store.getCalls.Load()
	for i := range 3 {
		got := requestElection(t, router, "/v1/dk-election?token=secret&namespace=prod&id=follower-"+strconv.Itoa(i), 200)
		assert.Equal(t, StatusDefeat, got.Status)
	}
	assert.EqualValues(t, 1, store.createCalls.Load())
	assert.EqualValues(t, 1, store.updateCalls.Load())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, 200, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "datakit_operator_election_active_scopes 1")
	for _, sensitive := range []string{"secret", "prod", "leader", "follower"} {
		assert.NotContains(t, recorder.Body.String(), sensitive)
	}
	assert.Equal(t, reads, store.getCalls.Load(), "followers and metrics must use the cache")
}
