A Kubernetes operator for provisioning and managing Aiven Databases and other resources 
as Custom Resources in your cluster.

**This operator is work-in-progress- and is not production-ready.**

## Usage
The following instructions apply only on a local running Kubernetes cluster.

Install the [Operator SDK CLI](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/) 
and test environment by following these [instructions](https://sdk.operatorframework.io/docs/building-operators/golang/references/envtest-setup/).

After the local environment is ready - create a `Secret` containing your Adyen API token:
```shell script
kubectl create secret generic aiven-token --from-literal='token=${AIVEN_TOKEN}'
```

Install aiven-operator into your local Kubernetes cluster.
```shell script
make install
```

Run aiven-operator without webhooks enabled.  If you have enabled webhooks in your 
deployments, you will need to have cert-manager already installed in the cluster, 
and it isn't easy to get it running and properly configured on the local env. 
Therefore we disable webhooks and will have an opportunity to validate these features 
on a public cloud Kubernetes cluster. Webhooks are responsible for validating whether 
depended entities are already created, and it is safe to execute reconciler of a CR. 
Therefore with webhooks disabled, a user has to, for example, wait until PG service 
is created before attempting to create a PG database. 
```shell script
make run ENABLE_WEBHOOKS=false 
```

`make run` command will send to std out all the log messages and auxiliary information 
regarding what is happening with Aiven customer resource. It is useful to have it running 
in a separate terminal for monitoring.

Create a Project and PG objects and wait as the operator creates and monitors the status 
of Aiven resources with the given configuration.

Create a project:
```shell script
kubectl apply -f <(echo "
apiVersion: k8s-operator.aiven.io/v1alpha1
kind: Project       
metadata:
  name: my-pr1   
spec:
  name: my-pr1       
  billing_address: NYC 
  // more fields are available
")  
```

Wait until Aiven Project created and then create a PG cluster:
```shell script
kubectl apply -f <(echo " 
apiVersion: k8s-operator.aiven.io/v1alpha1
kind: PG
metadata:
  name: my-pg1

spec:
  service_name: my-pg1
  project: my-pr1 
  cloud_name: google-europe-west1
  plan: startup-4
  maintenance_window_dow: friday
  maintenance_window_time: 23:00:00
  pg_user_config: {
    pg_version: '11'
    // more user configuration options are available
  }
")
```

Check the PG service status:

```shell script
kubectl describe PG my-pg1
```

A couple secrets will be created for each Aiven Service; example-connection and example-private-connection.

Create a service user for our PG cluster:
```shell script
kubectl apply -f <(echo "
apiVersion: k8s-operator.aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: my-usr1

spec:
  service_name: my-pg1
  project: my-pr1
  username: my-usr1
")
```

Add PG database connection pool:
```shell script
kubectl apply -f <(echo "
apiVersion: k8s-operator.aiven.io/v1alpha1
kind: ConnectionPool
metadata:
  name: my-pg-cp1

spec:
  service_name: my-pg1
  project: my-pr1
  database_name: defaultdb
  username: test123
  pool_name: pool1
  pool_mode: transaction
  pool_size: 10
")
```

## Test on GCP Kubernetes cluster

Instructions on how to build, push to the Docker registry and deploy Aiven Operator on FCP Kubernetes cluster

### GCP preparation

First of all, in your local environment install and configure gcloud command-line tool by initializing 
Cloud SDK, instructions can be found [here](https://cloud.google.com/sdk/docs/quickstart).

Then with the gcloud CLI tool or using the GCP web console create a new project and start a new 
Kubernetes cluster (can be found in Kubernetes Engine section) after that enable a Google Container 
Registry for that project.

Make sure you do have [Docker installed](https://cloud.google.com/container-registry/docs/advanced-authentication#prereqs) 
on your box, then configured it with permissions to access Docker repositories in the project. 
You can do so  with the following command:

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

### Local Open SDK preparation

Make sure you have [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager/) 
enabled. You can check if OLM is already installed by running the following command, 
which will detect the installed OLM version automatically (0.15.1 in this example):

```shell script
$ operator-sdk olm status
INFO[0000] Fetching CRDs for version "0.15.1"
INFO[0002] Fetching resources for version "0.15.1"
INFO[0002] Successfully got OLM status for version "0.15.1"

NAME                                            NAMESPACE    KIND                        STATUS
olm                                                          Namespace                   Installed
operatorgroups.operators.coreos.com                          CustomResourceDefinition    Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
...
```

All resources listed should have status `Installed`.

If OLM is not already installed, go ahead and install the latest version:

```shell script
$ operator-sdk olm install
INFO[0000] Fetching CRDs for version "latest"
INFO[0001] Fetching resources for version "latest"
INFO[0007] Creating CRDs and resources
INFO[0007]   Creating CustomResourceDefinition "clusterserviceversions.operators.coreos.com"
INFO[0007]   Creating CustomResourceDefinition "installplans.operators.coreos.com"
INFO[0007]   Creating CustomResourceDefinition "subscriptions.operators.coreos.com"
...
NAME                                            NAMESPACE    KIND                        STATUS
clusterserviceversions.operators.coreos.com                  CustomResourceDefinition    Installed
installplans.operators.coreos.com                            CustomResourceDefinition    Installed
subscriptions.operators.coreos.com                           CustomResourceDefinition    Installed
catalogsources.operators.coreos.com                          CustomResourceDefinition    Installed
...
```

**Note:** By default, `olm status` and `olm uninstall` auto-detect the OLM version installed in your cluster. 
This can fail if the installation is broken in some way, so the version of OLM can be overridden using 
the `--version` flag provided with these commands.

### Prepare an OLM bundle

Then create a bundle with the following command:

```shell
$ make bundle
/Users/ivansavciuc/go/bin/controller-gen "crd:trivialVersions=true" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
operator-sdk generate kustomize manifests -q
kustomize build config/manifests | operator-sdk generate bundle -q --overwrite --version 0.0.1  
... 
INFO[0000] Building annotations.yaml                    
INFO[0000] Writing annotations.yaml in /Users/ivansavciuc/GitHub/aiven-k8s-operator/bundle/metadata 
INFO[0000] Building Dockerfile                          
INFO[0000] Writing bundle.Dockerfile in /Users/ivansavciuc/GitHub/aiven-k8s-operator 
operator-sdk bundle validate ./bundle
INFO[0000] Found annotations file                        bundle-dir=bundle container-tool=docker
INFO[0000] Could not find optional dependencies file     bundle-dir=bundle container-tool=docker
... 
INFO[0000] All validation tests have completed successfully
```

OLM and Operator Registry consumes Operator bundles via an index image, which are composed of 
one or more bundles. To build an aiven-operator bundle, run:

```shell
make bundle-build BUNDLE_IMG=eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator:{your-username-version-tag}
docker push eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator:{your-username-version-tag}
```

Although we’ve validated on-disk manifests and metadata, we also must make sure the bundle itself is valid:

```shell
operator-sdk bundle validate eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator:{your-username-version-tag}
```

To make an index image make sure you have `opm` and `podman` installed, latest binary can be 
found [here](https://github.com/operator-framework/operator-registry/releases).

```shell
# Create the index image
opm index add --bundles eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator:{your-username-version-tag} --tag eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator-index:{your-username-version-tag}

# Push the index image
podman push eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator-index:v0.0.1
```

As the result we should have two images in GCP Container registry:
- `/aiven-operator:{your-username-version-tag}`
- `/aiven-operator-index:{your-username-version-tag}`

At this point we have our bundle and index image ready, we just need to create the required 
CatalogSource into the cluster so that we get access to our Aiven K8s Operator bundle.

```shell
OLM_NAMESPACE=$(kubectl get pods -A | grep catalog-operator | awk '{print $1}')

cat <<EOF | kubectl create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: aiven-catalog
  namespace: $OLM_NAMESPACE
spec:
  sourceType: grpc
  image: eu.gcr.io/{GCP_PROJECT_ID}/aiven-operator-index:v0.0.1
EOF
```

A pod will be created on the OLM namespace:

```shell
kubectl -n $OLM_NAMESPACE get pod -l olm.catalogSource=aiven-catalog
```
