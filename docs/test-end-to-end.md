## Requirements

- Install [`kubectl` with `kuttl` plugin](https://kuttl.dev/docs/cli.html#setup-the-kuttl-kubectl-plugin)
- Install [`avn`](https://github.com/aiven/aiven-client#install-from-pypi)
- A k8s context that points to a cluster that has the `aiven-kubernetes-operator` and `cert-manager` installed
- An [aiven](https://aiven.io/) account that has access to a project named `aiven-ci-kubernetes-operator`

## Usage

```shell
AIVEN_TOKEN=<your aiven token> make test-e2e
```

Will run all `kuttl` end-to-end tests under the `test/e2e/` directory.
