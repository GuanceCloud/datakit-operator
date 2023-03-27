## Using Operator to Inject Lib Files and Related Environment Variables into Pods

### Use Case

Users have deployed many Pods (Deployment/DaemonSet), and want to enable trace collection and send it to Datakit but the Pods lack the required dd-lib files and environment variables.

Is there a simple way to add dd-lib files in bulk to Pods?

### Usage

Firstly, it is feasible according to the mechanism of Kubenertes Admission Controller.

**Note that this is an intrusive behavior that modifies the user's original yaml file and injects the required data into it. Not everyone is willing to have their yaml modified.**

The specific method is as follows:

1. Users download and install datakit-operator.yaml in their own k8s cluster.
2. Add the specified Annotation to all Pods that need to add dd-lib files.
3. At the same time, add the environment variables `DD_AGENT_HOST` and `DD_TRACE_AGENT_PORT` to the Pod to specify the receiving address. Taking JAVA as an example, see [dd-trace JAVA startup documentation](https://docs.guance.com/datakit/ddtrace-java/#start-options) for details.

After datakit-operator runs, it will decide whether to add dd-lib files and environment variables based on the Annotation.

> The specified Annotation key is `admission.datakit/%s-lib.version`, where %s needs to be replaced with the specified language. Currently, `java`, `python`, and `js` are supported.

> The value is the specified version number. If null, the default version will be used: `java: v1.8.4-guance, python: v1.6.2, js: v3.9.2`.

1. [Install Datkait-Operator](#datakit-operator-install)
2. Modify the existing application yaml. Take nginx deployment as an example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
      annotations:
        admission.datakit/js-lib.version: ""
    spec:
      containers:
      - name: nginx
        image: nginx:1.22
        ports:
        - containerPort: 80
        env:
        - name: DD_AGENT_HOST
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: status.hostIP
        - name: DD_TRACE_AGENT_PORT
          value: 9529
```

There are three configurations that need to be manually added:

- Add Annotation `admission.datakit/js-lib.version: ""`.
- Add the environment variable `DD_AGENT_HOST`, and use hostIP for value.
- Add the environment variable `DD_TRACE_AGENT_PORT`, and set the value to Datakit's port 9529.

Create resources using the yaml file:

```shell
$ kubectl apply -f nginx.yaml
```

At this point, datakit-operator has added java-lib files to all Pods in nginx deployment. Verify as follows:

```shell
$ kubectl get pod
$ NAME                                   READY   STATUS    RESTARTS      AGE
nginx-deployment-7bd8dd85f-fzmt2       1/1     Running   0             4s
$ kubectl get pod nginx-deployment-7bd8dd85f-fzmt2 -o=jsonpath={.spec.initContainers\[0\].name}
$ datakit-lib-init
```

The initContainers named `datakit-lib-init` is added by Datakit-Operator, which contains the java-lib file and shares a volume with the main application container, allowing the main container to access this file.

### Configuring dd-lib Images by Using Environment Variables

The dd-lib images used by Datakit-Operator are stored in `pubrepo.jiagouyun.com/datakit-operator`. For some special environments, it may not be convenient to access this image repository. It is possible to configure the image path using environment variables.

The default environment variables are as follows:

- ENV_DD_JAVA_AGENT_IMAGE: `pubrepo.jiagouyun.com/datakit-operator/dd-lib-java-init:v1.8.4-guance`
- ENV_DD_PYTHON_AGENT_IMAGE: `pubrepo.jiagouyun.com/datakit-operator/dd-lib-python-init:v1.6.2`
- ENV_DD_JS_AGENT_IMAGE: `pubrepo.jiagouyun.com/datakit-operator/dd-lib-js-init:v3.9.2`

**Datakit-Operator does not check the image, if the image path is incorrect, Kubernetes will report an error when creating the Pod.**

To customize the image repository, let's continue with the example of injecting Java dd-libs into Pods:

1. In an environment that can access `pubrepo.jiagouyun.com`, pull the image `pubrepo.jiagouyun.com/datakit-operator/dd-lib-java-init:v1.8.4-guance` and save it to your own image repository, such as `inside.image.hub/datakit-operator/dd-lib-java-init:v1.8.4-guance`.
2. Modify datakit-operator.yaml and change the environment variable `ENV_DD_JAVA_AGENT_IMAGE` to `inside.image.hub/datakit-operator/dd-lib-java-init:v1.8.4-guance`, then apply this yaml.
3. After this, Datakit-Operator will use the new image path.


> If a version has already been specified in the Pod Annotation `admission.datakit/java-lib.version`, such as `admission.datakit/java-lib.version:v2.0.1-guance` or `admission.datakit/java-lib.version:latest`, that version will be used.

### Overview of Relevant Principles

Injecting dd-libs into Pods is a risky behavior because it involves intrusive modification of the original yaml. Here are the details of the implementation and execution process, but you can refer to the source code for more information.

[Admission Controller](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/admission-controllers/) is a feature of Kubernetes that intercepts requests that arrive at the API server after authentication, authorization, and before persistence, until the objects have been persisted.

1. Datakit-Operator takes advantage of this feature by registering its admission mutate with k8s, allowing itself to access and modify all Pods (at creation time).

2. When a new Pod is created (CREATE), the k8s apiserver sends the Pod configuration to Datakit-Operator, which scans the Annotations and finds `admission.datakit/js-lib-verion`. If not found, the configuration remains unchanged.

3. For Pods that meet the criteria, Datakit-Operator adds an `initContainers` to the configuration, whose image is obtained by converting `admission.datakit/js-lib-verion`, and which contains a lib file under the `/datadog-lib` path.

4. Datakit-Operator connects the `/datadog-lib` path of the `initContainers` and other containers, allowing other containers to access this path's files.

5. Datakit-Operator adds a special environment variable to all containers, such as `NODE_OPTIONS`=`--require=/datadog-lib/node_modules/dd-trace/init`.

6. Finally, Datakit-Operator sends the modified Pod configuration back to the k8s apiserver, which creates the Pod. At this point, the running Pod has been injected with the lib file.

In addition, this also works for Deployment and DaemonSet, as they are higher-level orchestration for Pods and ultimately create Pods.
