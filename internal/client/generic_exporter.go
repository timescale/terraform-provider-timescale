package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type GenericExporter struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Created    string `json:"created"`
	Type       string `json:"type"`
	RegionCode string `json:"regionCode"`

	Cloudwatch *CloudwatchGenericConfig `json:"cloudWatchConfig"`
}

// CloudwatchGenericConfig holds the specific configuration for an AWS CloudWatch exporter.
type CloudwatchGenericConfig struct {
	LogGroupName  string `json:"logGroupName"`
	LogStreamName string `json:"logStreamName"`
	Region        string `json:"awsRegion"`
	RoleARN       string `json:"awsRoleArn,omitempty"`
	AccessKey     string `json:"awsAccessKey,omitempty"`
	SecretKey     string `json:"awsSecretKey,omitempty"`
}

// GenericExporterConfig is a container for any type of exporter configuration.
// Note: This struct will be useful when we add more providers (check metric_exporter equivalent).
type GenericExporterConfig struct {
	Cloudwatch *CloudwatchGenericConfig
}

type CreateGenericExporterResponse struct {
	GenericExporter *GenericExporter `json:"createGenericExporter"`
}
type GetAllGenericExportersResponse struct {
	GenericExporters []*GenericExporter `json:"getAllGenericExporters"`
}

func (c *Client) CreateGenericExporter(ctx context.Context, name, region, typ, dataType string, config GenericExporterConfig) (*GenericExporter, error) {
	tflog.Trace(ctx, "Client.CreateGenericExporter")

	// Dynamically build the config
	// Note: This will be useful when we add more providers (check metric_exporter equivalent).
	var exporterConfig map[string]interface{}
	if config.Cloudwatch != nil {
		exporterConfig = map[string]interface{}{"configCloudWatch": config.Cloudwatch}
	} else {
		return nil, errors.New("exporter config cannot be empty")
	}

	req := map[string]interface{}{
		"operationName": "CreateGenericExporter",
		"query":         CreateGenericExporterMutation,
		"variables": map[string]interface{}{
			"projectId": c.projectID,
			"name":      name,
			"region":    region,
			"type":      typ,
			"dataType":  dataType,
			"config":    exporterConfig,
		},
	}

	var resp Response[CreateGenericExporterResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error executing API request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("API returned an error: %w", resp.Errors[0])
	}
	if resp.Data == nil || resp.Data.GenericExporter == nil {
		return nil, errors.New("API response did not contain exporter data")
	}

	return resp.Data.GenericExporter, nil
}

func (c *Client) GetAllGenericExporters(ctx context.Context) ([]*GenericExporter, error) {
	tflog.Trace(ctx, "Client.GetAllGenericExporters")
	req := map[string]interface{}{
		"operationName": "GetAllGenericExporters",
		"query":         GetAllGenericExportersQuery,
		"variables": map[string]string{
			"projectId": c.projectID,
		},
	}
	var resp Response[GetAllGenericExportersResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("error in the response: %w", resp.Errors[0])
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.GenericExporters, nil
}

func (c *Client) DeleteGenericExporter(ctx context.Context, id string) error {
	tflog.Trace(ctx, "Client.DeleteGenericExporter")

	req := map[string]interface{}{
		"operationName": "DeleteGenericExporter",
		"query":         DeleteGenericExporterMutation,
		"variables": map[string]any{
			"projectId":  c.projectID,
			"exporterId": id,
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

func (c *Client) UpdateGenericExporter(ctx context.Context, id, name string, config GenericExporterConfig) error {
	tflog.Trace(ctx, "Client.UpdateGenericExporter")

	// Dynamically build the config
	// Note: This will be useful when we add more providers (check metric_exporter equivalent).
	var exporterConfig map[string]interface{}
	if config.Cloudwatch != nil {
		exporterConfig = map[string]interface{}{"configCloudWatch": config.Cloudwatch}
	} else {
		return errors.New("exporter config cannot be empty for an update")
	}

	req := map[string]interface{}{
		"operationName": "UpdateGenericExporter",
		"query":         UpdateGenericExporterMutation,
		"variables": map[string]interface{}{
			"projectId":  c.projectID,
			"exporterId": id,
			"name":       name,
			"config":     exporterConfig,
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
