#!/bin/bash

echo "====Create Self-Signed Certificates using OpenSSL===="

openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -out tls.csr -subj "/CN=datakit-operator.datakit.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:datakit-operator.datakit.svc") -in tls.csr -signkey tls.key -out tls.crt -days 3650

## Show the expiration time of the generated 
# openssl x509 -enddate -noout -in tls.crt
