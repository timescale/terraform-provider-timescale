package client

type MetricExporter struct {
	ID                string `json:"id"`
	ProjectID         string `json:"projectId"`
	Name              string `json:"name"`
	AutoscaleSettings struct {
		Enabled bool `json:"enabled"`
	} `json:"autoscaleSettings"`
	Status        string         `json:"status"`
	RegionCode    string         `json:"regionCode"`
	Paused        bool           `json:"paused"`
	ServiceSpec   ServiceSpec    `json:"spec"`
	Resources     []ResourceSpec `json:"resources"`
	Created       string         `json:"created"`
	ReplicaStatus string         `json:"replicaStatus"`
	VPCEndpoint   *VPCEndpoint   `json:"vpcEndpoint"`
	ForkSpec      *ForkSpec      `json:"forkedFromId"`
}

type CreateMetricExporterRequest struct {
	Name       string
	ProjectID  string
	RegionCode string

	CloudWatch *CloudWatchMetricConfig
	Datadog    *DatadogMetricConfig
}

type CloudWatchMetricConfig struct {
	LogGroupName  string
	LogStreamName string
	Namespace     string
	AwsAccessKey  string
	AwsSecretKey  string
	AwsRegion     string
	AwsRoleArn    string
}

type DatadogMetricConfig struct {
	ApiKey string
	Site   string
}
