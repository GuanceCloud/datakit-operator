// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package election

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegisterRoutes exposes the DataKit election protocol and coordinator health.
func (c *Coordinator) RegisterRoutes(router *gin.Engine) {
	router.POST("/v1/dk-election", func(ctx *gin.Context) { c.handle(ctx, operationCampaign) })
	router.POST("/v1/dk-election/heartbeat", func(ctx *gin.Context) { c.handle(ctx, operationHeartbeat) })
	router.GET("/v1/dk-election/status", c.handleStatus)
	router.GET("/metrics", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(c.renderMetrics()))
	})
}

// The startup probe checks authorization without creating or modifying a Lease.
func (c *Coordinator) handleStatus(ctx *gin.Context) {
	probeCtx, cancel := context.WithTimeout(ctx.Request.Context(), time.Second)
	defer cancel()
	if err := c.checkReady(probeCtx); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{"content": gin.H{"status": StatusError, "error_code": errorCode(err)}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"content": gin.H{"status": "ready"}})
}

func (c *Coordinator) checkReady(ctx context.Context) error {
	if c.namespace == "" {
		return errCacheNotReady
	}
	for _, verb := range []string{"get", "list", "watch", "create", "update"} {
		review, err := c.accessReviews.Create(ctx, &authorizationv1.SelfSubjectAccessReview{
			Spec: authorizationv1.SelfSubjectAccessReviewSpec{ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: c.namespace, Group: "coordination.k8s.io", Resource: "leases", Verb: verb,
			}},
		}, metav1.CreateOptions{})
		if err != nil {
			return c.storageError(operationInformer, "check_rbac", c.namespace, err)
		}
		if !review.Status.Allowed {
			log.Warnf("DataKit election RBAC missing: namespace=%s verb=%s", c.namespace, verb)
			return errForbidden
		}
	}
	if !c.cache.Ready() {
		return errCacheNotReady
	}
	return nil
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, errForbidden):
		return "rbac_forbidden"
	case errors.Is(err, errCacheNotReady):
		return "cache_not_ready"
	default:
		return "storage_unavailable"
	}
}

func (c *Coordinator) handle(ctx *gin.Context, operation electionOperation) {
	token, namespace, id := ctx.Query("token"), ctx.Query("namespace"), ctx.Query("id")
	result := c.result(StatusError, namespace, id, nil, false)
	status := http.StatusOK
	switch {
	case token == "" || id == "":
		status = http.StatusBadRequest
		result.ErrorMsg = "token and id cannot be empty"
	default:
		elected, err := c.elect(ctx.Request.Context(), operation, token, namespace, id)
		if err != nil {
			status = http.StatusServiceUnavailable
			result.ErrorMsg = "DataKit election coordinator unavailable"
			result.ErrorCode = errorCode(err)
		} else {
			result = elected
		}
	}
	c.metrics.recordRequest(operation, result.Status)
	ctx.JSON(status, ResponseEnvelope{Content: result})
}
