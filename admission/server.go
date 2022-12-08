package admission

import (
	"net/http"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
)

var l = logger.DefaultSLogger("admission")

const (
	tlsKeyFile  = "/usr/local/datakit-operator/certs/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/certs/tls.crt"
)

func Run(addr string) {
	l = logger.SLogger("admission")
	l.Debugf("server running...")

	http.HandleFunc("/v1/webhooks/injectlib", handleInjectLib)
	server := &http.Server{
		Addr: addr,
	}

	err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	if err != nil {
		l.Error(err)
	}
}
