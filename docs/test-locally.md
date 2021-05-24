## Requirements

- Install the [Operator SDK CLI >=1.6](https://sdk.operatorframework.io/docs/installation/)
- Set test environment by following
  these [instructions](https://sdk.operatorframework.io/docs/building-operators/golang/testing/)
- Install and configure a local Kubernetes cluster, and any distribution will work; just make sure that `kubectl` is
  connected to a local cluster

```shell script
$ operator-sdk version
operator-sdk version: "v1.6.1-19-xxx", commit: "9b92c439354c090cdf9f178a210e1645d08b627c", kubernetes version: "v1.19.4", go version: "go1.16.3", GOOS: "linux", GOARCH: "amd64"
```

## Usage

After the local environment is ready - create a `Secret` containing your Adyen API token:

```shell script
kubectl create secret generic aiven-token --from-literal='token=${AIVEN_TOKEN}'
```

Install aiven-operator into your local Kubernetes cluster.

```shell script
make install
```

Run aiven-operator without webhooks enabled. If you have enabled webhooks in your deployments, you will need to have
cert-manager already installed in the cluster, and it isn't easy to get it running and properly configured on the local
env. Therefore we disable webhooks and will have an opportunity to validate these features on a public cloud Kubernetes
cluster. Webhooks are responsible for validating whether depended entities are already created, and it is safe to
execute reconciler of a CR. Therefore with webhooks disabled, a user has to, for example, wait until PG service is
created before attempting to create a PG database.

```shell script
make run ENABLE_WEBHOOKS=false 
```

`make run` command will send to std out all the log messages and auxiliary information regarding what is happening with
Aiven customer resource. It is useful to have it running in a separate terminal for monitoring.

Looks for example of usage here: [config/samples folder](../config/samples)