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
