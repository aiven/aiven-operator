// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package grafanauserconfig

// Azure AD OAuth integration
type AuthAzuread struct {
	// Automatically sign-up users on successful sign-in
	AllowSignUp *bool `groups:"create,update" json:"allow_sign_up,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Allowed domains
	AllowedDomains []string `groups:"create,update" json:"allowed_domains,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Require users to belong to one of given groups
	AllowedGroups []string `groups:"create,update" json:"allowed_groups,omitempty"`

	// +kubebuilder:validation:MaxLength=2048
	// Authorization URL
	AuthUrl string `groups:"create,update" json:"auth_url"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client ID from provider
	ClientId string `groups:"create,update" json:"client_id"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client secret from provider
	ClientSecret string `groups:"create,update" json:"client_secret"`

	// +kubebuilder:validation:MaxLength=2048
	// Token URL
	TokenUrl string `groups:"create,update" json:"token_url"`
}

// Generic OAuth integration
type AuthGenericOauth struct {
	// Automatically sign-up users on successful sign-in
	AllowSignUp *bool `groups:"create,update" json:"allow_sign_up,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Allowed domains
	AllowedDomains []string `groups:"create,update" json:"allowed_domains,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Require user to be member of one of the listed organizations
	AllowedOrganizations []string `groups:"create,update" json:"allowed_organizations,omitempty"`

	// +kubebuilder:validation:MaxLength=2048
	// API URL
	ApiUrl string `groups:"create,update" json:"api_url"`

	// +kubebuilder:validation:MaxLength=2048
	// Authorization URL
	AuthUrl string `groups:"create,update" json:"auth_url"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client ID from provider
	ClientId string `groups:"create,update" json:"client_id"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client secret from provider
	ClientSecret string `groups:"create,update" json:"client_secret"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9_\- ]+$`
	// Name of the OAuth integration
	Name *string `groups:"create,update" json:"name,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// OAuth scopes
	Scopes []string `groups:"create,update" json:"scopes,omitempty"`

	// +kubebuilder:validation:MaxLength=2048
	// Token URL
	TokenUrl string `groups:"create,update" json:"token_url"`
}

// Github Auth integration
type AuthGithub struct {
	// Automatically sign-up users on successful sign-in
	AllowSignUp *bool `groups:"create,update" json:"allow_sign_up,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Require users to belong to one of given organizations
	AllowedOrganizations []string `groups:"create,update" json:"allowed_organizations,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client ID from provider
	ClientId string `groups:"create,update" json:"client_id"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client secret from provider
	ClientSecret string `groups:"create,update" json:"client_secret"`

	// +kubebuilder:validation:MaxItems=50
	// Require users to belong to one of given team IDs
	TeamIds []int `groups:"create,update" json:"team_ids,omitempty"`
}

// GitLab Auth integration
type AuthGitlab struct {
	// Automatically sign-up users on successful sign-in
	AllowSignUp *bool `groups:"create,update" json:"allow_sign_up,omitempty"`

	// +kubebuilder:validation:MaxItems=50
	// Require users to belong to one of given groups
	AllowedGroups []string `groups:"create,update" json:"allowed_groups"`

	// +kubebuilder:validation:MaxLength=2048
	// API URL. This only needs to be set when using self hosted GitLab
	ApiUrl *string `groups:"create,update" json:"api_url,omitempty"`

	// +kubebuilder:validation:MaxLength=2048
	// Authorization URL. This only needs to be set when using self hosted GitLab
	AuthUrl *string `groups:"create,update" json:"auth_url,omitempty"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client ID from provider
	ClientId string `groups:"create,update" json:"client_id"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client secret from provider
	ClientSecret string `groups:"create,update" json:"client_secret"`

	// +kubebuilder:validation:MaxLength=2048
	// Token URL. This only needs to be set when using self hosted GitLab
	TokenUrl *string `groups:"create,update" json:"token_url,omitempty"`
}

