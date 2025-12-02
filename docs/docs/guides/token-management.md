# Token Management

The Aiven Operator supports token management to suit different needs and security requirements. You can use centralized tokens or per-resource tokens for fine-grained control.

## Overview

The Aiven Operator needs valid Aiven API tokens to manage resources. The operator supports two approaches:

1. **Centralized Token Management**: One token configured at the operator level
2. **Per-Resource Token Management**: Individual tokens specified for each resource
3. **Mixed Approach**: Combination of both, with per-resource tokens taking precedence

## Centralized Token Management

### Benefits

- **Simplified Management**: Configure once, use everywhere
- **Easy Token Rotation**: Update in one place instead of across all namespaces
- **Reduced Duplication**: No need to create secrets in every namespace
- **Operational Simplicity**: Fewer moving parts to manage

### Setup

1. **Create the token secret in the operator's namespace:**
   ```bash
   kubectl create secret generic aiven-token \
     --namespace YOUR_NAMESPACE \
     --from-literal=token=YOUR_AIVEN_API_TOKEN
   ```

2. **Configure the operator via Helm:**
   ```yaml
   # values.yaml
   defaultTokenSecret:
     name: "aiven-token"
     key: "token"
   ```

3. **Deploy the operator:**
   ```bash
   helm install aiven-operator aiven/aiven-operator \
     --namespace YOUR_NAMESPACE \
     --values values.yaml
   ```

4. **Create resources without authSecretRef:**
   ```yaml
   apiVersion: aiven.io/v1alpha1
   kind: PostgreSQL
   metadata:
     name: my-postgres
     namespace: production
   spec:
     # No authSecretRef needed - will use default token
     project: my-aiven-project
     cloudName: google-europe-west1
     plan: startup-4
     connInfoSecretTarget:
       name: postgres-connection
   ```

## Per-Resource Token Management

### Use Cases

- **Multi-tenant environments**: Different teams need different tokens
- **Security isolation**: Separate tokens for different environments
- **Token scoping**: Different tokens with different permissions

### Setup

1. **Create token secrets in each namespace:**

   ```bash
   kubectl create secret generic aiven-token-prod \
     --namespace production \
     --from-literal=token=PRODUCTION_AIVEN_TOKEN

   kubectl create secret generic aiven-token-staging \
     --namespace staging \
     --from-literal=token=STAGING_AIVEN_TOKEN
   ```

2. **Reference tokens in resources:**

   ```yaml
   apiVersion: aiven.io/v1alpha1
   kind: PostgreSQL
   metadata:
     name: my-postgres
     namespace: production
   spec:
     authSecretRef:
       name: aiven-token-prod
       key: token
     project: production-project
     cloudName: google-europe-west1
     plan: business-4
     connInfoSecretTarget:
       name: postgres-connection
   ```

## Mixed Approach

You can combine both approaches:

```yaml
# Operator configured with default token
defaultTokenSecret:
  name: "aiven-default-token"
  key: "token"
```

```yaml
# Most resources use default token
apiVersion: aiven.io/v1alpha1
kind: Valkey
metadata:
  name: shared-cache
  namespace: development
spec:
  # Uses default token
  project: dev-project
  plan: hobbyist
  
---
# Dedicated token
apiVersion: aiven.io/v1alpha1
kind: PostgreSQL
metadata:
  name: critical-db
  namespace: production
spec:
  authSecretRef:
    name: production-token  # Override default
    key: token
  project: production-project
  plan: business-8
```

## Token Priority

The operator resolves tokens in the following priority order:

1. **Resource-level `authSecretRef`** (highest priority)
2. **Operator-level `defaultTokenSecret`** (fallback)
3. **No token** (results in error)

## Updating Tokens

**Auth secrets are not watched** - you need to manually trigger updates:

- **Centralized tokens**: Restart the operator after updating the secret
- **Per-resource tokens**: Add an annotation to trigger reconciliation

## RBAC Requirements

The operator requires specific RBAC permissions to read authentication tokens from Kubernetes secrets. The required permissions differ based on your token management approach.

### Operator Permissions

The operator runs with a `ClusterRole` that includes these secret permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
  - apiGroups:
      - ""
    resources:
      - secrets
    verbs:
      - create  # Create connection info secrets
      - delete  # Clean up secrets on resource deletion
      - get     # Read auth tokens and connection info
      - list    # List secrets for management
      - patch   # Update existing secrets
      - update  # Modify secret contents
      - watch   # Monitor secret changes
```

### Permission Scope

**Centralized Tokens:**
- Operator needs `secrets/get` permission in **its own namespace** only
- Token secret must be in the same namespace as the operator deployment

**Per-Resource Tokens:**
- Operator needs `secrets/get` permission **cluster-wide** (all namespaces)
- Each resource's token secret must be in the same namespace as the resource
- Broader permissions required - operator can access secrets in any namespace