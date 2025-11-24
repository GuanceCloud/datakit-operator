// Unless explicitly stated otherwise all files in this repository are licensed
// under the MIT License.
// This product includes software developed at Guance Cloud (https://www.guance.com/).
// Copyright 2021-present Guance, Inc.

package client

import (
	"fmt"
	"net"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	loggingclientv1 "gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/pkg/kubernetes/pkg/client/clientset/versioned"
)

const (
	LimitQPS   = float32(1000)
	LimitBurst = 1000

	// nolint:gosec
	TokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

func DefaultConfigInCluster() (*rest.Config, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if len(host) == 0 || len(port) == 0 {
		return nil, fmt.Errorf("unable to load in-cluster configuration")
	}

	token, err := os.ReadFile(TokenFile)
	if err != nil {
		return nil, err
	}

	cfg := rest.Config{
		Host:            "https://" + net.JoinHostPort(host, port),
		TLSClientConfig: rest.TLSClientConfig{Insecure: true},
		BearerToken:     string(token),
		BearerTokenFile: TokenFile,
		RateLimiter:     flowcontrol.NewTokenBucketRateLimiter(LimitQPS, LimitBurst),
	}
	return &cfg, nil
}

type Client interface {
	Kubernetes() *kubernetes.Clientset
	Logging() *loggingclientv1.Clientset
}

type client struct {
	Clientset     *kubernetes.Clientset
	LoggingClient *loggingclientv1.Clientset
}

func (c *client) Kubernetes() *kubernetes.Clientset   { return c.Clientset }
func (c *client) Logging() *loggingclientv1.Clientset { return c.LoggingClient }

func NewClient(restConfig *rest.Config) (Client, error) {
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	loggingClient, err := loggingclientv1.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &client{
		Clientset:     clientset,
		LoggingClient: loggingClient,
	}, nil
}

func NewClientInCluster() (Client, error) {
	config, err := DefaultConfigInCluster()
	if err != nil {
		return nil, err
	}
	return NewClient(config)
}
