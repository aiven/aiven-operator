---
title: "Service user rotation"
---

# Service user rotation

`ServiceUserRotation` rotates credentials between two existing Aiven service users.
Applications read them from one Kubernetes Secret. The previous user's password
stays unchanged until that user is selected again, giving applications time to
switch.

## How rotation works

`rotationInterval` sets the delay from publishing credentials to the next
scheduled switch to another user. Users are selected in list order. Each
rotation generates a secure random password. The generator isn't configurable.
The first publication changes the first user's password immediately.

With two users, `a` and `b`, and a 30-day interval, the sequence looks like this.
`A1`, `B1` and `A2` stand for different generated passwords.

| Time | User in Secret | Password in Secret | Previous password |
| --- | --- | --- | --- |
| Day 0 | `a` | `A1` | — |
| Day 30 | `b` | `B1` | `A1` still works |
| Day 60 | `a` | `A2` | `B1` still works |

The Secret switches every 30 days, but `A1` normally lasts about 60 days: 30 as
the current password and another 30 after the switch to `b`. It's replaced while
preparing `a` for the next publication.

Always use this Secret as the source of credentials for your applications.
After a rotation changes the credentials in it, reload them in your applications
or restart the applications to pick up the new values.

You normally have one rotation interval after the switch, assuming the interval
stays unchanged. Complete the switch before the previous user is selected again.
The operator doesn't restart workloads or wait for applications to confirm the
switch.

Connection refreshes don't move the next deadline. Changing the interval
recalculates it from the last publication time, so shortening the interval can
make rotation immediately due.

The operator doesn't set a password expiry time. A password stops working when
it's replaced, so downtime or failed rotations can extend its lifetime.

If rotation is overdue after downtime, the operator rotates once and starts a
new interval. It doesn't catch up on missed windows. A valid pending rotation
is retried even if you extend the interval.

## Set up rotation

Use an existing, operational Aiven service and a token with the
[required permissions](../resources/serviceuserrotation.md#required-permissions).
Reading connection details also needs `service:secrets:read`.

Create both users and give them the permissions your application needs before
enabling rotation. Use `ServiceUser` resources or prepare them another way.
`ServiceUserRotation` doesn't create, recreate or delete users, and it doesn't
configure or check their roles and grants.

When using `ServiceUser`, leave `connInfoSecretSource` unset so it doesn't set
the password. Set `connInfoSecretTargetDisabled: true` at creation to disable
its separate connection Secret. This field can't be changed later.
See the [ServiceUser example for rotation](../resources/examples/serviceuser-for-rotation.yaml)
and create one resource for each username.

This example rotates between two users every 30 days:

```yaml
apiVersion: aiven.io/v1alpha1
kind: ServiceUserRotation
metadata:
  name: application-users
spec:
  authSecretRef:
    name: aiven-token
    key: token

  project: my-aiven-project
  serviceName: my-postgresql

  usernames:
    - application-user-a
    - application-user-b
  rotationInterval: 720h

  connInfoSecretTarget:
    name: application-database-credentials
    prefix: SERVICEUSER_
```

Choose exactly two distinct usernames, excluding `avnadmin`. The interval must be
at least one hour. The user list, its order, the Secret name and the key prefix
can't be changed after creation.

Use a dedicated Secret. Let this rotation resource be the only automation that
changes these users' passwords. Applications should use its Secret rather than
credentials from individual `ServiceUser` resources.

## The connection Secret

Applications read `SERVICEUSER_USERNAME`, `SERVICEUSER_PASSWORD` and the connection
fields from `application-database-credentials`. These names follow the example
above. You can choose a different prefix when creating the resource.

Between rotations, the operator refreshes endpoints, certificates and the CA
while keeping the published password. It removes optional Kafka SASL and Schema
Registry endpoint keys when those endpoints disappear. Other data keys are
preserved. Labels and annotations follow `spec.connInfoSecretTarget`, so manual
changes to them are overwritten.

The same Secret stores the rotation state:

- `aiven-rotation-published-at`: the last publication time, in RFC3339Nano format.
- `aiven-rotation-desired-username`: the next user.
- `aiven-rotation-desired-password`: the password prepared for that user.

Before changing a password in Aiven, the operator saves the next credentials in
these keys. Retries use the same saved password, including after a restart or a
lost API response. Once the new connection details are ready, one Secret update
publishes the credentials, records the time and removes the pending pair.

Before the first publication, the Secret contains only the pending pair.
Applications need the public connection keys to connect. Tools that reload
workloads on any Secret change may also react to preparation, before those
public keys change.

## Password changes outside rotation

Passwords go from the operator to Aiven during rotation. The operator doesn't
detect or undo password changes made elsewhere, and it doesn't copy passwords
from Aiven. Restoring an old password could break clients using a new one.

If someone changes the active password, the Secret keeps the old value. New
connections may fail until the next successful rotation or manual recovery.

If the active user is deleted, reconciliation fails while reading its connection
details. Its owner must restore the user and its permissions. Recreating it
doesn't restore the published password, and rotation still leaves that password
unchanged until the next scheduled switch or manual recovery.

## Errors and recovery

A failed password request leaves the pending pair saved for a retry. The current
username, password and publication time stay unchanged. If there's a published
user, its connection details are refreshed before the request. Errors reading
Aiven data or writing the Secret can still block that refresh.

A missing inactive user doesn't block connection refreshes. When its turn comes,
the password request fails and the operator retries the saved candidate. It
doesn't create the missing user.

Don't edit the managed credential keys. The operator discards incomplete pending
pairs and candidates outside the pool, but it doesn't repair arbitrary changes.
Discarding an invalid pair doesn't bring the next rotation forward.

Back up the whole Secret, including the rotation state:

- Deleting the Secret loses that state. The operator starts again with the first
  user and a new password.
- An old backup may contain a password that no longer works. Restoring the Secret
  doesn't restore that password in Aiven.
- An invalid publication time blocks credential updates. Restore it from a
  backup. Removing the timestamp is treated as no previous publication and can
  cause an immediate password change.

The `Error` condition reports the last reconciliation failure and disappears
after a successful reconcile. It doesn't tell you whether applications can
connect. Status shows the last successful reconciliation. Before the first one,
its username and times are empty. Changing status doesn't control rotation.
Monitor application connections as well as operator errors.

## Deletion

Deleting `ServiceUserRotation` stops rotation. The Aiven users, their passwords
and permissions stay unchanged. Their lifecycle remains with `ServiceUser` or
the external process that manages them.

The owned connection Secret is deleted. Copy any credentials you need to keep
elsewhere before deleting the rotation resource.
