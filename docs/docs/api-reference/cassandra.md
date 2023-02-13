---
title: "Cassandra"
---

| ApiVersion                  | Kind        |
|-----------------------------|-------------|
| aiven.io/v1alpha1 | Cassandra |

CassandraSpec defines the desired state of Cassandra.

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
- [`cassandra`](#cassandra){: name='cassandra'} (object). cassandra configuration values. See [below for nested schema](#cassandra).
- [`cassandra_version`](#cassandra_version){: name='cassandra_version'} (string, Enum: `4`). Cassandra major version. 
- [`ip_filter`](#ip_filter){: name='ip_filter'} (array, MaxItems: 1024). Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'. See [below for nested schema](#ip_filter).
- [`migrate_sstableloader`](#migrate_sstableloader){: name='migrate_sstableloader'} (boolean). Sets the service into migration mode enabling the sstableloader utility to be used to upload Cassandra data files. Available only on service create. 
- [`private_access`](#private_access){: name='private_access'} (object). Allow access to selected service ports from private networks. See [below for nested schema](#private_access).
- [`project_to_fork_from`](#project_to_fork_from){: name='project_to_fork_from'} (string, Immutable, MaxLength: 63). Name of another project to fork a service from. This has effect only when a new service is being created. 
- [`public_access`](#public_access){: name='public_access'} (object). Allow access to selected service ports from the public Internet. See [below for nested schema](#public_access).
- [`service_to_fork_from`](#service_to_fork_from){: name='service_to_fork_from'} (string, Immutable, MaxLength: 64). Name of another service to fork from. This has effect only when a new service is being created. 
- [`service_to_join_with`](#service_to_join_with){: name='service_to_join_with'} (string, MaxLength: 64). When bootstrapping, instead of creating a new Cassandra cluster try to join an existing one from another service. Can only be set on service creation. 
- [`static_ips`](#static_ips){: name='static_ips'} (boolean). Use static public IP addresses. 

### cassandra {: #cassandra }

cassandra configuration values.

**Optional**

- [`batch_size_fail_threshold_in_kb`](#batch_size_fail_threshold_in_kb){: name='batch_size_fail_threshold_in_kb'} (integer, Minimum: 1, Maximum: 1000000). Fail any multiple-partition batch exceeding this value. 50kb (10x warn threshold) by default. 
- [`batch_size_warn_threshold_in_kb`](#batch_size_warn_threshold_in_kb){: name='batch_size_warn_threshold_in_kb'} (integer, Minimum: 1, Maximum: 1000000). Log a warning message on any multiple-partition batch size exceeding this value.5kb per batch by default.Caution should be taken on increasing the size of this thresholdas it can lead to node instability. 
- [`datacenter`](#datacenter){: name='datacenter'} (string, MaxLength: 128). Name of the datacenter to which nodes of this service belong. Can be set only when creating the service. 

### ip_filter {: #ip_filter }

CIDR address block, either as a string, or in a dict with an optional description field.

**Required**

- [`network`](#network){: name='network'} (string, MaxLength: 43). CIDR address block. 

**Optional**

- [`description`](#description){: name='description'} (string, MaxLength: 1024). Description for IP filter list entry. 

### private_access {: #private_access }

Allow access to selected service ports from private networks.

**Required**

- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations. 

### public_access {: #public_access }

Allow access to selected service ports from the public Internet.

**Required**

- [`prometheus`](#prometheus){: name='prometheus'} (boolean). Allow clients to connect to prometheus from the public internet for service nodes that are in a project VPC or another type of private network. 

