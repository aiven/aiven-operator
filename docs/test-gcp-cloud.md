## Requirements

- Install the [Operator SDK CLI >=1.6](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/)
- Install GCP command line tool `gcloud`, instructions can be found [here](https://cloud.google.com/sdk/docs/quickstart)
- Install `kubectl`

## Usage

Then with the gcloud CLI tool or using the GCP web console create a new project and start a new Kubernetes cluster (can
be found in Kubernetes Engine section). After that, authenticate `gcloud` to your account and set a newly created
project as the default.

Go to your K8s Engine cluster and click on the "connect" tab, which will generate a command that will configure. It will
look something like that:

```shell
gcloud container clusters get-credentials cluster-1 --zone us-central1-c --project integral-magnet-12345
```

Please copy & Paste it to your host. It should connect your local `kubectl` to the Kubernetes cluster in the GCP cloud.
To validate this, run `kubectl cluster-info`, and the output should indicate that it is connected to the GCP Kubernetes
Engine cluster.

### OLM installation

Make sure you have [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager/)
installed on yor K8s cluster. To check this run `operator-sdk olm status` which should print something like that:

```shell
$ operator-sdk olm status

INFO[0004] Fetching CRDs for version "0.17.0"           
INFO[0004] Using locally stored resource manifests      
INFO[0029] Successfully got OLM status for version "0.17.0" 

NAME                                            NAMESPACE    KIND                        STATUS
operators.operators.coreos.com                               CustomResourceDefinition    Installed
operatorgroups.operators.coreos.com                          CustomResourceDefinition    Installed
installplans.operators.coreos.com                            CustomResourceDefinition    Installed
clusterserviceversions.operators.coreos.com                  CustomResourceDefinition    Installed
olm-operator                                    olm          Deployment                  Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
olm-operator-binding-olm                                     ClusterRoleBinding          Installed
operatorhubio-catalog                           olm          CatalogSource               Installed
olm-operators                                   olm          OperatorGroup               Installed
aggregate-olm-view                                           ClusterRole                 Installed
catalog-operator                                olm          Deployment                  Installed
aggregate-olm-edit                                           ClusterRole                 Installed
olm                                                          Namespace                   Installed
global-operators                                operators    OperatorGroup               Installed
operators                                                    Namespace                   Installed
packageserver                                   olm          ClusterServiceVersion       Installed
olm-operator-serviceaccount                     olm          ServiceAccount              Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
system:controller:operator-lifecycle-manager                 ClusterRole                 Installed
```

All resources listed should have status `Installed`.

Please use OLM v0.17.0 or later, If OLM is not already installed, go ahead and install the latest version:

```shell
$ operator-sdk olm install
```

**Note:** By default, `olm status` and `olm uninstall` auto-detect the OLM version installed in your cluster. This can
fail if the installation is broken in some way, so the version of OLM can be overridden using the `--version` flag
provided with these commands.

### Run bundle

The simples way to run the bundle is to use `operator-sdk`:

```shell
$ operator-sdk run bundle eu.gcr.io/<PROJECT>/aiven-operator-bundle:<VERSION>

INFO[0010] Successfully created registry pod: eu-gcr-io-integral-magnet-301414-aiven-operator-bundle-v0-0-70 
INFO[0010] Created CatalogSource: aiven-operator-catalog 
INFO[0011] OperatorGroup "operator-sdk-og" created      
INFO[0011] Created Subscription: aiven-operator-v0-0-1-sub 
INFO[0020] Approved InstallPlan install-xfdwd for the Subscription: aiven-operator-v0-0-1-sub 
INFO[0020] Waiting for ClusterServiceVersion "default/aiven-operator.v0.0.1" to reach 'Succeeded' phase 
INFO[0020]   Waiting for ClusterServiceVersion "default/aiven-operator.v0.0.1" to appear 
INFO[0025]   Found ClusterServiceVersion "default/aiven-operator.v0.0.1" phase: Pending 
INFO[0029]   Found ClusterServiceVersion "default/aiven-operator.v0.0.1" phase: InstallReady 
INFO[0030]   Found ClusterServiceVersion "default/aiven-operator.v0.0.1" phase: Installing 
INFO[0035]   Found ClusterServiceVersion "default/aiven-operator.v0.0.1" phase: Succeeded 
INFO[0035] OLM has successfully installed "aiven-operator.v0.0.1"
```

It should deploy Aiven Operator bundle to the `default` namespace, to validate this please run `kubectl get pods`, which
should print all run pods in the `default` namespace, look for something with the
name `aiven-k8s-operator-controller-manager`. To see what is happening inside read logs from `manager` container:

```shell
$ kubectl logs aiven-k8s-operator-controller-manager-5c9459844b-fmsk6 -c manager -f
```

Uninstall the operator:

```shell
$ operator-sdk cleanup aiven-operator
```

Create a `Secret` containing your Adyen API token:

```shell
kubectl create secret generic aiven-token --from-literal='token=${AIVEN_TOKEN}'
```

Looks for example of usage here: [config/samples folder](../config/samples)