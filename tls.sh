#!/bin/bash

echo "====Create Self-Signed Certificates using OpenSSL===="
mkdir -p certs
openssl genrsa -out certs/tls.key 2048
openssl req -new -key certs/tls.key -out certs/tls.csr -subj "/CN=datakit-operator.datakit.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:datakit-operator.datakit.svc") -in certs/tls.csr -signkey certs/tls.key -out certs/tls.crt
