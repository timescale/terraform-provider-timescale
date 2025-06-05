package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type DatadogMetricExporter struct {
	ID           string        `json:"id"`
	ExporterUUID string        `json:"exporterUuid"`
	ProjectID    string        `json:"projectId"`
	Created      string        `json:"created"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Config       DatadogConfig `json:"config"`
}

type DatadogConfig struct {
	APIKey string `json:"apiKey"`
	Site   string `json:"site"`
}

type CreateDatadogMetricExporterResponse struct {
	DatadogMetricExporter *DatadogMetricExporter `json:"createMetricExporter"`
}

type DatadogExporter struct {
	ID string `json:"id"`
}

func (c *Client) CreateDatadogMetricExporter(ctx context.Context, name, region, apiKey, site string) (*DatadogMetricExporter, error) {
	tflog.Trace(ctx, "Client.CreateDatadogMetricExporter")

	req := map[string]interface{}{
		"operationName": "CreateMetricExporter",
		"query":         CreateMetricExporterMutation,
		"variables": map[string]interface{}{
			"projectId":  c.projectID,
			"name":       name,
			"regionCode": region,
			"config": map[string]interface{}{
				"configDatadog": map[string]string{
					"apiKey": apiKey,
					"site":   site,
				},
			},
		},
	}

	var resp Response[CreateDatadogMetricExporterResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("error doing the request: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("error in the response: %w", resp.Errors[0])
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return resp.Data.DatadogMetricExporter, nil
}
