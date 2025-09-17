# Developing an operator with operator-sdk

This example operator (`DummyOperator`) allows creating CR of type `Dummy`. The controller of the operator creates a `deployment` and a `service` based on the `Dummy` object.

## Useful resources

https://anupamgogoi.medium.com/writing-a-kubernetes-operator-from-zero-to-hero-8ca5dc2462b7
https://github.com/anupamgogoi-wso2/go-apps/tree/master/demo-operator

https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/
https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm

https://www.youtube.com/watch?v=LLVoyXjYlYM&t=1321s
https://github.com/civo/operator-demo

## Prerequisites

- git
- go version 1.18
- Ensure that your GOPROXY is set to "https://proxy.golang.org|direct"

## Install operator-sdk

Installing through git:
```bash
git clone https://github.com/operator-framework/operator-sdk
cd operator-sdk
git checkout master
make install
```

You must have a k8s cluster available with kubeconfig configured.
Check for OLM (Operator Lifecycle management) (https://sdk.operatorframework.io/docs/olm-integration/tutorial-bundle/#enabling-olm):
```bash
operator-sdk olm status
operator-sdk olm install
```
Initialize the empty github repository for the operator and copy its URL.
Create the operator folder and initialize the operator (the `--plugins=go/v4-alpha` is to develop with Mac Silicon only):
```bash
mkdir dummy-operator
cd dummy-operator
operator-sdk init --plugins=go/v4 --domain alessandroargentieri.com --repo github.com/alessandroargentieri/dummy-operator
```

Initialize the API:
```bash
operator-sdk create api --group apps --version v1 --kind Dummy --resource --controller
make manifests
```

Shape the Custom Resource with all the necessary fields by modifying the `api/v1/dummy_types.go` with `spec` and `status`
Update the CR definition with:
```bash
make manifest
```
Shape the controller logic (`controllers/dummy_controller.go`)
Modify the YAML given as sample (`config/samples/apps_v1_dummy.yaml`)

## Running the controller outside the cluster

You can debug and run the controller outside the cluster thanks to the configured kubeconfig you have in your terminal:
```bash
# apply the CR definition to the cluster
$ kubectl apply -f config/crd/bases/
customresourcedefinition.apiextensions.k8s.io/dummies.apps.alessandroargentieri.com created

# run the controller outside the cluster
$ go run main.go
```

Deploy an example of CR:
```bash
kubectl apply -f config/samples/apps_v1_dummy.yaml
```

## Run the operator inside the cluster

To run the operator inside the cluste you need to adjust the permission on top of the reconcile method of the controller (`controllers/dummy_controller.go`)

```golang
// +kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {...}
```

Update the manifests:
```bash
make manifests
```

Login your terminal to the Docker repository account and build and publish the docker image:
N.B.:
To let it work for k3s amd processors from a Mac Silicon I needed to update the Dockerfile in this part:
```Dockerfile
# RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go
```
Run the build and push:
```bash
make docker-build IMG=alessandroargentieri/dummy-operator:v0.0.1
make docker-push IMG=alessandroargentieri/dummy-operator:v0.0.1
```

I've created a `dist` folder copying/pasting the files from `config/rbac` and `config/crd/bases/apps.alessandroargentieri.com_dummies.yaml`.

I've changed some portions of the YAMLs and added a `controller.yaml` to allow deploying the controller inside the cluster through its docker image and a k8s deployment object.
I've preponed the number from 1 to 6 to the yaml file names to let them be applied in order.

Then, simply apply:
```bash
$ kubectl apply -f dist
namespace/operators unchanged
serviceaccount/dummy-operator-svc unchanged
clusterrole.rbac.authorization.k8s.io/manager-role configured
clusterrolebinding.rbac.authorization.k8s.io/manager-rolebinding unchanged
customresourcedefinition.apiextensions.k8s.io/dummies.apps.alessandroargentieri.com unchanged
deployment.apps/dummy-operator-controller-deployment created
```
Deploy an example of CR:
```bash
$ kubectl apply -f - << EOF
apiVersion: apps.alessandroargentieri.com/v1
kind: Dummy
metadata:
  labels:
    app.kubernetes.io/name: dummy-sample
  name: dummy-sample
spec:
  dummyDeployment:
    image: nginx
    replicas: 3
  dummyService:
    type: NodePort
    port: 80
    targetPort: 80
    nodePort: 31500
EOF
dummy.apps.alessandroargentieri.com/dummy-sample created
```
Check for the operator logs:
```bash
$ kubectl get pods -n operators
NAME                                                    READY   STATUS    RESTARTS   AGE
dummy-operator-controller-deployment-6cd77658b6-rgbbx   1/1     Running   0          39h

$ kubectl logs dummy-operator-controller-deployment-6cd77658b6-rgbbx -n operators
```

Port forward to nginx service to verify that service and deployment have been correctly created by the operator:

```bash
$ kubectl port-forward services/dummy-sample-3-service 8085:80
Forwarding from 127.0.0.1:8085 -> 80
Forwarding from [::1]:8085 -> 80
Handling connection for 8085
```

From another terminal perform the call to the service and check if the ngix deployment responds:

```bash
curl http://localhost:8085
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
html { color-scheme: light dark; }
body { width: 35em; margin: 0 auto;
font-family: Tahoma, Verdana, Arial, sans-serif; }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
```

# Operator-sdk autogenerated readme portion

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/dummy-operator:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/dummy-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```
### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

