---
title: "ClickhouseGrant"
---

## Usage examples

??? example "example_2"
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseGrant
    metadata:
      name: demo-ch-grant
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: my-aiven-project
      serviceName: my-clickhouse
    
      privilegeGrants:
        - grantees:
            - user: user1
            - user: my-clickhouse-user-🦄
          privileges:
            - SELECT
            - INSERT
          database: my-db
          # If table is omitted, the privileges are granted on all tables in the database
          # If columns is omitted, the privileges are granted on all columns in the table
        - grantees:
            - role: my-role
          privileges:
            - SELECT
          database: my-db
          table: my-table
          columns:
            - col1
            - col2
    
      roleGrants:
        - roles:
            - other-role
          grantees:
            - user: my-user
            - role: my-role
    ```

??? example 
    ```yaml
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseGrant
    metadata:
      name: my-clickhouse-grant
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-clickhouse-service
    
      privilegeGrants:
        - grantees:
            - role: my-clickhouse-role
          privileges:
            - INSERT
            - SELECT
            - CREATE TABLE
            - CREATE VIEW
          database: my-clickhouse-db
      roleGrants:
        - grantees:
            - user: my-clickhouse-user
          roles:
            - my-clickhouse-role
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: Clickhouse
    metadata:
      name: my-clickhouse-service
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      cloudName: google-europe-west1
      plan: startup-16
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseDatabase
    metadata:
      name: my-clickhouse-db
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-clickhouse-service
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseUser
    metadata:
      name: my-clickhouse-user
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-clickhouse-service
    
    ---
    
    apiVersion: aiven.io/v1alpha1
    kind: ClickhouseRole
    metadata:
      name: my-clickhouse-role
    spec:
      authSecretRef:
        name: aiven-token
        key: token
    
      project: aiven-project-name
      serviceName: my-clickhouse-service
      role: my-clickhouse-role
    ```

!!! info
	To create this resource, a `Secret` containing Aiven token must be [created](/aiven-operator/authentication.html) first.

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `ClickhouseGrant`:

```shell
kubectl get clickhousegrants demo-ch-grant
```

The output is similar to the following:
```shell
Name             Project             Service Name     
demo-ch-grant    my-aiven-project    my-clickhouse    
```

## ClickhouseGrant {: #ClickhouseGrant }

ClickhouseGrant is the Schema for the ClickhouseGrants API

!!! Warning "Ambiguity in the `GRANT` syntax"

    Due to [an ambiguity](https://github.com/aiven/ospo-tracker/issues/350) in the `GRANT` syntax in Clickhouse, you should not have users and roles with the same name. It is not clear if a grant refers to the user or the role.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `ClickhouseGrant`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). ClickhouseGrantSpec defines the desired state of ClickhouseGrant. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`ClickhouseGrant`](#ClickhouseGrant)._

ClickhouseGrantSpec defines the desired state of ClickhouseGrant.

**Required**

- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.
- [`serviceName`](#spec.serviceName-property){: name='spec.serviceName-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]+$`, MaxLength: 63). Specifies the name of the service that this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`privilegeGrants`](#spec.privilegeGrants-property){: name='spec.privilegeGrants-property'} (array of objects). Configuration to grant a privilege. Privileges not in the manifest are revoked. Existing privileges are retained; new ones are granted. See below for [nested schema](#spec.privilegeGrants).
- [`roleGrants`](#spec.roleGrants-property){: name='spec.roleGrants-property'} (array of objects). Configuration to grant a role. Role grants not in the manifest are revoked. Existing role grants are retained; new ones are granted. See below for [nested schema](#spec.roleGrants).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## privilegeGrants {: #spec.privilegeGrants }

_Appears on [`spec`](#spec)._

PrivilegeGrant represents the privileges to be granted to users or roles.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#granting-privilege-syntax.

**Required**

- [`database`](#spec.privilegeGrants.database-property){: name='spec.privilegeGrants.database-property'} (string). The database that the grant refers to.
- [`grantees`](#spec.privilegeGrants.grantees-property){: name='spec.privilegeGrants.grantees-property'} (array of objects, MinItems: 1). List of grantees (users or roles) to grant the privilege to. See below for [nested schema](#spec.privilegeGrants.grantees).
- [`privileges`](#spec.privilegeGrants.privileges-property){: name='spec.privilegeGrants.privileges-property'} (array of strings). The privileges to grant, i.e. `INSERT`, `SELECT`.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#assigning-role-syntax.

**Optional**

- [`columns`](#spec.privilegeGrants.columns-property){: name='spec.privilegeGrants.columns-property'} (array of strings). The column that the grant refers to.
- [`table`](#spec.privilegeGrants.table-property){: name='spec.privilegeGrants.table-property'} (string). The tables that the grant refers to. To grant a privilege on all tables in a database, omit this field instead of writing `table: "*"`.
- [`withGrantOption`](#spec.privilegeGrants.withGrantOption-property){: name='spec.privilegeGrants.withGrantOption-property'} (boolean). If true, then the grantee (user or role) get the permission to execute the `GRANT` query.
Users can grant privileges of the same scope they have and less.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#granting-privilege-syntax.

### grantees {: #spec.privilegeGrants.grantees }

_Appears on [`spec.privilegeGrants`](#spec.privilegeGrants)._

Grantee represents a user or a role to which privileges or roles are granted.

**Optional**

- [`role`](#spec.privilegeGrants.grantees.role-property){: name='spec.privilegeGrants.grantees.role-property'} (string).
- [`user`](#spec.privilegeGrants.grantees.user-property){: name='spec.privilegeGrants.grantees.user-property'} (string).

## roleGrants {: #spec.roleGrants }

_Appears on [`spec`](#spec)._

RoleGrant represents the roles to be assigned to users or roles.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#assigning-role-syntax.

**Required**

- [`grantees`](#spec.roleGrants.grantees-property){: name='spec.roleGrants.grantees-property'} (array of objects, MinItems: 1). List of grantees (users or roles) to grant the privilege to. See below for [nested schema](#spec.roleGrants.grantees).
- [`roles`](#spec.roleGrants.roles-property){: name='spec.roleGrants.roles-property'} (array of strings, MinItems: 1). List of roles to grant to the grantees.

**Optional**

- [`withAdminOption`](#spec.roleGrants.withAdminOption-property){: name='spec.roleGrants.withAdminOption-property'} (boolean). If true, the grant is executed with `ADMIN OPTION` privilege.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#admin-option.

### grantees {: #spec.roleGrants.grantees }

_Appears on [`spec.roleGrants`](#spec.roleGrants)._

Grantee represents a user or a role to which privileges or roles are granted.

**Optional**

- [`role`](#spec.roleGrants.grantees.role-property){: name='spec.roleGrants.grantees.role-property'} (string).
- [`user`](#spec.roleGrants.grantees.user-property){: name='spec.roleGrants.grantees.user-property'} (string).

