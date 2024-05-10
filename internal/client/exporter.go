package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"
)

type Exporter struct {
	ID         string          `json:"id"`
	ProjectID  string          `json:"projectId"`
	Created    time.Time       `json:"created"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	RegionCode string          `json:"regionCode"`
	Config     json.RawMessage `json:"config"`
}

type GetAllMetricExportersResponse struct {
	Exporters []*Exporter `json:"getAllMetricExporters"`
}

type GetAllGenericExporterResponse struct {
	Exporters []*Exporter `json:"getAllGenericExporters"`
}

type GetExporterByIDRequest struct {
	ID string
}

type AttachExporterRequest struct {
	ServiceID  string
	ExporterID string
}

type DetachExporterRequest struct {
	ServiceID  string
	ExporterID string
}

func (c *Client) getAllMetricExporters(ctx context.Context) ([]*Exporter, error) {
	tflog.Trace(ctx, "MetricExporter.GetAll")
	req := graphQLRequest{
		operationName: "GetAllMetricExporters",
		query:         GetAllMetricExporters,
		variables: map[string]interface{}{
			"projectId": c.projectID,
		},
	}
	var resp Response[GetAllMetricExportersResponse]
	err := c.do(ctx, req.build(), &resp)
	if err = coalesceErrors(resp, err); err != nil {
		return nil, err
	}
	return resp.Data.Exporters, nil
}

func (c *Client) getAllLogExporters(ctx context.Context) ([]*Exporter, error) {
	tflog.Trace(ctx, "MetricExporter.GetAllLogExporters")
	req := graphQLRequest{
		operationName: "GetAllGenericExporters",
		query:         GetAllGenericMetricExporters,
		variables: map[string]interface{}{
			"projectId": c.projectID,
		},
	}
	var resp Response[GetAllGenericExporterResponse]
	err := c.do(ctx, req.build(), &resp)
	if err = coalesceErrors(resp, err); err != nil {
		return nil, err
	}
	return resp.Data.Exporters, nil
}

func (c *Client) getAllExporters(ctx context.Context) ([]*Exporter, error) {
	tflog.Trace(ctx, "Client.getAllExporters")
	metricExporters, err := c.getAllMetricExporters(ctx)
	if err != nil {
		return nil, err
	}
	logExporters, err := c.getAllLogExporters(ctx)
	if err != nil {
		return nil, err
	}
	return append(metricExporters, logExporters...), nil
}

func (c *Client) GetExporterByID(ctx context.Context, request *GetExporterByIDRequest) (*Exporter, error) {
	tflog.Trace(ctx, "Client.GetExporterByID")
	exporters, err := c.getAllExporters(ctx)
	if err != nil {
		return nil, err
	}
	e := lo.Filter(exporters, func(e *Exporter, _ int) bool {
		return e.ID == request.ID
	})
	if len(e) == 0 {
		return nil, errNotFound
	}
	return e[0], nil
}

func (c *Client) AttachMetricExporter(ctx context.Context, request *AttachExporterRequest) error {
	tflog.Trace(ctx, "Client.AttachMetricExporter")
	req := &graphQLRequest{
		operationName: "AttachServiceToMetricExporter",
		query:         AttachMetricExporterMutation,
		variables: map[string]interface{}{
			"projectId":  c.projectID,
			"serviceId":  request.ServiceID,
			"exporterId": request.ExporterID,
		},
	}
	var resp Response[any]
	err := c.do(ctx, req.build(), &resp)
	return coalesceErrors(resp, err)
}

func (c *Client) AttachLogExporter(ctx context.Context, request *AttachExporterRequest) error {
	tflog.Trace(ctx, "Client.AttachLogExporter")
	req := &graphQLRequest{
		operationName: "AttachServiceToGenericExporter",
		query:         AttachGenericExporterMutation,
		variables: map[string]interface{}{
			"projectId":  c.projectID,
			"serviceId":  request.ServiceID,
			"exporterId": request.ExporterID,
		},
	}
	var resp Response[any]
	err := c.do(ctx, req.build(), &resp)
	return coalesceErrors(resp, err)
}

func (c *Client) DetachLogExporter(ctx context.Context, request *DetachExporterRequest) error {
	tflog.Trace(ctx, "Client.DetachLogExporter")
	req := &graphQLRequest{
		operationName: "DetachServiceFromGenericExporter",
		query:         DetachGenericMetricExporterMutation,
		variables: map[string]interface{}{
			"projectId":  c.projectID,
			"serviceId":  request.ServiceID,
			"exporterId": request.ExporterID,
		},
	}
	var resp Response[any]
	err := c.do(ctx, req.build(), &resp)
	return coalesceErrors(resp, err)
}

func (c *Client) DetachMetricExporter(ctx context.Context, request *DetachExporterRequest) error {
	tflog.Trace(ctx, "Client.DetachMetricExporter")
	req := &graphQLRequest{
		operationName: "DetachServiceFromMetricExporter",
		query:         DetachMetricExporterMutation,
		variables: map[string]interface{}{
			"projectId":  c.projectID,
			"serviceId":  request.ServiceID,
			"exporterId": request.ExporterID,
		},
	}
	var resp Response[any]
	err := c.do(ctx, req.build(), &resp)
	return coalesceErrors(resp, err)
}
