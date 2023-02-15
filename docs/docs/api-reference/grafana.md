---
title: "Grafana"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Grafana |

GrafanaSpec defines the desired state of Grafana.

- [`authSecretRef`](#authSecretRef){: name='authSecretRef'} (object). Authentication reference to Aiven token in a secret. See [below for nested schema](#authSecretRef).
- [`cloudName`](#cloudName){: name='cloudName'} (string, MaxLength: 256). Cloud the service runs in. 
- [`connInfoSecretTarget`](#connInfoSecretTarget){: name='connInfoSecretTarget'} (object). Information regarding secret creation. See [below for nested schema](#connInfoSecretTarget).
- [`disk_space`](#disk_space){: name='disk_space'} (string). The disk space of the service, possible values depend on the service type, the cloud provider and the project. Reducing will result in the service re-balancing. 
- [`maintenanceWindowDow`](#maintenanceWindowDow){: name='maintenanceWindowDow'} (string, Enum: `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`). Day of week when maintenance operations should be performed. One monday, tuesday, wednesday, etc. 
- [`maintenanceWindowTime`](#maintenanceWindowTime){: name='maintenanceWindowTime'} (string, MaxLength: 8). Time of day when maintenance operations should be performed. UTC time in HH:mm:ss format. 
- [`plan`](#plan){: name='plan'} (string, MaxLength: 128). Subscription plan. 
- [`project`](#project){: name='project'} (string, Immutable, MaxLength: 63). Target project. 
- [`projectVPCRef`](#projectVPCRef){: name='projectVPCRef'} (object, Immutable). ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically. See [below for nested schema](#projectVPCRef).
- [`projectVpcId`](#projectVpcId){: name='projectVpcId'} (string, Immutable, MaxLength: 36). Identifier of the VPC the service should be in, if any. 
- [`serviceIntegrations`](#serviceIntegrations){: name='serviceIntegrations'} (array, Immutable, MaxItems: 1). Service integrations to specify when creating a service. Not applied after initial service creation. See [below for nested schema](#serviceIntegrations).
- [`tags`](#tags){: name='tags'} (object). Tags are key-value pairs that allow you to categorize services. 
- [`terminationProtection`](#terminationProtection){: name='terminationProtection'} (boolean). Prevent service from being deleted. It is recommended to have this enabled for all services. 
- [`userConfig`](#userConfig){: name='userConfig'} (object). Cassandra specific user configuration options. See [below for nested schema](#userConfig).

## authSecretRef {: #authSecretRef }

Authentication reference to Aiven token in a secret.

**Optional**

- [`key`](#key){: name='key'} (string, MinLength: 1).  
- [`name`](#name){: name='name'} (string, MinLength: 1).  

## connInfoSecretTarget {: #connInfoSecretTarget }

Information regarding secret creation.

**Required**

- [`name`](#name){: name='name'} (string). Name of the Secret resource to be created. 

## projectVPCRef {: #projectVPCRef }

ProjectVPCRef reference to ProjectVPC resource to use its ID as ProjectVPCID automatically.

**Required**

- [`name`](#name){: name='name'} (string, MinLength: 1).  

**Optional**

- [`namespace`](#namespace){: name='namespace'} (string, MinLength: 1).  

## serviceIntegrations {: #serviceIntegrations }

Service integrations to specify when creating a service. Not applied after initial service creation.

**Required**

- [`integrationType`](#integrationType){: name='integrationType'} (string, Enum: `read_replica`).  
- [`sourceServiceName`](#sourceServiceName){: name='sourceServiceName'} (string, MinLength: 1, MaxLength: 64).  

## userConfig {: #userConfig }

Cassandra specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`alerting_enabled`](#alerting_enabled){: name='alerting_enabled'} (boolean). Enable or disable Grafana alerting functionality. 
- [`alerting_error_or_timeout`](#alerting_error_or_timeout){: name='alerting_error_or_timeout'} (string, Enum: `alerting`, `keep_state`). Default error or timeout setting for new alerting rules. 
- [`alerting_max_annotations_to_keep`](#alerting_max_annotations_to_keep){: name='alerting_max_annotations_to_keep'} (integer, Minimum: 0, Maximum: 1000000). Max number of alert annotations that Grafana stores. 0 (default) keeps all alert annotations. 
- [`alerting_nodata_or_nullvalues`](#alerting_nodata_or_nullvalues){: name='alerting_nodata_or_nullvalues'} (string, Enum: `alerting`, `no_data`, `keep_state`, `ok`). Default value for 'no data or null values' for new alerting rules. 
- [`allow_embedding`](#allow_embedding){: name='allow_embedding'} (boolean). Allow embedding Grafana dashboards with iframe/frame/object/embed tags. Disabled by default to limit impact of clickjacking. 
- [`auth_azuread`](#auth_azuread){: name='auth_azuread'} (object). Azure AD OAuth integration. See [below for nested schema](#auth_azuread).
- [`auth_basic_enabled`](#auth_basic_enabled){: name='auth_basic_enabled'} (boolean). Enable or disable basic authentication form, used by Grafana built-in login. 
- [`auth_generic_oauth`](#auth_generic_oauth){: name='auth_generic_oauth'} (object). Generic OAuth integration. See [below for nested schema](#auth_generic_oauth).
- [`auth_github`](#auth_github){: name='auth_github'} (object). Github Auth integration. See [below for nested schema](#auth_github).
- [`auth_gitlab`](#auth_gitlab){: name='auth_gitlab'} (object). GitLab Auth integration. See [below for nested schema](#auth_gitlab).
- [`auth_google`](#auth_google){: name='auth_google'} (object). Google Auth integration. See [below for nested schema](#auth_google).
- [`cookie_samesite`](#cookie_samesite){: name='cookie_samesite'} (string, Enum: `lax`, `strict`, `none`). Cookie SameSite attribute: 'strict' prevents sending cookie for cross-site requests, effectively disabling direct linking from other sites to Grafana. 'lax' is the default value. 
- [`custom_domain`](#custom_domain){: name='custom_domain'} (string, MaxLength: 255). Serve the web frontend using a custom CNAME pointing to the Aiven DNS name. 
- [`dashboard_previews_enabled`](#dashboard_previews_enabled){: name='dashboard_previews_enabled'} (boolean). This feature is new in Grafana 9 and is quite resource intensive. It may cause low-end plans to work more slowly while the dashboard previews are rendering. 
- [`dashboards_min_refresh_interval`](#dashboards_min_refresh_interval){: name='dashboards_min_refresh_interval'} (string, Pattern: `^[0-9]+(ms|s|m|h|d)$`, MaxLength: 16). Signed sequence of decimal numbers, followed by a unit suffix (ms, s, m, h, d), e.g. 30s, 1h. 
- [`dashboards_versions_to_keep`](#dashboards_versions_to_keep){: name='dashboards_versions_to_keep'} (integer, Minimum: 1, Maximum: 100). Dashboard versions to keep per dashboard. 
- [`dataproxy_send_user_header`](#dataproxy_send_user_header){: name='dataproxy_send_user_header'} (boolean). Send 'X-Grafana-User' header to data source. 
- [`dataproxy_timeout`](#dataproxy_timeout){: name='dataproxy_timeout'} (integer, Minimum: 15, Maximum: 90). Timeout for data proxy requests in seconds. 
- [`date_formats`](#date_formats){: name='date_formats'} (object). Grafana date format specifications. See [below for nested schema](#date_formats).
- [`disable_gravatar`](#disable_gravatar){: name='disable_gravatar'} (boolean). Set to true to disable gravatar. Defaults to false (gravatar is enabled). 
- [`editors_can_admin`](#editors_can_admin){: name='editors_can_admin'} (boolean). Editors can manage folders, teams and dashboards created by them. 
- [`external_image_storage`](#external_image_storage){: name='external_image_storage'} (object). External image store settings. See [below for nested schema](#external_image_storage).
- [`google_analytics_ua_id`](#google_analytics_ua_id){: name='google_analytics_ua_id'} (string, Pattern: `^(G|UA|YT|MO)-[a-zA-Z0-9-]+$`, MaxLength: 64). Google Analytics ID. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`metrics_enabled`](#metrics_enabled){: name='metrics_enabled'} (boolean). Enable Grafana /metrics endpoint. 
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`recovery_basebackup_name`](#recovery_basebackup_name){: name='recovery_basebackup_name'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service. 
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`smtp_server`](#smtp_server){: name='smtp_server'} (object). SMTP server settings. See [below for nested schema](#smtp_server).
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 
- [`user_auto_assign_org`](#user_auto_assign_org){: name='user_auto_assign_org'} (boolean). Auto-assign new users on signup to main organization. Defaults to false. 
- [`user_auto_assign_org_role`](#user_auto_assign_org_role){: name='user_auto_assign_org_role'} (string, Enum: `Viewer`, `Admin`, `Editor`). Set role for new signups. Defaults to Viewer. 
- [`viewers_can_edit`](#viewers_can_edit){: name='viewers_can_edit'} (boolean). Users with view-only permission can edit but not save dashboards. 

### auth_azuread {: #auth_azuread }

Azure AD OAuth integration.

**Required**

- [`auth_url`](#auth_url){: name='auth_url'} (string, MaxLength: 2048). Authorization URL. 
- [`client_id`](#client_id){: name='client_id'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider. 
- [`client_secret`](#client_secret){: name='client_secret'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider. 
- [`token_url`](#token_url){: name='token_url'} (string, MaxLength: 2048). Token URL. 

**Optional**

- [`allow_sign_up`](#allow_sign_up){: name='allow_sign_up'} (boolean). Automatically sign-up users on successful sign-in. 
- [`allowed_domains`](#allowed_domains){: name='allowed_domains'} (array, MaxItems: 50). Allowed domains. 
- [`allowed_groups`](#allowed_groups){: name='allowed_groups'} (array, MaxItems: 50). Require users to belong to one of given groups. 

### auth_generic_oauth {: #auth_generic_oauth }

Generic OAuth integration.

**Required**

- [`api_url`](#api_url){: name='api_url'} (string, MaxLength: 2048). API URL. 
- [`auth_url`](#auth_url){: name='auth_url'} (string, MaxLength: 2048). Authorization URL. 
- [`client_id`](#client_id){: name='client_id'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider. 
- [`client_secret`](#client_secret){: name='client_secret'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider. 
- [`token_url`](#token_url){: name='token_url'} (string, MaxLength: 2048). Token URL. 

**Optional**

- [`allow_sign_up`](#allow_sign_up){: name='allow_sign_up'} (boolean). Automatically sign-up users on successful sign-in. 
- [`allowed_domains`](#allowed_domains){: name='allowed_domains'} (array, MaxItems: 50). Allowed domains. 
- [`allowed_organizations`](#allowed_organizations){: name='allowed_organizations'} (array, MaxItems: 50). Require user to be member of one of the listed organizations. 
- [`name`](#name){: name='name'} (string, Pattern: `^[a-zA-Z0-9_\- ]+$`, MaxLength: 128). Name of the OAuth integration. 
- [`scopes`](#scopes){: name='scopes'} (array, MaxItems: 50). OAuth scopes. 

### auth_github {: #auth_github }

Github Auth integration.

**Required**

- [`client_id`](#client_id){: name='client_id'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider. 
- [`client_secret`](#client_secret){: name='client_secret'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider. 

**Optional**

- [`allow_sign_up`](#allow_sign_up){: name='allow_sign_up'} (boolean). Automatically sign-up users on successful sign-in. 
- [`allowed_organizations`](#allowed_organizations){: name='allowed_organizations'} (array, MaxItems: 50). Require users to belong to one of given organizations. 
- [`team_ids`](#team_ids){: name='team_ids'} (array, MaxItems: 50). Require users to belong to one of given team IDs. 

### auth_gitlab {: #auth_gitlab }

GitLab Auth integration.

**Required**

- [`allowed_groups`](#allowed_groups){: name='allowed_groups'} (array, MaxItems: 50). Require users to belong to one of given groups. 
- [`client_id`](#client_id){: name='client_id'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider. 
- [`client_secret`](#client_secret){: name='client_secret'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider. 

**Optional**

- [`allow_sign_up`](#allow_sign_up){: name='allow_sign_up'} (boolean). Automatically sign-up users on successful sign-in. 
- [`api_url`](#api_url){: name='api_url'} (string, MaxLength: 2048). API URL. This only needs to be set when using self hosted GitLab. 
- [`auth_url`](#auth_url){: name='auth_url'} (string, MaxLength: 2048). Authorization URL. This only needs to be set when using self hosted GitLab. 
- [`token_url`](#token_url){: name='token_url'} (string, MaxLength: 2048). Token URL. This only needs to be set when using self hosted GitLab. 

### auth_google {: #auth_google }

Google Auth integration.

**Required**

- [`allowed_domains`](#allowed_domains){: name='allowed_domains'} (array, MaxItems: 64). Domains allowed to sign-in to this Grafana. 
- [`client_id`](#client_id){: name='client_id'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client ID from provider. 
- [`client_secret`](#client_secret){: name='client_secret'} (string, Pattern: `^[\040-\176]+$`, MaxLength: 1024). Client secret from provider. 

**Optional**

- [`allow_sign_up`](#allow_sign_up){: name='allow_sign_up'} (boolean). Automatically sign-up users on successful sign-in. 

### date_formats {: #date_formats }

Grafana date format specifications.

**Optional**

- [`default_timezone`](#default_timezone){: name='default_timezone'} (string, MaxLength: 64). Default time zone for user preferences. Value 'browser' uses browser local time zone. 
- [`full_date`](#full_date){: name='full_date'} (string, MaxLength: 128). Moment.js style format string for cases where full date is shown. 
- [`interval_day`](#interval_day){: name='interval_day'} (string, MaxLength: 128). Moment.js style format string used when a time requiring day accuracy is shown. 
- [`interval_hour`](#interval_hour){: name='interval_hour'} (string, MaxLength: 128). Moment.js style format string used when a time requiring hour accuracy is shown. 
- [`interval_minute`](#interval_minute){: name='interval_minute'} (string, MaxLength: 128). Moment.js style format string used when a time requiring minute accuracy is shown. 
- [`interval_month`](#interval_month){: name='interval_month'} (string, MaxLength: 128). Moment.js style format string used when a time requiring month accuracy is shown. 
- [`interval_second`](#interval_second){: name='interval_second'} (string, MaxLength: 128). Moment.js style format string used when a time requiring second accuracy is shown. 
- [`interval_year`](#interval_year){: name='interval_year'} (string, MaxLength: 128). Moment.js style format string used when a time requiring year accuracy is shown. 

### external_image_storage {: #external_image_storage }

External image store settings.

**Required**

- [`access_key`](#access_key){: name='access_key'} (string, Pattern: `^[A-Z0-9]+$`, MaxLength: 4096). S3 access key. Requires permissions to the S3 bucket for the s3:PutObject and s3:PutObjectAcl actions. 
- [`bucket_url`](#bucket_url){: name='bucket_url'} (string, MaxLength: 2048). Bucket URL for S3. 
- [`provider`](#provider){: name='provider'} (string, Enum: `s3`). Provider type. 
- [`secret_key`](#secret_key){: name='secret_key'} (string, Pattern: `^[A-Za-z0-9/+=]+$`, MaxLength: 4096). S3 secret key. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Required**

- [`grafana`](#grafana){: name='grafana'} (boolean). Allow clients to connect to grafana with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Required**

- [`grafana`](#grafana){: name='grafana'} (boolean). Enable grafana. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Required**

- [`grafana`](#grafana){: name='grafana'} (boolean). Allow clients to connect to grafana from the public internet for service nodes that are in a project VPC or another type of private network. 

### smtp_server {: #smtp_server }

SMTP server settings.

**Required**

- [`from_address`](#from_address){: name='from_address'} (string, MaxLength: 319). Address used for sending emails. 
- [`host`](#host){: name='host'} (string, MaxLength: 255). Server hostname or IP. 
- [`port`](#port){: name='port'} (integer, Minimum: 1, Maximum: 65535). SMTP server port. 

**Optional**

- [`from_name`](#from_name){: name='from_name'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 128). Name used in outgoing emails, defaults to Grafana. 
- [`password`](#password){: name='password'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 255). Password for SMTP authentication. 
- [`skip_verify`](#skip_verify){: name='skip_verify'} (boolean). Skip verifying server certificate. Defaults to false. 
- [`starttls_policy`](#starttls_policy){: name='starttls_policy'} (string, Enum: `OpportunisticStartTLS`, `MandatoryStartTLS`, `NoStartTLS`). Either OpportunisticStartTLS, MandatoryStartTLS or NoStartTLS. Default is OpportunisticStartTLS. 
- [`username`](#username){: name='username'} (string, Pattern: `^[^\x00-\x1F]+$`, MaxLength: 255). Username for SMTP authentication. 

