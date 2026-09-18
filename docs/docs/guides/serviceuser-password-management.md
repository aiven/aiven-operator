# ServiceUser Password Management

The operator manages ServiceUser passwords in one of two modes, chosen by whether `connInfoSecretSource` is set on the CR.

For scheduled rotation between two existing users with a stable connection Secret,
use [ServiceUserRotation](serviceuser-rotation.md).

`ServiceUser` can manage these users and their permissions. Leave
`connInfoSecretSource` unset so it doesn't set their passwords. Set
`connInfoSecretTargetDisabled: true` at creation to disable its separate connection
Secret. Applications should read credentials from the rotation Secret.

Only `ServiceUserRotation` should change these users' passwords. Don't configure
other resources or processes to change them.

## Mode 1: Generated (no `connInfoSecretSource`)

Aiven generates the password at creation. The operator publishes it to the target secret and never modifies it again. If something else changes the password (e.g. `ALTER USER` directly in the database), the operator has nothing declared to enforce — the target secret will be updated to reflect what Aiven state, with empty value.

## Mode 2: Declared password (`connInfoSecretSource` set)

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUser
metadata:
  name: my-user
spec:
  project: my-project
  serviceName: my-service
  connInfoSecretSource:
    name: my-user-password
    passwordKey: PASSWORD
```

The source secret is the source of truth. On every reconcile the operator reads it and pushes the value to Aiven. **Direct password changes in the database will be reverted on the next reconcile cycle.**

To rotate the password, update the source secret. The operator watches it and reconciles automatically.
