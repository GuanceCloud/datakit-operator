package apis

import (
	"net/http"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
	"gitlab.jiagouyun.com/cloudcare-tools/datakit-operator/apis/admission"
)

var l = logger.DefaultSLogger("apis")

const (
	tlsKeyFile  = "/usr/local/datakit-operator/certs/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/certs/tls.crt"
)

func Run(addr string) {
	l = logger.SLogger("apis")
	admission.Steup()

	http.HandleFunc("/v1/webhooks/inject", admission.HandleInject)
	server := &http.Server{
		Addr: addr,
	}

	l.Debugf("server running...")
	err := server.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	if err != nil {
		l.Error(err)
	}
}
