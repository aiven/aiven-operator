# Deletion Policy

The Aiven Operator provides a deletion policy annotation that prevents deletion of Aiven resources when Kubernetes resources are removed. This feature works with all Aiven operator resources.

## Overview

By default, when you delete an Aiven operator resource (like `PostgreSQL`, `KafkaTopic`, `ServiceUser`, etc.), the operator deletes both the Kubernetes resource and the corresponding Aiven service. The `controllers.aiven.io/deletion-policy: Orphan` annotation allows you to override this behavior and preserve the Aiven resource while removing only the Kubernetes resource.

## Using the Deletion Policy

### Step 1: Add the Annotation

Add the deletion policy annotation to the resource you want to protect:

```yaml
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL  # or any other Aiven resource
metadata:
  name: my-database
  namespace: my-namespace
  annotations:
    controllers.aiven.io/deletion-policy: Orphan
spec:
  # ... existing configuration
```

### Step 2: Delete the Kubernetes Resource

Now you can safely delete the Kubernetes resource:

```bash
kubectl delete postgresql my-database -n my-namespace
```

The Kubernetes resource is deleted, but the Aiven service remains intact and continues running.