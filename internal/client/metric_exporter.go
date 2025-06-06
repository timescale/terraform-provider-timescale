package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
)

type MetricExporter struct {
	ID           string `json:"id"`
	ExporterUUID string `json:"exporterUuid"`
	Name         string `json:"name"`
	Created      string `json:"created"`
	Type         string `json:"type"`
	RegionCode   string `json:"regionCode"`

	// These will be populated by the custom UnmarshalJSON method because the API uses always the same 'config' key.
	Datadog    *DatadogMetricConfig
	Prometheus *PrometheusMetricConfig
	Cloudwatch *CloudwatchMetricConfig
}

// UnmarshalJSON provides custom logic for parsing the polymorphic 'config' object.
func (m *MetricExporter) UnmarshalJSON(data []byte) error {
	var temp struct {
		ID           string          `json:"id"`
		ExporterUUID string          `json:"exporterUuid"`
		Name         string          `json:"name"`
		Created      string          `json:"created"`
		Type         string          `json:"type"`
		RegionCode   string          `json:"regionCode"`
		Config       json.RawMessage `json:"config"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("error in initial unmarshal: %w", err)
	}

	// Copy all the common fields from the temporary struct
	m.ID = temp.ID
	m.ExporterUUID = temp.ExporterUUID
	m.Name = temp.Name
	m.Created = temp.Created
	m.Type = temp.Type
	m.RegionCode = temp.RegionCode

	// Parse config depending on Type
	switch strings.ToUpper(temp.Type) {
	case "DATADOG":
		var ddConfig DatadogMetricConfig
		if err := json.Unmarshal(temp.Config, &ddConfig); err != nil {
			return fmt.Errorf("error unmarshaling datadog config: %w", err)
		}
		m.Datadog = &ddConfig

	case "PROMETHEUS":
		var promConfig PrometheusMetricConfig
		if err := json.Unmarshal(temp.Config, &promConfig); err != nil {
			return fmt.Errorf("error unmarshaling prometheus config: %w", err)
		}
		m.Prometheus = &promConfig

	case "CLOUDWATCH":
		var cwConfig CloudwatchMetricConfig
		if err := json.Unmarshal(temp.Config, &cwConfig); err != nil {
			return fmt.Errorf("error unmarshaling cloudwatch config: %w", err)
		}
		m.Cloudwatch = &cwConfig
	}

	return nil
}

// DatadogMetricConfig holds the specific configuration for a Datadog exporter.
type DatadogMetricConfig struct {
	APIKey string `json:"apiKey"`
	Site   string `json:"site"`
}

// PrometheusMetricConfig holds the specific configuration for a Prometheus exporter.
type PrometheusMetricConfig struct {
	Username string `json:"user"`
	Password string `json:"password"`
}

// CloudwatchMetricConfig holds the specific configuration for an AWS CloudWatch exporter.
type CloudwatchMetricConfig struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
	Namespace     string `json:"namespace"`
	Region        string `json:"awsRegion"`
	RoleARN       string `json:"awsRoleArn,omitempty"`
	AccessKey     string `json:"awsAccessKey,omitempty"`
	SecretKey     string `json:"awsSecretKey,omitempty"`
}

// MetricExporterConfig is a container for any type of exporter configuration.
type MetricExporterConfig struct {
	Datadog    *DatadogMetricConfig
	Prometheus *PrometheusMetricConfig
	Cloudwatch *CloudwatchMetricConfig
}

type CreateMetricExporterResponse struct {
	MetricExporter *MetricExporter `json:"createMetricExporter"`
}
type GetAllMetricExportersResponse struct {
	DatadogMetricExporters []*MetricExporter `json:"getAllMetricExporters"`
}

func (c *Client) CreateMetricExporter(ctx context.Context, name, region string, config MetricExporterConfig) (*MetricExporter, error) {
	tflog.Trace(ctx, "Client.CreateMetricExporter")

	// Dynamically build the config
	var exporterConfig map[string]interface{}
	if config.Datadog != nil {
		exporterConfig = map[string]interface{}{"configDatadog": config.Datadog}
	} else if config.Prometheus != nil {
		exporterConfig = map[string]interface{}{"configPrometheus": config.Prometheus}
	} else if config.Cloudwatch != nil {
		exporterConfig = map[string]interface{}{"configCloudWatch": config.Cloudwatch}
	} else {
		return nil, errors.New("exporter config cannot be empty")
	}

	req := map[string]interface{}{
		"operationName": "CreateMetricExporter",
		"query":         CreateMetricExporterMutation,
		"variables": map[string]interface{}{
			"projectId":  c.projectID,
			"name":       name,
			"regionCode": region,
			"config":     exporterConfig,
		},
	}

	var resp Response[CreateMetricExporterResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.MetricExporter == nil {
		return nil, errors.New("API response did not contain exporter data")
	}

	return resp.Data.MetricExporter, nil
}

func (c *Client) GetAllMetricExporters(ctx context.Context) ([]*MetricExporter, error) {
	tflog.Trace(ctx, "Client.GetAllMetricExporters")
	req := map[string]interface{}{
		"operationName": "GetAllMetricExporters",
		"query":         GetAllMetricExportersQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[GetAllMetricExportersResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("error in the response: %w", resp.Errors[0])
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.DatadogMetricExporters, nil
}

func (c *Client) DeleteMetricExporter(ctx context.Context, uuid string) error {
	tflog.Trace(ctx, "Client.DeleteMetricExporter")

	req := map[string]interface{}{
		"operationName": "DeleteMetricExporter",
		"query":         DeleteMetricExporterMutation,
		"variables": map[string]any{
			"projectId":    c.projectID,
			"exporterUuid": uuid,
		},
	}
	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return fmt.Errorf("error doing the request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("error in the response: %w", resp.Errors[0])
	}
	return nil
}

func (c *Client) UpdateMetricExporter(ctx context.Context, uuid, name string, config MetricExporterConfig) error {
	tflog.Trace(ctx, "Client.UpdateMetricExporter")

	// Dynamically build the config
	var exporterConfig map[string]interface{}
	if config.Datadog != nil {
		exporterConfig = map[string]interface{}{"configDatadog": config.Datadog}
	} else if config.Prometheus != nil {
		exporterConfig = map[string]interface{}{"configPrometheus": config.Prometheus}
	} else if config.Cloudwatch != nil {
		exporterConfig = map[string]interface{}{"configCloudWatch": config.Cloudwatch}
	} else {
		return errors.New("exporter config cannot be empty for an update")
	}

	req := map[string]interface{}{
		"operationName": "UpdateMetricExporter",
		"query":         UpdateMetricExporterMutation,
		"variables": map[string]interface{}{
			"projectId":    c.projectID,
			"exporterUuid": uuid,
			"name":         name,
			"config":       exporterConfig,
		},
	}

	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}

	return nil
}
