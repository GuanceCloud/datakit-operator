// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package apis

import (
	"context"
	"net/http"
	"time"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/logging"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/webhook"
)

var log = logger.DefaultSLogger("apis")

const (
	tlsKeyFile  = "/usr/local/datakit-operator/certs/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/certs/tls.crt"
)

func Run(ctx context.Context, addr string) error {
	log = logger.SLogger("apis")

	mux := http.NewServeMux()
	webhook.Setup(ctx)
	logging.Setup(ctx)

	// 路由由上层统一决定
	mux.HandleFunc("/v1/webhooks/inject", webhook.HandleInject)
	mux.HandleFunc("/v1/logging/configs", logging.HandleConfigs)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
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
