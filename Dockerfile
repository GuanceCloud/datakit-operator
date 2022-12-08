FROM ubuntu:18.04 AS base
ARG TARGETARCH

RUN mkdir -p /usr/local/datakit-operator

COPY build/${TARGETARCH}/ /usr/local/datakit-operator
COPY build/certs/ /usr/local/datakit-operator/certs
WORKDIR /usr/local/datakit-operator

CMD ["/usr/local/datakit-operator/datakit-operator"]
