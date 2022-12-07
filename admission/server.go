package admission

import (
	"fmt"
	"net/http"

	"gitlab.jiagouyun.com/cloudcare-tools/cliutils/logger"
)

var l = logger.DefaultSLogger("admission")

func Run(port int) {
	l = logger.SLogger("admission")

	if err := writeTLSFile(); err != nil {
		l.Error(err)
		return
	}

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
