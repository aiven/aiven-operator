---
title: "Grafana"
---

## Usage example

!!! note "Prerequisites"
	* A Kubernetes cluster with the operator installed using [helm](../installation/helm.md), [kubectl](../installation/kubectl.md) or [kind](../contributing/developer-guide.md) (for local development).
	* A Kubernetes [Secret](../authentication.md) with an Aiven authentication token.

```yaml linenums="1"
apiVersion: aiven.io/v1alpha1
kind: Grafana
metadata:
  name: my-grafana
spec:
  authSecretRef:
    name: aiven-token
    key: token

  connInfoSecretTarget:
    name: grafana-secret
    prefix: MY_SECRET_PREFIX_
    annotations:
      foo: bar
    labels:
      baz: egg

  project: my-aiven-project
  cloudName: google-europe-west1
  plan: startup-1

  maintenanceWindowDow: sunday
  maintenanceWindowTime: 11:00:00

  userConfig:
    public_access:
      grafana: true
    ip_filter:
      - network: 0.0.0.0
        description: whatever
      - network: 10.20.0.0/16
```

Apply the resource with:

```shell
kubectl apply -f example.yaml
```

Verify the newly created `Grafana`:

```shell
kubectl get grafanas my-grafana
```

The output is similar to the following:
```shell
Name          Project             Region                 Plan         State      
my-grafana    my-aiven-project    google-europe-west1    startup-1    RUNNING    
```

To view the details of the `Secret`, use the following command:
```shell
kubectl describe secret grafana-secret
```

