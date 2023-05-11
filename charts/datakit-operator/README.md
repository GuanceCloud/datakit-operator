# Datakit Operator Helm Chart

This Helm chart installs [Datakit Operator](https://github.com/GuanceCloud/datakit-operator) with configurable TLS, RBAC and much more configurations. This chart caters a number of different use cases and setups.

- [Requirements](#requirements)
- [Installing](#installing)
- [Uninstalling](#uninstalling)
- [Configuration](#configuration)

## Requirements

- Kubernetes 1.14+

- Helm 3.0+

  

## Installing

 ```shell
$ helm install datakit-operator datakit-operator --repo https://pubrepo.guance.com/chartrepo/datakit-operator -n datakit --create-namespace 
 ```


## Uninstalling

```shell
$ helm uninstall datakit-operator -n datakit
```

## Configuration

| Parameter                         | Description                                                                                                                                                                        | Default                                                                 | Required             |
| ------------------------          | ------------------------------------------------------------                                                                                                                       | ------------------------------------------------------------            | --------             |
| `image.repository`                | The DataKit Docker image                                                                                                                                                           | `pubrepo.guance.com/chartrepo/datakit`                                  | `true`               |
| `image.pullPolicy`                | The Kubernetes [imagePullPolicy][] value                                                                                                                                           | `IfNotPresent`                                                          |                      |
| `image.tag`                       | The DataKit Docker image tag                                                                                                                                                       | `""`                                                                    |                      |
| `env`                             | env Add env for customization,[more](https://docs.guance.com/datakit/datakit-operator/#datakit-operator-inject-logfwd-configurations)                                                                        | `[]`                                                                    |                      |
| `nameOverride`                    | Overrides the `clusterName` when used in the naming of resources                                                                                                                   | ""                                                                      |                      |
| `fullnameOverride`                | Overrides the `clusterName` and `nodeGroup` when used in the naming of resources. This should only be used when using a single `nodeGroup`, otherwise you will have name conflicts | ""                                                                      |                      |
| `podAnnotations`                  | Configurable [annotations][] applied to all OpenSearch pods                                                                                                                        |                                                     |  |  |
| `tolerations`                     | Configurable [tolerations][]                                                                                                                                                       | `- operator: Exists`                                                    |                      |
| `service.type`                    | DataKit [Service Types][]                                                                                                                                                          | `ClusterIP`                                                             |                      |
| `service.port`                    | DataKit service port                                                                                                                                                               | `443`                                                                  |                      |
| `service.targetPort`              | DataKit service targetPort                                                                                                                                                         | `9543`                                                                  |                      |



[environment from variables]: https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/#configure-all-key-value-pairs-in-a-configmap-as-container-environment-variables

[hostAliases]: https://kubernetes.io/docs/concepts/services-networking/add-entries-to-pod-etc-hosts-with-host-aliases/

[image.pullPolicy]: https://kubernetes.io/docs/concepts/containers/images/#updating-images

[annotations]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/

[tolerations]: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/

[service types]: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