// Google Auth integration
type AuthGoogle struct {
	// Automatically sign-up users on successful sign-in
	AllowSignUp *bool `groups:"create,update" json:"allow_sign_up,omitempty"`

	// +kubebuilder:validation:MaxItems=64
	// Domains allowed to sign-in to this Grafana
	AllowedDomains []string `groups:"create,update" json:"allowed_domains"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client ID from provider
	ClientId string `groups:"create,update" json:"client_id"`

	// +kubebuilder:validation:MaxLength=1024
	// +kubebuilder:validation:Pattern=`^[\040-\176]+$`
	// Client secret from provider
	ClientSecret string `groups:"create,update" json:"client_secret"`
}

// Grafana date format specifications
type DateFormats struct {
	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`(?i)^([a-zA-Z_]+/){1,2}[a-zA-Z_-]+$|^(Etc/)?(UTC|GMT)([+-](\d){1,2})?$|^(Factory)$|^(browser)$`
	// Default time zone for user preferences. Value 'browser' uses browser local time zone.
	DefaultTimezone *string `groups:"create,update" json:"default_timezone,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string for cases where full date is shown
	FullDate *string `groups:"create,update" json:"full_date,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring day accuracy is shown
	IntervalDay *string `groups:"create,update" json:"interval_day,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring hour accuracy is shown
	IntervalHour *string `groups:"create,update" json:"interval_hour,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring minute accuracy is shown
	IntervalMinute *string `groups:"create,update" json:"interval_minute,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring month accuracy is shown
	IntervalMonth *string `groups:"create,update" json:"interval_month,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring second accuracy is shown
	IntervalSecond *string `groups:"create,update" json:"interval_second,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^(([Hh]mm(ss)?|Mo|MM?M?M?|Do|DDDo|DD?D?D?|ddd?d?|do?|w[o|w]?|W[o|W]?|Qo?|N{1,5}|YYYYYY|YYYYY|YYYY|YY|y{2,4}|yo?|gg(ggg?)?|GG(GGG?)?|e|E|a|A|hh?|HH?|kk?|mm?|ss?|S{1,9}|x|X|zz?|ZZ?|LTS|LT|LL?L?L?|l{1,4}|[-+/T,;.: ]?)*)$`
	// Moment.js style format string used when a time requiring year accuracy is shown
	IntervalYear *string `groups:"create,update" json:"interval_year,omitempty"`
}

