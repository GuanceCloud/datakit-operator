#!/bin/bash

echo "====Create Self-Signed Certificates using OpenSSL===="

certs_dir=build/certs
mkdir -p $certs_dir

openssl genrsa -out $certs_dir/tls.key 2048
openssl req -new -key $certs_dir/tls.key -out $certs_dir/tls.csr -subj "/CN=datakit-operator.datakit.svc"
openssl x509 -req -extfile <(printf "subjectAltName=DNS:datakit-operator.datakit.svc") -in $certs_dir/tls.csr -signkey $certs_dir/tls.key -out $certs_dir/tls.crt
