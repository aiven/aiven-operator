---
title: "ClickhouseGrant"
---

## Usage example

??? example 
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
          # If table is not specified, the privileges are granted on all tables in the database
          # If columns is not specified, the privileges are granted on all columns in the table
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

## ClickhouseGrant {: #ClickhouseGrant }

ClickhouseGrant is the Schema for the ClickhouseGrants API.

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
- [`privilegeGrants`](#spec.privilegeGrants-property){: name='spec.privilegeGrants-property'} (array of objects). Configuration to grant a privilege. See below for [nested schema](#spec.privilegeGrants).
- [`roleGrants`](#spec.roleGrants-property){: name='spec.roleGrants-property'} (array of objects). Configuration to grant a role. See below for [nested schema](#spec.roleGrants).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## privilegeGrants {: #spec.privilegeGrants }

_Appears on [`spec`](#spec)._

Configuration to grant a privilege.

**Required**

- [`database`](#spec.privilegeGrants.database-property){: name='spec.privilegeGrants.database-property'} (string). The database that the grant refers to.
- [`grantees`](#spec.privilegeGrants.grantees-property){: name='spec.privilegeGrants.grantees-property'} (array of objects, MinItems: 1). List of grantees (users or roles) to grant the privilege to. See below for [nested schema](#spec.privilegeGrants.grantees).
- [`privileges`](#spec.privilegeGrants.privileges-property){: name='spec.privilegeGrants.privileges-property'} (array of strings). The privileges to grant, i.e. `INSERT`, `SELECT`.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#assigning-role-syntax.

**Optional**

- [`columns`](#spec.privilegeGrants.columns-property){: name='spec.privilegeGrants.columns-property'} (array of strings). The column that the grant refers to.
- [`table`](#spec.privilegeGrants.table-property){: name='spec.privilegeGrants.table-property'} (string). The tables that the grant refers to.
- [`withGrantOption`](#spec.privilegeGrants.withGrantOption-property){: name='spec.privilegeGrants.withGrantOption-property'} (boolean). If true, then the grantee (user or role) get the permission to execute the `GRANT`` query.
Users can grant privileges of the same scope they have and less.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#granting-privilege-syntax.

### grantees {: #spec.privilegeGrants.grantees }

_Appears on [`spec.privilegeGrants`](#spec.privilegeGrants)._

List of grantees (users or roles) to grant the privilege to.

**Optional**

- [`role`](#spec.privilegeGrants.grantees.role-property){: name='spec.privilegeGrants.grantees.role-property'} (string).
- [`user`](#spec.privilegeGrants.grantees.user-property){: name='spec.privilegeGrants.grantees.user-property'} (string).

## roleGrants {: #spec.roleGrants }

_Appears on [`spec`](#spec)._

Configuration to grant a role.

**Required**

- [`grantees`](#spec.roleGrants.grantees-property){: name='spec.roleGrants.grantees-property'} (array of objects, MinItems: 1). List of grantees (users or roles) to grant the privilege to. See below for [nested schema](#spec.roleGrants.grantees).
- [`roles`](#spec.roleGrants.roles-property){: name='spec.roleGrants.roles-property'} (array of strings, MinItems: 1). List of roles to grant to the grantees.

**Optional**

- [`withAdminOption`](#spec.roleGrants.withAdminOption-property){: name='spec.roleGrants.withAdminOption-property'} (boolean). If true, the grant is executed with `ADMIN OPTION` privilege.
See https://clickhouse.com/docs/en/sql-reference/statements/grant#admin-option.

### grantees {: #spec.roleGrants.grantees }

_Appears on [`spec.roleGrants`](#spec.roleGrants)._

List of grantees (users or roles) to grant the privilege to.

**Optional**

- [`role`](#spec.roleGrants.grantees.role-property){: name='spec.roleGrants.grantees.role-property'} (string).
- [`user`](#spec.roleGrants.grantees.user-property){: name='spec.roleGrants.grantees.user-property'} (string).

