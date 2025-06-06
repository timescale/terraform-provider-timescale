package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MetricExporter struct {
	ID           string `json:"id"`
	ExporterUUID string `json:"exporterUuid"`
	Name         string `json:"name"`
	Created      string `json:"created"`
	Type         string `json:"type"`

	Datadog    *DatadogConfig    `json:"datadogConfig,omitempty"`
	Prometheus *PrometheusConfig `json:"prometheusConfig,omitempty"`
	Cloudwatch *CloudwatchConfig `json:"cloudwatchConfig,omitempty"`
}

// DatadogConfig holds the specific configuration for a Datadog exporter.
type DatadogConfig struct {
	APIKey string `json:"apiKey"`
	Site   string `json:"site"`
}

// PrometheusConfig holds the specific configuration for a Prometheus exporter.
type PrometheusConfig struct {
	Username string `json:"user"`
	Password string `json:"password"`
}

// CloudwatchConfig holds the specific configuration for an AWS CloudWatch exporter.
type CloudwatchConfig struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
	Namespace     string `json:"namespace"`
	Region        string `json:"awsRegion"`
	RoleARN       string `json:"awsRoleArn,omitempty"`
	AccessKey     string `json:"awsAccessKey,omitempty"`
	SecretKey     string `json:"awsSecretKey,omitempty"`
}

// ExporterConfig is a container for any type of exporter configuration.
type ExporterConfig struct {
	Datadog    *DatadogConfig
	Prometheus *PrometheusConfig
	Cloudwatch *CloudwatchConfig
}

type CreateMetricExporterResponse struct {
	MetricExporter *MetricExporter `json:"createMetricExporter"`
}
type GetAllMetricExportersResponse struct {
	DatadogMetricExporters []*MetricExporter `json:"getAllMetricExporters"`
}

func (c *Client) CreateMetricExporter(ctx context.Context, name, region string, config ExporterConfig) (*MetricExporter, error) {
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

func (c *Client) UpdateMetricExporter(ctx context.Context, uuid, name string, config ExporterConfig) error {
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
