// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	typedauthorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
)

const (
	StatusSuccess = "success"
	StatusDefeat  = "defeat"
	StatusError   = "error"

	leaseDuration     = 30 * time.Second
	requestInterval   = 3 * time.Second
	managedLeaseLabel = "election.datakits.io/managed"
	maxCASAttempts    = 5
)

var (
	errCacheNotReady = errors.New("DataKit election Lease cache is not ready")
	errStorage       = errors.New("DataKit election Lease storage is unavailable")
	errForbidden     = errors.New("DataKit election Lease access is forbidden")
)

type Result struct {
	Status        string `json:"status"`
	Namespace     string `json:"namespace"`
	ID            string `json:"id"`
	IncumbencyID  string `json:"incumbency_id"`
	ErrorMsg      string `json:"error_msg"`
	ErrorCode     string `json:"error_code,omitempty"`
	Interval      int    `json:"interval"`
	LeaseDuration int    `json:"lease_duration"`
	Epoch         int64  `json:"epoch"`
}

type ResponseEnvelope struct {
	Content Result `json:"content"`
}

type leaseStore interface {
	Get(context.Context, string, metav1.GetOptions) (*coordinationv1.Lease, error)
	Create(context.Context, *coordinationv1.Lease, metav1.CreateOptions) (*coordinationv1.Lease, error)
	Update(context.Context, *coordinationv1.Lease, metav1.UpdateOptions) (*coordinationv1.Lease, error)
}

type leaseCache interface {
	Start(context.Context)
	Ready() bool
	Get(string) (*coordinationv1.Lease, error)
	List(labels.Selector) ([]*coordinationv1.Lease, error)
}

type Coordinator struct {
	accessReviews typedauthorizationv1.SelfSubjectAccessReviewInterface
	store         leaseStore
	cache         leaseCache
	now           func() time.Time     // Wall time is only used for persisted timestamps and lease age.
	elapsed       func() time.Duration // Monotonic time determines whether a Lease can be taken over.
	namespace     string
	metrics       electionMetrics
	observations  leaseObservations
}

func (c *Coordinator) elect(ctx context.Context, operation electionOperation, token, namespace, id string) (Result, error) {
	if !c.cache.Ready() {
		return Result{}, errCacheNotReady
	}
	name := leaseName(token, namespace)
	lease, err := c.cache.Get(name)
	if err != nil && !apierrors.IsNotFound(err) {
		return Result{}, c.storageError(operation, "read_cache", name, err)
	}
	// Only negative replies use cached state. A success always requires a CAS write.
	if c.remaining(lease, fromCache) > 0 && (operation == operationCampaign || holder(lease) != id) {
		return c.result(StatusDefeat, namespace, id, lease, true), nil
	}

	for attempt := 0; ; attempt++ {
		lease, err = c.store.Get(ctx, name, metav1.GetOptions{})
		missing := apierrors.IsNotFound(err)
		if err != nil && !missing {
			return Result{}, c.storageError(operation, "get", name, err)
		}
		active := c.remaining(lease, fromStore) > 0
		if operation == operationCampaign && active ||
			operation == operationHeartbeat && (!active || holder(lease) != id) {
			return c.result(StatusDefeat, namespace, id, lease, active), nil
		}
		// Reread after the final conflict, but do not infer that a failed write succeeded.
		if attempt == maxCASAttempts {
			return Result{}, c.storageError(operation, "conflict", name, errors.New("lease update still conflicts"))
		}

		var updated *coordinationv1.Lease
		switch {
		case missing:
			updated, err = c.store.Create(ctx, newLease(c.namespace, name, id, c.now()), metav1.CreateOptions{})
		case operation == operationCampaign:
			candidate := lease.DeepCopy()
			acquireLease(candidate, id, c.now())
			updated, err = c.store.Update(ctx, candidate, metav1.UpdateOptions{})
		default:
			renewed := lease.DeepCopy()
			renewTime := metav1.NewMicroTime(c.now().UTC())
			renewed.Spec.RenewTime = &renewTime
			updated, err = c.store.Update(ctx, renewed, metav1.UpdateOptions{})
		}
		if apierrors.IsConflict(err) || apierrors.IsAlreadyExists(err) || apierrors.IsNotFound(err) {
			c.metrics.recordCASConflict(operation)
			continue
		}
		if err != nil {
			return Result{}, c.storageError(operation, "write", name, err)
		}
		_ = c.remaining(updated, fromStore)
		if operation == operationCampaign {
			reason := transitionLeaseExpired
			if missing {
				reason = transitionFirstAcquire
			}
			c.metrics.recordTransition(reason)
			log.Infof("DataKit leader transition: scope=%s holder=%q epoch=%d reason=%s",
				name, holder(updated), epoch(updated), transitionNames[reason])
		}
		return c.result(StatusSuccess, namespace, id, updated, true), nil
	}
}

func (c *Coordinator) result(status, namespace, id string, lease *coordinationv1.Lease, active bool) Result {
	result := Result{
		Status: status, Namespace: namespace, ID: id,
		Interval: int(requestInterval / time.Second), LeaseDuration: int(leaseDuration / time.Second),
		Epoch: epoch(lease),
	}
	if active {
		result.IncumbencyID = holder(lease)
	}
	if lease != nil && lease.Spec.LeaseDurationSeconds != nil {
		result.LeaseDuration = int(*lease.Spec.LeaseDurationSeconds)
	}
	return result
}

func (c *Coordinator) storageError(operation electionOperation, action, scope string, cause error) error {
	c.metrics.recordStorageError(operation)
	log.Errorf("DataKit election storage failure: operation=%s action=%s scope=%s error=%v",
		operationNames[operation], action, scope, cause)
	if apierrors.IsForbidden(cause) {
		return errForbidden
	}
	return errStorage
}

func leaseName(token, namespace string) string {
	payload := make([]byte, 8+len(token)+len(namespace))
	binary.BigEndian.PutUint32(payload[:4], uint32(len(token)))
	copy(payload[4:], token)
	offset := 4 + len(token)
	binary.BigEndian.PutUint32(payload[offset:offset+4], uint32(len(namespace)))
	copy(payload[offset+4:], namespace)
	sum := sha256.Sum256(payload)
	return "datakit-election-" + hex.EncodeToString(sum[:])
}

func newLease(namespace, name, id string, now time.Time) *coordinationv1.Lease {
	lease := &coordinationv1.Lease{ObjectMeta: metav1.ObjectMeta{
		Name: name, Namespace: namespace, Labels: map[string]string{managedLeaseLabel: "true"},
	}}
	acquireLease(lease, id, now)
	return lease
}

func acquireLease(lease *coordinationv1.Lease, id string, now time.Time) {
	timestamp := metav1.NewMicroTime(now.UTC())
	durationSeconds := int32(leaseDuration / time.Second)
	nextEpoch := int32(epoch(lease) + 1)
	lease.Spec.HolderIdentity = &id
	lease.Spec.LeaseDurationSeconds = &durationSeconds
	lease.Spec.AcquireTime = &timestamp
	lease.Spec.RenewTime = &timestamp
	lease.Spec.LeaseTransitions = &nextEpoch
}

func holder(lease *coordinationv1.Lease) string {
	if lease == nil || lease.Spec.HolderIdentity == nil {
		return ""
	}
	return *lease.Spec.HolderIdentity
}

func epoch(lease *coordinationv1.Lease) int64 {
	if lease == nil || lease.Spec.LeaseTransitions == nil {
		return 0
	}
	return int64(*lease.Spec.LeaseTransitions)
}
