# ServiceUser Password Management

The operator manages ServiceUser passwords in one of two modes, chosen by whether `connInfoSecretSource` is set on the CR.

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
