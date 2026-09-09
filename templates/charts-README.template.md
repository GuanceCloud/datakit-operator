# DataKit Operator Helm Chart

This Helm chart installs [DataKit Operator]() with configurable TLS, RBAC and much more configurations. This chart caters a number of different use cases and setups.

- [Requirements](#requirements)
- [Installing](#installing)
- [Uninstalling](#uninstalling)
- [Tracing injection](#tracing-injection)
- [Central election](#central-election)
- [Configuration](#configuration)

## Requirements

- Kubernetes 1.14+

- Helm 3.0+

  

## Installing

 ```shell
$ helm install datakit-operator datakit-operator --repo https://pubrepo.(@BRAND_DOMAIN)/chartrepo/datakit-operator -n datakit --create-namespace 
 ```


## Uninstalling

```shell
$ helm uninstall datakit-operator -n datakit
```

## Tracing injection

The shipped configuration enables one Java DDTrace rule for Pods in the `default` namespace. This preserves the existing default behavior; the Operator does not detect the application language automatically.

DDTrace also supports Python, PHP, and Node.js. OpenTelemetry supports Java, Python, and Node.js, but is disabled by default with `otels: []`. To enable another language or OpenTelemetry, add explicit rules with mutually exclusive language labels. See the [DDTrace injection documentation](https://docs.(@BRAND_DOMAIN)/datakit/operator-ddtrace/) and [OpenTelemetry injection documentation](https://docs.(@BRAND_DOMAIN)/datakit/operator-otel/) for complete configuration examples.

## Central election

DataKit Operator v1.9.1 or later can replace DataWay/Kodo for central election when used with DataKit 2.12.0 or later in the same Kubernetes cluster. The chart includes the required Kubernetes Lease permissions. Configure `ENV_ENABLE_ELECTION=true` and `ENV_ELECTION_OPERATOR_URL=https://datakit-operator.datakit.svc:443` on every DataKit in the election group. DataKit selects the election service at startup and does not switch at runtime. See the [DataKit Operator documentation](https://docs.(@BRAND_DOMAIN)/datakit/datakit-operator/#central-election) for setup and upgrade instructions.

## Configuration

| Parameter                | Description                                                                                                                                                                        | Default                                                      | Required |
| ------------------------ | ------------------------------------------------------------                                                                                                                       | ------------------------------------------------------------ | -------- |
| `image.repository`       | The DataKit Docker image                                                                                                                                                           | `pubrepo.(@BRAND_DOMAIN).com/chartrepo/datakit`              | `true`   |
| `image.pullPolicy`       | The Kubernetes [imagePullPolicy][] value                                                                                                                                           | `IfNotPresent`                                               |          |
| `image.tag`              | The DataKit Docker image tag                                                                                                                                                       | `""`                                                         |          |
| `env`                    | env Add env for customization,[more](https://docs.(@BRAND_DOMAIN).com/datakit/datakit-operator/#datakit-operator-inject-logfwd-configurations)                                     | `[]`                                                         |          |
| `nameOverride`           | Overrides the `clusterName` when used in the naming of resources                                                                                                                   | ""                                                           |          |
| `fullnameOverride`       | Overrides the `clusterName` and `nodeGroup` when used in the naming of resources. This should only be used when using a single `nodeGroup`, otherwise you will have name conflicts | ""                                                           |          |
| `podAnnotations`         | Configurable [annotations][] applied to all OpenSearch pods                                                                                                                        |                                                              |          |  |
| `tolerations`            | Configurable [tolerations][]                                                                                                                                                       | `- operator: Exists`                                         |          |
| `service.type`           | DataKit [Service Types][]                                                                                                                                                          | `ClusterIP`                                                  |          |
| `service.port`           | DataKit service port                                                                                                                                                               | `443`                                                        |          |
| `service.targetPort`     | DataKit service targetPort                                                                                                                                                         | `9543`                                                       |          |



[environment from variables]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables

[hostAliases]: https://kubernetes.io/docs/concepts/services-networking/add-entries-to-pod-etc-hosts-with-host-aliases/

[image.pullPolicy]: https://kubernetes.io/docs/concepts/containers/images/#updating-images

[annotations]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/

[tolerations]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/

[service types]: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
