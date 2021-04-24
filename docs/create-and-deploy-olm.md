## Requirements

- Install the [Operator SDK CLI >=1.6](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/)
- Install GCP command line tool `gcloud`, instructions can be found [here](https://cloud.google.com/sdk/docs/quickstart)
- Install `kubectl`, `opm` and `podman`

## Usage

Then with the gcloud CLI tool or using the GCP web console create a new project and start a new Kubernetes cluster (can
be found in Kubernetes Engine section). After that, authenticate `gcloud` to your account and set a newly created
project as the default. Then enable a Google Container Registry for that project.

Make sure you do
have [Docker installed](https://cloud.google.com/container-registry/docs/advanced-authentication#prereqs)
on your box, then configured it with permissions to access Docker repositories in the project. You can do so with the
following command:

```shell script
$ gcloud auth configure-docker
```

Your credentials are saved in your user home directory `$HOME/.docker/config.json`.

Now let`s create Kubernetes Secret based on existing Docker credentials:

```shell script
$ kubectl create secret generic regcred \
    --from-file=.dockerconfigjson=$HOME/.docker/config.json \
    --type=kubernetes.io/dockerconfigjson
```

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

Go to your K8s Engine cluster and click on the "connect" tab, which will generate a command that will configure. It will
look something like that:

```shell
gcloud container clusters get-credentials cluster-1 --zone us-central1-c --project integral-magnet-12345
```

Please copy & Paste it to your host. It should connect your local `kubectl` to the Kubernetes cluster in the GCP cloud.
To validate this, run `kubectl cluster-info`, and the output should indicate that it is connected to the GCP Kubernetes
Engine cluster.

### Building and pushing bundle to the container registry

Build Aiven Operator bundle:

```shell
make bundle IMG="eu.gcr.io/<PROJECT>/aiven-operator-bundle:<VERSION>"
```

Build and push manager container:

```shell
make docker-build docker-push IMG="eu.gcr.io/<PROJECT>/aiven-operator:<VERSION>"
```

Build and push bundle image:

```shell
make bundle-build bundle-push BUNDLE_IMG="eu.gcr.io/<PROJECT>/aiven-operator-bundle:<VERSION>"
```

VValidating bundle:

```shell
operator-sdk bundle validate eu.gcr.io/<PROJECT>/aiven-operator-bundle:<VERSION>
```

As the result we should have two images in GCP Container registry:

- `/aiven-operator:<VERSION>`
- `/aiven-operator-bundle:<VERSION>`

At this point we have our bundle ready, we just need to create the required CatalogSource into the cluster so that we
get access to our Aiven K8s Operator bundle. To do that please
follow [GCP cloud testing instructions](test-gcp-cloud.md). 