You can use the [jq](https://github.com/jqlang/jq) to quickly decode the `Secret`:

```shell
kubectl get secret grafana-secret -o json | jq '.data | map_values(@base64d)'
```

The output is similar to the following:

```{ .json .no-copy }
{
	"GRAFANA_HOST": "<secret>",
	"GRAFANA_PORT": "<secret>",
	"GRAFANA_USER": "<secret>",
	"GRAFANA_PASSWORD": "<secret>",
	"GRAFANA_URI": "<secret>",
	"GRAFANA_HOSTS": "<secret>",
}
```

---

## Grafana {: #Grafana }

Grafana is the Schema for the grafanas API.

!!! Info "Exposes secret keys"

    `GRAFANA_HOST`, `GRAFANA_PORT`, `GRAFANA_USER`, `GRAFANA_PASSWORD`, `GRAFANA_URI`, `GRAFANA_HOSTS`.

**Required**

- [`apiVersion`](#apiVersion-property){: name='apiVersion-property'} (string). Value `aiven.io/v1alpha1`.
- [`kind`](#kind-property){: name='kind-property'} (string). Value `Grafana`.
- [`metadata`](#metadata-property){: name='metadata-property'} (object). Data that identifies the object, including a `name` string and optional `namespace`.
- [`spec`](#spec-property){: name='spec-property'} (object). GrafanaSpec defines the desired state of Grafana. See below for [nested schema](#spec).

## spec {: #spec }

_Appears on [`Grafana`](#Grafana)._

GrafanaSpec defines the desired state of Grafana.

**Required**

- [`plan`](#spec.plan-property){: name='spec.plan-property'} (string, MaxLength: 128). Subscription plan.
- [`project`](#spec.project-property){: name='spec.project-property'} (string, Immutable, Pattern: `^[a-zA-Z0-9_-]+$`, MaxLength: 63). Identifies the project this resource belongs to.

**Optional**

- [`authSecretRef`](#spec.authSecretRef-property){: name='spec.authSecretRef-property'} (object). Authentication reference to Aiven token in a secret. See below for [nested schema](#spec.authSecretRef).
- [`cloudName`](#spec.cloudName-property){: name='spec.cloudName-property'} (string, MaxLength: 256). Cloud the service runs in.
- [`connInfoSecretTarget`](#spec.connInfoSecretTarget-property){: name='spec.connInfoSecretTarget-property'} (object). Secret configuration. See below for [nested schema](#spec.connInfoSecretTarget).
- [`connInfoSecretTargetDisabled`](#spec.connInfoSecretTargetDisabled-property){: name='spec.connInfoSecretTargetDisabled-property'} (boolean, Immutable). When true, the secret containing connection information will not be created, defaults to false. This field cannot be changed after resource creation.
- [`disk_space`](#spec.disk_space-property){: name='spec.disk_space-property'} (string, Pattern: `(?i)^[1-9][0-9]*(GiB|G)?$`). The disk space of the service, possible values depend on the service type, the cloud provider and the project.
    Reducing will result in the service re-balancing.
    The removal of this field does not change the value.
- [`maintenanceWindowDow`](#spec.maintenanceWindowDow-property){: name='spec.maintenanceWindowDow-property'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc.
- [`maintenanceWindowTime`](#spec.maintenanceWindowTime-property){: name='spec.maintenanceWindowTime-property'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format.
- [`powered`](#spec.powered-property){: name='spec.powered-property'} (boolean, Default value: `true`). Determines the power state of the service. When `true` (default), the service is running.
    When `false`, the service is powered off.
    For more information please see [Aiven documentation](https://aiven.io/docs/platform/concepts/service-power-cycle).
    Note that:
    - Annotation `controllers.aiven.io/instance-is-running` will be set to `false`
    - Services cannot be created in a powered off state (the value is ignored during creation)
    - It is highly recommended to not run dependent resources when the service is powered off.
      Creating a new resource or updating an existing resource that depends on a powered off service will result in an error.
      Existing resources will need to be manually recreated after the service is powered on.
    - For Kafka services with backups: Topic configuration, schemas and connectors are all backed up, but not the data in topics. All topic data is lost on power off.
    - For Kafka services without backups: Topic configurations including all topic data is lost on power off.
- [`projectVPCRef`](#spec.projectVPCRef-property){: name='spec.projectVPCRef-property'} (object). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See below for [nested schema](#spec.projectVPCRef).
- [`projectVpcId`](#spec.projectVpcId-property){: name='spec.projectVpcId-property'} (string, MaxLength: 36). Identifier of the VPC the service should be in, if any.
- [`serviceIntegrations`](#spec.serviceIntegrations-property){: name='spec.serviceIntegrations-property'} (array of objects, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See below for [nested schema](#spec.serviceIntegrations).
- [`tags`](#spec.tags-property){: name='spec.tags-property'} (object, AdditionalProperties: string). Tags are key-value pairs that allow you to categorize services.
- [`technicalEmails`](#spec.technicalEmails-property){: name='spec.technicalEmails-property'} (array of objects, MaxItems: 10). Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability. See below for [nested schema](#spec.technicalEmails).
- [`terminationProtection`](#spec.terminationProtection-property){: name='spec.terminationProtection-property'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services.
- [`userConfig`](#spec.userConfig-property){: name='spec.userConfig-property'} (object). Cassandra specific user configuration options. See below for [nested schema](#spec.userConfig).

## authSecretRef {: #spec.authSecretRef }

_Appears on [`spec`](#spec)._

Authentication reference to Aiven token in a secret.

**Required**

- [`key`](#spec.authSecretRef.key-property){: name='spec.authSecretRef.key-property'} (string, MinLength: 1).
- [`name`](#spec.authSecretRef.name-property){: name='spec.authSecretRef.name-property'} (string, MinLength: 1).

## connInfoSecretTarget {: #spec.connInfoSecretTarget }

_Appears on [`spec`](#spec)._

Secret configuration.

**Required**

- [`name`](#spec.connInfoSecretTarget.name-property){: name='spec.connInfoSecretTarget.name-property'} (string, Immutable). Name of the secret resource to be created. By default, it is equal to the resource name.

**Optional**

- [`annotations`](#spec.connInfoSecretTarget.annotations-property){: name='spec.connInfoSecretTarget.annotations-property'} (object, AdditionalProperties: string). Annotations added to the secret.
- [`labels`](#spec.connInfoSecretTarget.labels-property){: name='spec.connInfoSecretTarget.labels-property'} (object, AdditionalProperties: string). Labels added to the secret.
- [`prefix`](#spec.connInfoSecretTarget.prefix-property){: name='spec.connInfoSecretTarget.prefix-property'} (string). Prefix for the secret's keys.
    Added "as is" without any transformations.
    By default, is equal to the kind name in uppercase + underscore, e.g. `KAFKA_`, `REDIS_`, etc.

## projectVPCRef {: #spec.projectVPCRef }

_Appears on [`spec`](#spec)._

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#spec.projectVPCRef.name-property){: name='spec.projectVPCRef.name-property'} (string, MinLength: 1).

**Optional**

- [`namespace`](#spec.projectVPCRef.namespace-property){: name='spec.projectVPCRef.namespace-property'} (string, MinLength: 1).

## serviceIntegrations {: #spec.serviceIntegrations }

_Appears on [`spec`](#spec)._

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#spec.serviceIntegrations.integrationType-property){: name='spec.serviceIntegrations.integrationType-property'} (string, Enum: `read_replica`).
- [`sourceServiceName`](#spec.serviceIntegrations.sourceServiceName-property){: name='spec.serviceIntegrations.sourceServiceName-property'} (string, MinLength: 1, MaxLength: 64).

## technicalEmails {: #spec.technicalEmails }

_Appears on [`spec`](#spec)._

Defines the email addresses that will receive alerts about upcoming maintenance updates or warnings about service instability.

**Required**

- [`email`](#spec.technicalEmails.email-property){: name='spec.technicalEmails.email-property'} (string). Email address.

## userConfig {: #spec.userConfig }

_Appears on [`spec`](#spec)._

Cassandra specific user configuration options.

**Optional**

- [`additional_backup_regions`](#spec.userConfig.additional_backup_regions-property){: name='spec.userConfig.additional_backup_regions-property'} (array of strings, MaxItems: 1). Additional Cloud Regions for Backup Replication.
- [`alerting_enabled`](#spec.userConfig.alerting_enabled-property){: name='spec.userConfig.alerting_enabled-property'} (boolean). DEPRECATED: setting has no effect with Grafana 11 and onward. Enable or disable Grafana legacy alerting functionality. This should not be enabled with unified_alerting_enabled.
- [`alerting_error_or_timeout`](#spec.userConfig.alerting_error_or_timeout-property){: name='spec.userConfig.alerting_error_or_timeout-property'} (string, Enum: `alerting`, `keep_state`). Default error or timeout setting for new alerting rules.
- [`alerting_max_annotations_to_keep`](#spec.userConfig.alerting_max_annotations_to_keep-property){: name='spec.userConfig.alerting_max_annotations_to_keep-property'} (integer, Minimum: 0, Maximum: 1000000). Max number of alert annotations that Grafana stores. 0 (default) keeps all alert annotations.
- [`alerting_nodata_or_nullvalues`](#spec.userConfig.alerting_nodata_or_nullvalues-property){: name='spec.userConfig.alerting_nodata_or_nullvalues-property'} (string, Enum: `alerting`, `keep_state`, `no_data`, `ok`). Default value for 'no data or null values' for new alerting rules.
- [`allow_embedding`](#spec.userConfig.allow_embedding-property){: name='spec.userConfig.allow_embedding-property'} (boolean). Allow embedding Grafana dashboards with iframe/frame/object/embed tags. Disabled by default to limit impact of clickjacking.
- [`auth_azuread`](#spec.userConfig.auth_azuread-property){: name='spec.userConfig.auth_azuread-property'} (object). Azure AD OAuth integration. See below for [nested schema](#spec.userConfig.auth_azuread).
- [`auth_basic_enabled`](#spec.userConfig.auth_basic_enabled-property){: name='spec.userConfig.auth_basic_enabled-property'} (boolean). Enable or disable basic authentication form, used by Grafana built-in login.
- [`auth_generic_oauth`](#spec.userConfig.auth_generic_oauth-property){: name='spec.userConfig.auth_generic_oauth-property'} (object). Generic OAuth integration. See below for [nested schema](#spec.userConfig.auth_generic_oauth).
- [`auth_github`](#spec.userConfig.auth_github-property){: name='spec.userConfig.auth_github-property'} (object). Github Auth integration. See below for [nested schema](#spec.userConfig.auth_github).
- [`auth_gitlab`](#spec.userConfig.auth_gitlab-property){: name='spec.userConfig.auth_gitlab-property'} (object). GitLab Auth integration. See below for [nested schema](#spec.userConfig.auth_gitlab).
- [`auth_google`](#spec.userConfig.auth_google-property){: name='spec.userConfig.auth_google-property'} (object). Google Auth integration. See below for [nested schema](#spec.userConfig.auth_google).
- [`cookie_samesite`](#spec.userConfig.cookie_samesite-property){: name='spec.userConfig.cookie_samesite-property'} (string, Enum: `lax`, `none`, `strict`). Cookie SameSite attribute: `strict` prevents sending cookie for cross-site requests, effectively disabling direct linking from other sites to Grafana. `lax` is the default value.
- [`custom_domain`](#spec.userConfig.custom_domain-property){: name='spec.userConfig.custom_domain-property'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name.
- [`dashboard_previews_enabled`](#spec.userConfig.dashboard_previews_enabled-property){: name='spec.userConfig.dashboard_previews_enabled-property'} (boolean). Enable browsing of dashboards in grid (pictures) mode. This feature is new in Grafana 9 and is quite resource intensive. It may cause low-end plans to work more slowly while the dashboard previews are rendering.
- [`dashboard_scenes_enabled`](#spec.userConfig.dashboard_scenes_enabled-property){: name='spec.userConfig.dashboard_scenes_enabled-property'} (boolean). Enable use of the Grafana Scenes Library as the dashboard engine. i.e. the `dashboardScene` feature flag. Upstream blog post at https://grafana.com/blog/2024/10/31/grafana-dashboards-are-now-powered-by-scenes-big-changes-same-ui/.
- [`dashboards_min_refresh_interval`](#spec.userConfig.dashboards_min_refresh_interval-property){: name='spec.userConfig.dashboards_min_refresh_interval-property'} (string, Pattern: `^[0-9]+(ms|s|m|h|d)$`, MaxLength: 16). Signed sequence of decimal numbers, followed by a unit suffix (ms, s, m, h, d), e.g. 30s, 1h.
- [`dashboards_versions_to_keep`](#spec.userConfig.dashboards_versions_to_keep-property){: name='spec.userConfig.dashboards_versions_to_keep-property'} (integer, Minimum: 1, Maximum: 100). Dashboard versions to keep per dashboard.
- [`dataproxy_send_user_header`](#spec.userConfig.dataproxy_send_user_header-property){: name='spec.userConfig.dataproxy_send_user_header-property'} (boolean). Send `X-Grafana-User` header to data source.
- [`dataproxy_timeout`](#spec.userConfig.dataproxy_timeout-property){: name='spec.userConfig.dataproxy_timeout-property'} (integer, Minimum: 15, Maximum: 90). Timeout for data proxy requests in seconds.
- [`date_formats`](#spec.userConfig.date_formats-property){: name='spec.userConfig.date_formats-property'} (object). Grafana date format specifications. See below for [nested schema](#spec.userConfig.date_formats).
- [`disable_gravatar`](#spec.userConfig.disable_gravatar-property){: name='spec.userConfig.disable_gravatar-property'} (boolean). Set to true to disable gravatar. Defaults to false (gravatar is enabled).
- [`editors_can_admin`](#spec.userConfig.editors_can_admin-property){: name='spec.userConfig.editors_can_admin-property'} (boolean). Editors can manage folders, teams and dashboards created by them.
- [`external_image_storage`](#spec.userConfig.external_image_storage-property){: name='spec.userConfig.external_image_storage-property'} (object). External image store settings. See below for [nested schema](#spec.userConfig.external_image_storage).
- [`google_analytics_ua_id`](#spec.userConfig.google_analytics_ua_id-property){: name='spec.userConfig.google_analytics_ua_id-property'} (string, Pattern: `^(G|UA|YT|MO)-[a-zA-Z0-9-]+$`, MaxLength: 64). Google Analytics ID.
- [`ip_filter`](#spec.userConfig.ip_filter-property){: name='spec.userConfig.ip_filter-property'} (array of objects, MaxItems: 2048). Allow incoming connections from CIDR address block, e.g. `10.20.0.0/16`. See below for [nested schema](#spec.userConfig.ip_filter).
- [`metrics_enabled`](#spec.userConfig.metrics_enabled-property){: name='spec.userConfig.metrics_enabled-property'} (boolean). Enable Grafana's /metrics endpoint.
- [`oauth_allow_insecure_email_lookup`](#spec.userConfig.oauth_allow_insecure_email_lookup-property){: name='spec.userConfig.oauth_allow_insecure_email_lookup-property'} (boolean). Enforce user lookup based on email instead of the unique ID provided by the IdP.
- [`private_access`](#spec.userConfig.private_access-property){: name='spec.userConfig.private_access-property'} (object). Allow access to selected service ports from private networks. See below for [nested schema](#spec.userConfig.private_access).
- [`privatelink_access`](#spec.userConfig.privatelink_access-property){: name='spec.userConfig.privatelink_access-property'} (object). Allow access to selected service components through Privatelink. See below for [nested schema](#spec.userConfig.privatelink_access).
- [`project_to_fork_from`](#spec.userConfig.project_to_fork_from-property){: name='spec.userConfig.project_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created.
- [`public_access`](#spec.userConfig.public_access-property){: name='spec.userConfig.public_access-property'} (object). Allow access to selected service ports from the public Internet. See below for [nested schema](#spec.userConfig.public_access).
- [`recovery_basebackup_name`](#spec.userConfig.recovery_basebackup_name-property){: name='spec.userConfig.recovery_basebackup_name-property'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service.
- [`service_log`](#spec.userConfig.service_log-property){: name='spec.userConfig.service_log-property'} (boolean). Store logs for the service so that they are available in the HTTP API and console.
- [`service_to_fork_from`](#spec.userConfig.service_to_fork_from-property){: name='spec.userConfig.service_to_fork_from-property'} (string, Immutable, Pattern: `^[a-z][-a-z0-9]{0,63}$|^$`, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created.
- [`smtp_server`](#spec.userConfig.smtp_server-property){: name='spec.userConfig.smtp_server-property'} (object). SMTP server settings. See below for [nested schema](#spec.userConfig.smtp_server).
- [`static_ips`](#spec.userConfig.static_ips-property){: name='spec.userConfig.static_ips-property'} (boolean). Use static public IP addresses.
- [`unified_alerting_enabled`](#spec.userConfig.unified_alerting_enabled-property){: name='spec.userConfig.unified_alerting_enabled-property'} (boolean). Enable or disable Grafana unified alerting functionality. By default this is enabled and any legacy alerts will be migrated on upgrade to Grafana 9+. To stay on legacy alerting, set unified_alerting_enabled to false and alerting_enabled to true. See https://grafana.com/docs/grafana/latest/alerting/ for more details.
- [`user_auto_assign_org`](#spec.userConfig.user_auto_assign_org-property){: name='spec.userConfig.user_auto_assign_org-property'} (boolean). Auto-assign new users on signup to main organization. Defaults to false.
- [`user_auto_assign_org_role`](#spec.userConfig.user_auto_assign_org_role-property){: name='spec.userConfig.user_auto_assign_org_role-property'} (string, Enum: `Admin`, `Editor`, `Viewer`). Set role for new signups. Defaults to Viewer.
- [`viewers_can_edit`](#spec.userConfig.viewers_can_edit-property){: name='spec.userConfig.viewers_can_edit-property'} (boolean). Users with view-only permission can edit but not save dashboards.
- [`wal`](#spec.userConfig.wal-property){: name='spec.userConfig.wal-property'} (boolean). Setting to enable/disable Write-Ahead Logging. The default value is false (disabled).

### auth_azuread {: #spec.userConfig.auth_azuread }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Azure AD OAuth integration.

**Required**

- [`auth_url`](#spec.userConfig.auth_azuread.auth_url-property){: name='spec.userConfig.auth_azuread.auth_url-property'} (string, MaxLength: 2048). Authorization URL.
- [`client_id`](#spec.userConfig.auth_azuread.client_id-property){: name='spec.userConfig.auth_azuread.client_id-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider.
- [`client_secret`](#spec.userConfig.auth_azuread.client_secret-property){: name='spec.userConfig.auth_azuread.client_secret-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider.
- [`token_url`](#spec.userConfig.auth_azuread.token_url-property){: name='spec.userConfig.auth_azuread.token_url-property'} (string, MaxLength: 2048). Token URL.

**Optional**

- [`allow_sign_up`](#spec.userConfig.auth_azuread.allow_sign_up-property){: name='spec.userConfig.auth_azuread.allow_sign_up-property'} (boolean). Automatically sign-up users on successful sign-in.
- [`allowed_domains`](#spec.userConfig.auth_azuread.allowed_domains-property){: name='spec.userConfig.auth_azuread.allowed_domains-property'} (array of strings, MaxItems: 50). Allowed domains.
- [`allowed_groups`](#spec.userConfig.auth_azuread.allowed_groups-property){: name='spec.userConfig.auth_azuread.allowed_groups-property'} (array of strings, MaxItems: 50). Require users to belong to one of given groups.

### auth_generic_oauth {: #spec.userConfig.auth_generic_oauth }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Generic OAuth integration.

**Required**

- [`api_url`](#spec.userConfig.auth_generic_oauth.api_url-property){: name='spec.userConfig.auth_generic_oauth.api_url-property'} (string, MaxLength: 2048). API URL.
- [`auth_url`](#spec.userConfig.auth_generic_oauth.auth_url-property){: name='spec.userConfig.auth_generic_oauth.auth_url-property'} (string, MaxLength: 2048). Authorization URL.
- [`client_id`](#spec.userConfig.auth_generic_oauth.client_id-property){: name='spec.userConfig.auth_generic_oauth.client_id-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider.
- [`client_secret`](#spec.userConfig.auth_generic_oauth.client_secret-property){: name='spec.userConfig.auth_generic_oauth.client_secret-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider.
- [`token_url`](#spec.userConfig.auth_generic_oauth.token_url-property){: name='spec.userConfig.auth_generic_oauth.token_url-property'} (string, MaxLength: 2048). Token URL.

**Optional**

- [`allow_sign_up`](#spec.userConfig.auth_generic_oauth.allow_sign_up-property){: name='spec.userConfig.auth_generic_oauth.allow_sign_up-property'} (boolean). Automatically sign-up users on successful sign-in.
- [`allowed_domains`](#spec.userConfig.auth_generic_oauth.allowed_domains-property){: name='spec.userConfig.auth_generic_oauth.allowed_domains-property'} (array of strings, MaxItems: 50). Allowed domains.
- [`allowed_organizations`](#spec.userConfig.auth_generic_oauth.allowed_organizations-property){: name='spec.userConfig.auth_generic_oauth.allowed_organizations-property'} (array of strings, MaxItems: 50). Require user to be member of one of the listed organizations.
- [`auto_login`](#spec.userConfig.auth_generic_oauth.auto_login-property){: name='spec.userConfig.auth_generic_oauth.auto_login-property'} (boolean). Allow users to bypass the login screen and automatically log in.
- [`name`](#spec.userConfig.auth_generic_oauth.name-property){: name='spec.userConfig.auth_generic_oauth.name-property'} (string, Pattern: `^[a-zA-Z0-9_\- ]+$`, MaxLength: 128). Name of the OAuth integration.
- [`scopes`](#spec.userConfig.auth_generic_oauth.scopes-property){: name='spec.userConfig.auth_generic_oauth.scopes-property'} (array of strings, MaxItems: 50). OAuth scopes.
- [`use_refresh_token`](#spec.userConfig.auth_generic_oauth.use_refresh_token-property){: name='spec.userConfig.auth_generic_oauth.use_refresh_token-property'} (boolean). Set to true to use refresh token and check access token expiration.

### auth_github {: #spec.userConfig.auth_github }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Github Auth integration.

**Required**

- [`client_id`](#spec.userConfig.auth_github.client_id-property){: name='spec.userConfig.auth_github.client_id-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider.
- [`client_secret`](#spec.userConfig.auth_github.client_secret-property){: name='spec.userConfig.auth_github.client_secret-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider.

**Optional**

- [`allow_sign_up`](#spec.userConfig.auth_github.allow_sign_up-property){: name='spec.userConfig.auth_github.allow_sign_up-property'} (boolean). Automatically sign-up users on successful sign-in.
- [`allowed_organizations`](#spec.userConfig.auth_github.allowed_organizations-property){: name='spec.userConfig.auth_github.allowed_organizations-property'} (array of strings, MaxItems: 50). Require users to belong to one of given organizations.
- [`auto_login`](#spec.userConfig.auth_github.auto_login-property){: name='spec.userConfig.auth_github.auto_login-property'} (boolean). Allow users to bypass the login screen and automatically log in.
- [`skip_org_role_sync`](#spec.userConfig.auth_github.skip_org_role_sync-property){: name='spec.userConfig.auth_github.skip_org_role_sync-property'} (boolean). Stop automatically syncing user roles.
- [`team_ids`](#spec.userConfig.auth_github.team_ids-property){: name='spec.userConfig.auth_github.team_ids-property'} (array of integers, MaxItems: 50). Require users to belong to one of given team IDs.

### auth_gitlab {: #spec.userConfig.auth_gitlab }

_Appears on [`spec.userConfig`](#spec.userConfig)._

GitLab Auth integration.

**Required**

- [`allowed_groups`](#spec.userConfig.auth_gitlab.allowed_groups-property){: name='spec.userConfig.auth_gitlab.allowed_groups-property'} (array of strings, MaxItems: 50). Require users to belong to one of given groups.
- [`client_id`](#spec.userConfig.auth_gitlab.client_id-property){: name='spec.userConfig.auth_gitlab.client_id-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider.
- [`client_secret`](#spec.userConfig.auth_gitlab.client_secret-property){: name='spec.userConfig.auth_gitlab.client_secret-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider.

**Optional**

- [`allow_sign_up`](#spec.userConfig.auth_gitlab.allow_sign_up-property){: name='spec.userConfig.auth_gitlab.allow_sign_up-property'} (boolean). Automatically sign-up users on successful sign-in.
- [`api_url`](#spec.userConfig.auth_gitlab.api_url-property){: name='spec.userConfig.auth_gitlab.api_url-property'} (string, MaxLength: 2048). This only needs to be set when using self hosted GitLab.
- [`auth_url`](#spec.userConfig.auth_gitlab.auth_url-property){: name='spec.userConfig.auth_gitlab.auth_url-property'} (string, MaxLength: 2048). This only needs to be set when using self hosted GitLab.
- [`token_url`](#spec.userConfig.auth_gitlab.token_url-property){: name='spec.userConfig.auth_gitlab.token_url-property'} (string, MaxLength: 2048). This only needs to be set when using self hosted GitLab.

### auth_google {: #spec.userConfig.auth_google }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Google Auth integration.

**Required**

- [`allowed_domains`](#spec.userConfig.auth_google.allowed_domains-property){: name='spec.userConfig.auth_google.allowed_domains-property'} (array of strings, MaxItems: 64). Domains allowed to sign-in to this Grafana.
- [`client_id`](#spec.userConfig.auth_google.client_id-property){: name='spec.userConfig.auth_google.client_id-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider.
- [`client_secret`](#spec.userConfig.auth_google.client_secret-property){: name='spec.userConfig.auth_google.client_secret-property'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider.

**Optional**

- [`allow_sign_up`](#spec.userConfig.auth_google.allow_sign_up-property){: name='spec.userConfig.auth_google.allow_sign_up-property'} (boolean). Automatically sign-up users on successful sign-in.

### date_formats {: #spec.userConfig.date_formats }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Grafana date format specifications.

**Optional**

- [`default_timezone`](#spec.userConfig.date_formats.default_timezone-property){: name='spec.userConfig.date_formats.default_timezone-property'} (string, MaxLength: 64). Default time zone for user preferences. Value `browser` uses browser local time zone.
- [`full_date`](#spec.userConfig.date_formats.full_date-property){: name='spec.userConfig.date_formats.full_date-property'} (string, MaxLength: 128). Moment.js style format string for cases where full date is shown.
- [`interval_day`](#spec.userConfig.date_formats.interval_day-property){: name='spec.userConfig.date_formats.interval_day-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring day accuracy is shown.
- [`interval_hour`](#spec.userConfig.date_formats.interval_hour-property){: name='spec.userConfig.date_formats.interval_hour-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring hour accuracy is shown.
- [`interval_minute`](#spec.userConfig.date_formats.interval_minute-property){: name='spec.userConfig.date_formats.interval_minute-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring minute accuracy is shown.
- [`interval_month`](#spec.userConfig.date_formats.interval_month-property){: name='spec.userConfig.date_formats.interval_month-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring month accuracy is shown.
- [`interval_second`](#spec.userConfig.date_formats.interval_second-property){: name='spec.userConfig.date_formats.interval_second-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring second accuracy is shown.
- [`interval_year`](#spec.userConfig.date_formats.interval_year-property){: name='spec.userConfig.date_formats.interval_year-property'} (string, MaxLength: 128). Moment.js style format string used when a time requiring year accuracy is shown.

### external_image_storage {: #spec.userConfig.external_image_storage }

_Appears on [`spec.userConfig`](#spec.userConfig)._

External image store settings.

**Required**

- [`access_key`](#spec.userConfig.external_image_storage.access_key-property){: name='spec.userConfig.external_image_storage.access_key-property'} (string, Pattern: `^[A-Z0-9]+$`, MaxLength: 4096). S3 access key. Requires permissions to the S3 bucket for the s3:PutObject and s3:PutObjectAcl actions.
- [`bucket_url`](#spec.userConfig.external_image_storage.bucket_url-property){: name='spec.userConfig.external_image_storage.bucket_url-property'} (string, MaxLength: 2048). Bucket URL for S3.
- [`provider`](#spec.userConfig.external_image_storage.provider-property){: name='spec.userConfig.external_image_storage.provider-property'} (string, Enum: `s3`). External image store provider.
- [`secret_key`](#spec.userConfig.external_image_storage.secret_key-property){: name='spec.userConfig.external_image_storage.secret_key-property'} (string, Pattern: `^[A-Za-z0-9/+=]+$`, MaxLength: 4096). S3 secret key.

### ip_filter {: #spec.userConfig.ip_filter }

_Appears on [`spec.userConfig`](#spec.userConfig)._

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#spec.userConfig.ip_filter.network-property){: name='spec.userConfig.ip_filter.network-property'} (string, MaxLength: 43). CIDR address block.

**Optional**

- [`description`](#spec.userConfig.ip_filter.description-property){: name='spec.userConfig.ip_filter.description-property'} (string, MaxLength: 1024). Description for IP filter list entry.

### private_access {: #spec.userConfig.private_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from private networks.

**Required**

- [`grafana`](#spec.userConfig.private_access.grafana-property){: name='spec.userConfig.private_access.grafana-property'} (boolean). Allow clients to connect to grafana with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations.

### privatelink_access {: #spec.userConfig.privatelink_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service components through Privatelink.

**Required**

- [`grafana`](#spec.userConfig.privatelink_access.grafana-property){: name='spec.userConfig.privatelink_access.grafana-property'} (boolean). Enable grafana.

### public_access {: #spec.userConfig.public_access }

_Appears on [`spec.userConfig`](#spec.userConfig)._

Allow access to selected service ports from the public Internet.

**Required**

- [`grafana`](#spec.userConfig.public_access.grafana-property){: name='spec.userConfig.public_access.grafana-property'} (boolean). Allow clients to connect to grafana from the public internet for service nodes that are in a project VPC or another type of private network.

### smtp_server {: #spec.userConfig.smtp_server }

_Appears on [`spec.userConfig`](#spec.userConfig)._

SMTP server settings.

**Required**

- [`from_address`](#spec.userConfig.smtp_server.from_address-property){: name='spec.userConfig.smtp_server.from_address-property'} (string, MaxLength: 319). Address used for sending emails.
- [`host`](#spec.userConfig.smtp_server.host-property){: name='spec.userConfig.smtp_server.host-property'} (string, MaxLength: 255). Server hostname or IP.
- [`port`](#spec.userConfig.smtp_server.port-property){: name='spec.userConfig.smtp_server.port-property'} (integer, Minimum: 1, Maximum: 65535). SMTP server port.

**Optional**

- [`from_name`](#spec.userConfig.smtp_server.from_name-property){: name='spec.userConfig.smtp_server.from_name-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 128). Name used in outgoing emails, defaults to Grafana.
- [`password`](#spec.userConfig.smtp_server.password-property){: name='spec.userConfig.smtp_server.password-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 255). Password for SMTP authentication.
- [`skip_verify`](#spec.userConfig.smtp_server.skip_verify-property){: name='spec.userConfig.smtp_server.skip_verify-property'} (boolean). Skip verifying server certificate. Defaults to false.
- [`starttls_policy`](#spec.userConfig.smtp_server.starttls_policy-property){: name='spec.userConfig.smtp_server.starttls_policy-property'} (string, Enum: `MandatoryStartTLS`, `NoStartTLS`, `OpportunisticStartTLS`). Either OpportunisticStartTLS, MandatoryStartTLS or NoStartTLS. Default is OpportunisticStartTLS.
- [`username`](#spec.userConfig.smtp_server.username-property){: name='spec.userConfig.smtp_server.username-property'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 255). Username for SMTP authentication.
