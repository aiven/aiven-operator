A Kubernetes operator for provisioning and managing Aiven Databases and other resources as Custom Resources in your
cluster.

**This operator is work-in-progress- and is not production-ready.**

## Documentation

### Dependencies

Required for testing on any environment:
* Golang >=1.15
* [Operator SDK CLI](https://sdk.operatorframework.io/docs/installation/) >= 1.16
* [Ginkgo](https://github.com/onsi/ginkgo#global-installation)

### Different setups

- [Testing locally using env-test](docs/test-locally.md)
- [Testing in the cloud](docs/test-gcp-cloud.md)
- [Create and deploy OLM bundle](docs/create-and-deploy-olm.md)

### Examples

Samples on how to use Aiven resource can be found [here](config/samples).
