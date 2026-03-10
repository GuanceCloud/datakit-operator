// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package apis

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/cluster"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/logging"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/client"
)

var log = logger.DefaultSLogger("apis")

const (
	tlsKeyFile  = "/usr/local/datakit-operator/certs/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/certs/tls.crt"
)

func Run(ctx context.Context, addr string) error {
	log = logger.SLogger("apis")

	// 创建 k8s client，这是核心功能，失败则退出
	k8sClient, err := client.NewClientInCluster()
	if err != nil {
		log.Errorf("failed to create k8s client: %v", err)
		return err
	}

	webhook.Setup(ctx)
	logging.Setup(ctx)
	cluster.Setup(ctx)

	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(ginLogger())

	// 注册路由
	router.GET("/v1/ping", handlePing)
	log.Info("route registered /v1/ping")

	router.POST("/v1/webhooks/inject", webhook.HandleInject)
	log.Info("route registered /v1/webhook/inject")

	if err := logging.CheckClusterLoggingConfigRBAC(k8sClient); err != nil {
		log.Warnf("RBAC check failed for logging: %v, logging API will be disabled", err)
	} else {
		logging.StartLoggingConfigWatcher(ctx, k8sClient)
		router.GET("/v1/logging/configs", logging.HandleConfigs)
		log.Info("route registered /v1/logging/configs")
	}

	if err := cluster.CheckPodRBAC(k8sClient); err != nil {
		log.Warnf("RBAC check failed: %v, cluster API will be disabled", err)
	} else {
		clusterHandler := cluster.NewHandler(k8sClient)

		if err := clusterHandler.Start(ctx); err != nil {
			log.Warnf("failed to start cluster handler: %v, cluster API will be disabled", err)
		} else {
			coreV1Group := router.Group("/v1/cluster/api/v1")
			{
				coreV1Group.GET("/pods", clusterHandler.ListAllPods)
				coreV1Group.GET("/namespaces/:namespace/pods", clusterHandler.ListPods)
				coreV1Group.GET("/namespaces/:namespace/pods/:name", clusterHandler.GetPod)
			}
			log.Info("route registered /v1/clusters/:path")
		}
	}

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Infof("server listening on %s", addr)

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error(err)
			return err
		}
		log.Info("server shutdown complete")
		return nil

	case err := <-errCh:
		if err != nil {
			log.Error(err)
			return err
		}
		return nil
	}
}

// PingResponse ping 接口的响应结构
type PingResponse struct {
	Status string `json:"status"`
}

// handlePing 处理 ping 请求，返回 operator 运行状态
func handlePing(c *gin.Context) {
	c.JSON(http.StatusOK, PingResponse{
		Status: "ok",
	})
}

func ginLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Debugf("[%s] %s %s %d %v %s",
			clientIP,
			method,
			path,
			statusCode,
			latency,
			c.Request.UserAgent(),
		)
	}
}
