// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package clickhousekafkauserconfig

// Table column
type Columns struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	// Column name
	Name string `groups:"create,update" json:"name"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1000
	// Column type
	Type string `groups:"create,update" json:"type"`
}

// Kafka topic
type Topics struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=249
	// Name of the topic
	Name string `groups:"create,update" json:"name"`
}

// Table to create
type Tables struct {
	// +kubebuilder:validation:Enum="beginning";"earliest";"end";"largest";"latest";"smallest"
	// Action to take when there is no initial offset in offset store or the desired offset is out of range
	AutoOffsetReset *string `groups:"create,update" json:"auto_offset_reset,omitempty"`

	// +kubebuilder:validation:MaxItems=100
	// Table columns
	Columns []*Columns `groups:"create,update" json:"columns"`

	// +kubebuilder:validation:Enum="Avro";"AvroConfluent";"CSV";"JSONAsString";"JSONCompactEachRow";"JSONCompactStringsEachRow";"JSONEachRow";"JSONStringsEachRow";"MsgPack";"Parquet";"RawBLOB";"TSKV";"TSV";"TabSeparated"
	// Message data format
	DataFormat string `groups:"create,update" json:"data_format"`

	// +kubebuilder:validation:Enum="basic";"best_effort";"best_effort_us"
	// Method to read DateTime from text input formats
	DateTimeInputFormat *string `groups:"create,update" json:"date_time_input_format,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=249
	// Kafka consumers group
	GroupName string `groups:"create,update" json:"group_name"`

	// +kubebuilder:validation:Enum="default";"stream"
	// How to handle errors for Kafka engine
	HandleErrorMode *string `groups:"create,update" json:"handle_error_mode,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000000
	// Number of row collected by poll(s) for flushing data from Kafka
	MaxBlockSize *int `groups:"create,update" json:"max_block_size,omitempty"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000000000
	// The maximum number of rows produced in one kafka message for row-based formats
	MaxRowsPerMessage *int `groups:"create,update" json:"max_rows_per_message,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=40
	// Name of the table
	Name string `groups:"create,update" json:"name"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// The number of consumers per table per replica
	NumConsumers *int `groups:"create,update" json:"num_consumers,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000000
	// Maximum amount of messages to be polled in a single Kafka poll
	PollMaxBatchSize *int `groups:"create,update" json:"poll_max_batch_size,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=30000
	// Timeout in milliseconds for a single poll from Kafka. Takes the value of the stream_flush_interval_ms server setting by default (500ms).
	PollMaxTimeoutMs *int `groups:"create,update" json:"poll_max_timeout_ms,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000000000
	// Skip at least this number of broken messages from Kafka topic per block
	SkipBrokenMessages *int `groups:"create,update" json:"skip_broken_messages,omitempty"`

	// Provide an independent thread for each consumer. All consumers run in the same thread by default.
	ThreadPerConsumer *bool `groups:"create,update" json:"thread_per_consumer,omitempty"`

	// +kubebuilder:validation:MaxItems=100
	// Kafka topics
	Topics []*Topics `groups:"create,update" json:"topics"`
}

// Integration user config
type ClickhouseKafkaUserConfig struct {
	// +kubebuilder:validation:MaxItems=100
	// Tables to create
	Tables []*Tables `groups:"create,update" json:"tables,omitempty"`
}
