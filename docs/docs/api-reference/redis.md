---
title: "Redis"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Redis |

RedisSpec defines the desired state of Redis.

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
- [`userConfig`](#userConfig){: name='userConfig'} (object). Redis specific user configuration options. See [below for nested schema](#userConfig).

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

Redis specific user configuration options.

**Optional**

- [`additional_backup_regions`](#additional_backup_regions){: name='additional_backup_regions'} (array, MaxItems: 1). Additional Cloud Regions for Backup Replication. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`migration`](#migration){: name='migration'} (object). Migrate data from existing server. See [below for nested schema](#migration).
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`privatelink_access`](#privatelink_access){: name='privatelink_access'} (object). Allow access to selected service components through Privatelink. See [below for nested schema](#privatelink_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`recovery_basebackup_name`](#recovery_basebackup_name){: name='recovery_basebackup_name'} (string, Pattern: `^[a-zA-Z0-9-_:.]+$`, MaxLength: 128). Name of the basebackup to restore in forked service. 
- [`redis_acl_channels_default`](#redis_acl_channels_default){: name='redis_acl_channels_default'} (string, Enum: `allchannels`, `resetchannels`). Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default. 
- [`redis_io_threads`](#redis_io_threads){: name='redis_io_threads'} (integer, Minimum: 1, Maximum: 32). Redis IO thread count. 
- [`redis_lfu_decay_time`](#redis_lfu_decay_time){: name='redis_lfu_decay_time'} (integer, Minimum: 1, Maximum: 120). LFU maxmemory-policy counter decay time in minutes. 
- [`redis_lfu_log_factor`](#redis_lfu_log_factor){: name='redis_lfu_log_factor'} (integer, Minimum: 0, Maximum: 100). Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies. 
- [`redis_maxmemory_policy`](#redis_maxmemory_policy){: name='redis_maxmemory_policy'} (string, Enum: `noeviction`, `allkeys-lru`, `volatile-lru`, `allkeys-random`, `volatile-random`, `volatile-ttl`, `volatile-lfu`, `allkeys-lfu`). Redis maxmemory-policy. 
- [`redis_notify_keyspace_events`](#redis_notify_keyspace_events){: name='redis_notify_keyspace_events'} (string, Pattern: `^[KEg\$lshzxeA]*$`, MaxLength: 32). Set notify-keyspace-events option. 
- [`redis_number_of_databases`](#redis_number_of_databases){: name='redis_number_of_databases'} (integer, Minimum: 1, Maximum: 128). Set number of redis databases. Changing this will cause a restart of redis service. 
- [`redis_persistence`](#redis_persistence){: name='redis_persistence'} (string, Enum: `off`, `rdb`). When persistence is 'rdb', Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to backup schedule for backup purposes. When persistence is 'off', no RDB dumps and backups are done, so data can be lost at any moment if service is restarted for any reason, or if service is powered off. Also service can't be forked. 
- [`redis_pubsub_client_output_buffer_limit`](#redis_pubsub_client_output_buffer_limit){: name='redis_pubsub_client_output_buffer_limit'} (integer, Minimum: 32, Maximum: 512). Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan. 
- [`redis_ssl`](#redis_ssl){: name='redis_ssl'} (boolean). Require SSL to access Redis. 
- [`redis_timeout`](#redis_timeout){: name='redis_timeout'} (integer, Minimum: 0, Maximum: 31536000). Redis idle connection timeout in seconds. 
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### migration {: #migration }

Migrate data from existing server.

**Required**

- [`host`](#host){: name='host'} (string, MaxLength: 255). Hostname or IP address of the server where to migrate data from. 
- [`port`](#port){: name='port'} (integer, Minimum: 1, Maximum: 65535). Port number of the server where to migrate data from. 

**Optional**

- [`dbname`](#dbname){: name='dbname'} (string, MaxLength: 63). Database name for bootstrapping the initial connection. 
- [`ignore_dbs`](#ignore_dbs){: name='ignore_dbs'} (string, MaxLength: 2048). Comma-separated list of databases, which should be ignored during migration (supported by MySQL only at the moment). 
- [`method`](#method){: name='method'} (string, Enum: `dump`, `replication`). The migration method to be used (currently supported only by Redis and MySQL service types). 
- [`password`](#password){: name='password'} (string, MaxLength: 256). Password for authentication with the server where to migrate data from. 
- [`ssl`](#ssl){: name='ssl'} (boolean). The server where to migrate data from is secured with SSL. 
- [`username`](#username){: name='username'} (string, MaxLength: 256). User name for authentication with the server where to migrate data from. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Optional**

- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 
- [`redis`](#redis){: name='redis'} (boolean). Allow clients to connect to redis with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### privatelink_access {: #privatelink_access }

Allow access to selected service components through Privatelink.

**Optional**

- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Enable prometheus. 
- [`redis`](#redis){: name='redis'} (boolean). Enable redis. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Optional**

- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 
- [`redis`](#redis){: name='redis'} (boolean). Allow clients to connect to redis from the public internet for service nodes that are in a project VPC or another type of private network. 

