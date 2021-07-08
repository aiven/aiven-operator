# Aiven Kubernetes Operator
Provision and manage [Aiven Services](https://aiven.io/) from your Kubernetes cluster.

See the full documentation [here](https://aiven.github.io/aiven-kubernetes-operator/).

## Installation
To install the Operator, please follow the [installation instructions](https://aiven.github.io/aiven-kubernetes-operator/docs/installation/).


## Deploying PostgreSQL at Aiven
Now let's create a `PG` resource with the following YAML – please fill in your project name under in the `project` field:
```yaml
apiVersion: aiven.io/v1alpha1
kind: PG
metadata:
  name: aiven-pg
spec:

  # reads the authentication token
  authSecretRef:
    name: aiven-token
    key: token

  # stores the PostgreSQL connection information on the specified Secret
  connInfoSecretTarget:
    name: pg-connection

  project: <your-project-name>
  cloudName: google-europe-west1
  plan: hobbyist
  maintenanceWindowDow: friday
  maintenanceWindowTime: 23:00:00
  pgUserConfig:
    pg_version: '11'
```

Watch the resource being created and wait until its status is `RUNNING`:
```bash
$ watch kubectl get pg.aiven.io aiven-pg
```

After created, the Operator will create a Kubernetes Secret containing the PostgreSQL connection information:
```bash
$ kubectl describe secret pg-connection
```

Use the following [jq](https://github.com/stedolan/jq) command to decode the Secret:
```bash
$ kubectl get secret pg-connection -o json | jq '.data | map_values(@base64d)'
```

## Connecting to PostgreSQL
Let's run a `psql` command to test the database connection using the generated Secret:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: psql-test-connection
spec:
  restartPolicy: Never
  containers:
    - image: postgres:11
      name: postgres
      command: ['psql', '$(DATABASE_URI)', '-c', 'SELECT version();']
      envFrom:
      - secretRef:
          name: pg-connection
```

The Pod should the PostgreSQL version. You can verify with the following command:
```bash
$ kubectl logs psql-test-connection
                                           version                                           
---------------------------------------------------------------------------------------------
 PostgreSQL 11.12 on x86_64-pc-linux-gnu, compiled by gcc, a 68c5366192 p 6b9244f01a, 64-bit
(1 row)
```

## Contributing
We welcome and encourage contributions to this project. Please take a look at our [Contribution guide line](https://aiven.github.io/aiven-kubernetes-operator/docs/contributing/).

## License
[Apache 2](LICENSE).
