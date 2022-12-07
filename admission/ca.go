package admission

import (
	"io/ioutil"
)

const (
	tlsKeyFile  = "/usr/local/datakit-operator/tls.key"
	tlsCertFile = "/usr/local/datakit-operator/tls.cert"
)

func writeTLSFile() (err error) {
	keyData, err := ioutil.ReadFile("certs/tls.key")
	if err != nil {
		return err
	}

	certData, err := ioutil.ReadFile("certs/tls.cert")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(tlsKeyFile, keyData, 0666)
	err = ioutil.WriteFile(tlsCertFile, certData, 0666)
	return
}