// External image store settings
type ExternalImageStorage struct {
	// +kubebuilder:validation:MaxLength=4096
	// +kubebuilder:validation:Pattern=`^[A-Z0-9]+$`
	// S3 access key. Requires permissions to the S3 bucket for the s3:PutObject and s3:PutObjectAcl actions
	AccessKey string `groups:"create,update" json:"access_key"`

	// +kubebuilder:validation:MaxLength=2048
	// Bucket URL for S3
	BucketUrl string `groups:"create,update" json:"bucket_url"`

	// +kubebuilder:validation:Enum="s3"
	// Provider type
	Provider string `groups:"create,update" json:"provider"`

	// +kubebuilder:validation:MaxLength=4096
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9/+=]+$`
	// S3 secret key
	SecretKey string `groups:"create,update" json:"secret_key"`
}

// CIDR address block, either as a string, or in a dict with an optional description field
type IpFilter struct {
	// +kubebuilder:validation:MaxLength=1024
	// Description for IP filter list entry
	Description *string `groups:"create,update" json:"description,omitempty"`

	// +kubebuilder:validation:MaxLength=43
	// CIDR address block
	Network string `groups:"create,update" json:"network"`
}

// Allow access to selected service ports from private networks
type PrivateAccess struct {
	// Allow clients to connect to grafana with a DNS name that always resolves to the service's private IP addresses. Only available in certain network locations
	Grafana *bool `groups:"create,update" json:"grafana,omitempty"`
}

// Allow access to selected service components through Privatelink
type PrivatelinkAccess struct {
	// Enable grafana
	Grafana *bool `groups:"create,update" json:"grafana,omitempty"`
}

// Allow access to selected service ports from the public Internet
type PublicAccess struct {
	// Allow clients to connect to grafana from the public internet for service nodes that are in a project VPC or another type of private network
	Grafana *bool `groups:"create,update" json:"grafana,omitempty"`
}

// SMTP server settings
type SmtpServer struct {
	// +kubebuilder:validation:MaxLength=319
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9_\-\.+\'&]+@(([\da-zA-Z])([_\w-]{,62})\.){,127}(([\da-zA-Z])[_\w-]{,61})?([\da-zA-Z]\.((xn\-\-[a-zA-Z\d]+)|([a-zA-Z\d]{2,})))$`
	// Address used for sending emails
	FromAddress string `groups:"create,update" json:"from_address"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[^\x00-\x1F]+$`
	// Name used in outgoing emails, defaults to Grafana
	FromName *string `groups:"create,update" json:"from_name,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Server hostname or IP
	Host string `groups:"create,update" json:"host"`

	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[^\x00-\x1F]+$`
	// Password for SMTP authentication
	Password *string `groups:"create,update" json:"password,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// SMTP server port
	Port int `groups:"create,update" json:"port"`

	// Skip verifying server certificate. Defaults to false
	SkipVerify *bool `groups:"create,update" json:"skip_verify,omitempty"`

	// +kubebuilder:validation:Enum="OpportunisticStartTLS";"MandatoryStartTLS";"NoStartTLS"
	// Either OpportunisticStartTLS, MandatoryStartTLS or NoStartTLS. Default is OpportunisticStartTLS.
	StarttlsPolicy *string `groups:"create,update" json:"starttls_policy,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[^\x00-\x1F]+$`
	// Username for SMTP authentication
	Username *string `groups:"create,update" json:"username,omitempty"`
}
type GrafanaUserConfig struct {
	// +kubebuilder:validation:MaxItems=1
	// Additional Cloud Regions for Backup Replication
	AdditionalBackupRegions []string `groups:"create,update" json:"additional_backup_regions,omitempty"`

	// Enable or disable Grafana alerting functionality
	AlertingEnabled *bool `groups:"create,update" json:"alerting_enabled,omitempty"`

	// +kubebuilder:validation:Enum="alerting";"keep_state"
	// Default error or timeout setting for new alerting rules
	AlertingErrorOrTimeout *string `groups:"create,update" json:"alerting_error_or_timeout,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000
	// Max number of alert annotations that Grafana stores. 0 (default) keeps all alert annotations.
	AlertingMaxAnnotationsToKeep *int `groups:"create,update" json:"alerting_max_annotations_to_keep,omitempty"`

	// +kubebuilder:validation:Enum="alerting";"no_data";"keep_state";"ok"
	// Default value for 'no data or null values' for new alerting rules
	AlertingNodataOrNullvalues *string `groups:"create,update" json:"alerting_nodata_or_nullvalues,omitempty"`

	// Allow embedding Grafana dashboards with iframe/frame/object/embed tags. Disabled by default to limit impact of clickjacking
	AllowEmbedding *bool `groups:"create,update" json:"allow_embedding,omitempty"`

	// Azure AD OAuth integration
	AuthAzuread *AuthAzuread `groups:"create,update" json:"auth_azuread,omitempty"`

	// Enable or disable basic authentication form, used by Grafana built-in login
	AuthBasicEnabled *bool `groups:"create,update" json:"auth_basic_enabled,omitempty"`

	// Generic OAuth integration
	AuthGenericOauth *AuthGenericOauth `groups:"create,update" json:"auth_generic_oauth,omitempty"`

	// Github Auth integration
	AuthGithub *AuthGithub `groups:"create,update" json:"auth_github,omitempty"`

	// GitLab Auth integration
	AuthGitlab *AuthGitlab `groups:"create,update" json:"auth_gitlab,omitempty"`

	// Google Auth integration
	AuthGoogle *AuthGoogle `groups:"create,update" json:"auth_google,omitempty"`

	// +kubebuilder:validation:Enum="lax";"strict";"none"
	// Cookie SameSite attribute: 'strict' prevents sending cookie for cross-site requests, effectively disabling direct linking from other sites to Grafana. 'lax' is the default value.
	CookieSamesite *string `groups:"create,update" json:"cookie_samesite,omitempty"`

	// +kubebuilder:validation:MaxLength=255
	// Serve the web frontend using a custom CNAME pointing to the Aiven DNS name
	CustomDomain *string `groups:"create,update" json:"custom_domain,omitempty"`

	// This feature is new in Grafana 9 and is quite resource intensive. It may cause low-end plans to work more slowly while the dashboard previews are rendering.
	DashboardPreviewsEnabled *bool `groups:"create,update" json:"dashboard_previews_enabled,omitempty"`

	// +kubebuilder:validation:MaxLength=16
	// +kubebuilder:validation:Pattern=`^[0-9]+(ms|s|m|h|d)$`
	// Signed sequence of decimal numbers, followed by a unit suffix (ms, s, m, h, d), e.g. 30s, 1h
	DashboardsMinRefreshInterval *string `groups:"create,update" json:"dashboards_min_refresh_interval,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// Dashboard versions to keep per dashboard
	DashboardsVersionsToKeep *int `groups:"create,update" json:"dashboards_versions_to_keep,omitempty"`

	// Send 'X-Grafana-User' header to data source
	DataproxySendUserHeader *bool `groups:"create,update" json:"dataproxy_send_user_header,omitempty"`

	// +kubebuilder:validation:Minimum=15
	// +kubebuilder:validation:Maximum=90
	// Timeout for data proxy requests in seconds
	DataproxyTimeout *int `groups:"create,update" json:"dataproxy_timeout,omitempty"`

	// Grafana date format specifications
	DateFormats *DateFormats `groups:"create,update" json:"date_formats,omitempty"`

	// Set to true to disable gravatar. Defaults to false (gravatar is enabled)
	DisableGravatar *bool `groups:"create,update" json:"disable_gravatar,omitempty"`

	// Editors can manage folders, teams and dashboards created by them
	EditorsCanAdmin *bool `groups:"create,update" json:"editors_can_admin,omitempty"`

	// External image store settings
	ExternalImageStorage *ExternalImageStorage `groups:"create,update" json:"external_image_storage,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:Pattern=`^(G|UA|YT|MO)-[a-zA-Z0-9-]+$`
	// Google Analytics ID
	GoogleAnalyticsUaId *string `groups:"create,update" json:"google_analytics_ua_id,omitempty"`

	// +kubebuilder:validation:MaxItems=1024
	// Allow incoming connections from CIDR address block, e.g. '10.20.0.0/16'
	IpFilter []*IpFilter `groups:"create,update" json:"ip_filter,omitempty"`

	// Enable Grafana /metrics endpoint
	MetricsEnabled *bool `groups:"create,update" json:"metrics_enabled,omitempty"`

	// Allow access to selected service ports from private networks
	PrivateAccess *PrivateAccess `groups:"create,update" json:"private_access,omitempty"`

	// Allow access to selected service components through Privatelink
	PrivatelinkAccess *PrivatelinkAccess `groups:"create,update" json:"privatelink_access,omitempty"`

	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Name of another project to fork a service from. This has effect only when a new service is being created.
	ProjectToForkFrom *string `groups:"create" json:"project_to_fork_from,omitempty"`

	// Allow access to selected service ports from the public Internet
	PublicAccess *PublicAccess `groups:"create,update" json:"public_access,omitempty"`

	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9-_:.]+$`
	// Name of the basebackup to restore in forked service
	RecoveryBasebackupName *string `groups:"create,update" json:"recovery_basebackup_name,omitempty"`

	// +kubebuilder:validation:MaxLength=64
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	// Name of another service to fork from. This has effect only when a new service is being created.
	ServiceToForkFrom *string `groups:"create" json:"service_to_fork_from,omitempty"`

	// SMTP server settings
	SmtpServer *SmtpServer `groups:"create,update" json:"smtp_server,omitempty"`

	// Use static public IP addresses
	StaticIps *bool `groups:"create,update" json:"static_ips,omitempty"`

	// Auto-assign new users on signup to main organization. Defaults to false
	UserAutoAssignOrg *bool `groups:"create,update" json:"user_auto_assign_org,omitempty"`

	// +kubebuilder:validation:Enum="Viewer";"Admin";"Editor"
	// Set role for new signups. Defaults to Viewer
	UserAutoAssignOrgRole *string `groups:"create,update" json:"user_auto_assign_org_role,omitempty"`

	// Users with view-only permission can edit but not save dashboards
	ViewersCanEdit *bool `groups:"create,update" json:"viewers_can_edit,omitempty"`
}
