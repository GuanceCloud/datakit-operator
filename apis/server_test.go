// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package apis

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinRecoveryDoesNotLogQueryCredentials(t *testing.T) {
	previousMode := gin.Mode()
	gin.SetMode(gin.DebugMode)
	defer gin.SetMode(previousMode)
	for _, query := range []string{
		"token=tkn-top-secret&namespace=production",
		"token=tkn-top-secret;ignored=true",
		"token=tkn-top-secret%zz",
	} {
		t.Run(query, func(t *testing.T) {
			var recoveryLog bytes.Buffer
			router := gin.New()
			router.Use(gin.RecoveryWithWriter(&recoveryLog), redactTokenOnPanic())
			router.POST("/v1/dk-election", func(*gin.Context) { panic("forced panic") })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/dk-election?"+query, nil))
			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("panic status = %d, want 500", recorder.Code)
			}
			if strings.Contains(recoveryLog.String(), "tkn-top-secret") ||
				!strings.Contains(recoveryLog.String(), "POST /v1/dk-election HTTP/1.1") {
				t.Fatalf("recovery did not strip query credentials: %s", recoveryLog.String())
			}
		})
	}
}
