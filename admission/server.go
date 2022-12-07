package admission

import (
	"fmt"
	"net/http"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
)

var l = logger.DefaultSLogger("admission")

const (
	tlsKeyFile  = "/usr/local/datakit-operator/certs/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/certs/tls.crt"
)

func Run(port int) {
	l = logger.SLogger("admission")
	l.Debugf("server running...")

	http.HandleFunc("/v1/webhooks/injectlib", handleInjectLib)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	if err != nil {
		l.Error(err)
	}
}
