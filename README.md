# Aiven Kubernetes Operator
Provision and manage [Aiven Services](https://aiven.io/) from your Kubernetes cluster.

See the full documentation [here](https://aiven.github.io/aiven-kubernetes-operator/).

## Installation
Clone this repository:
```bash
$ git clone git@github.com:aiven/aiven-kubernetes-operator.git
$ cd aiven-kubernetes-operator
```

Install the `cert-manager` Operator:
> cert-manager is needed to correctly deploy the Operator [webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).
```bash
$ make install-cert-manager
```

Install the CRDs:
```bash
$ make install
```

Install the Operator:
```bash
$ make deploy
```

## Deploying PostgreSQL at Aiven
Sign in or create an account at [Aiven](https://console.aiven.io/signup?utm_source=github&utm_medium=organic&utm_campaign=k8s-operator&utm_content=signup), generate an [authentication token](https://help.aiven.io/en/articles/2059201-authentication-tokens) and take note of your Aiven project name.

Create a [Kubernetes Secret](https://kubernetes.io/docs/concepts/configuration/secret/) to store the generated token:
```bash
$ export AIVEN_TOKEN="<your-token-here>"
$ kubectl create secret generic aiven-token --from-literal=token="$AIVEN_TOKEN"
```

Now let's create a `PG` resource with the following YAML – please fill in your project name under in the `project` field:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: pg-sample
spec:
  project: <project>
  cloudName: google-europe-west1
  plan: hobbyist
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
  pgUserConfig:
    pg_version: '11'
```

Watch the resource being created and wait until its status is `RUNNING`:
```bash
$ watch kubectl get pg.aiven.io pg-sample
```

After created, the Operator will create a Kubernetes Secret containing the PostgreSQL connection information:
```bash
$ kubectl describe secret pg-sample-pg-secret
```

Use the following [jq](https://github.com/stedolan/jq) command to decode the Secret:
```bash
$ kubectl get secret pg-sample-pg-secret -o json | jq '.data | map_values(@base64d)'
```

## Sample App ✨
Let's deploy a [simple Golang application](https://github.com/jonatasbaldin/simple-golang-postgres) to test the database connection using the generated Secret:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: simple-golang-application
spec:
  containers:
    - image: jonatasbaldin/simple-golang-postgres
      name: simple-golang-postgres
      envFrom:
      - secretRef:
          name: pg-sample-pg-secret
```

After deploying, let's see if the application run:
```bash
$ kubectl logs simple-golang-application
2021/06/09 08:00:51 Connection to PostgreSQL successful!
```

## Contributing
We welcome and encourage contributions to this project. Please take a look at our [Contribution guide line](https://aiven.github.io/aiven-kubernetes-operator/docs/contributing/).

## License
[MIT](LICENSE).
