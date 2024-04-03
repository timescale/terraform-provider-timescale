package client

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/samber/lo"
)

const (
	configNameDatadog    = "configDatadog"
	configNameCloudwatch = "configCloudwatch"
	providerDatadog      = "datadog"
	providerCloudwatch   = "cloudwatch"
)

type exporterManager interface {
	Create(ctx context.Context, request *CreateExporterRequest) (*Exporter, error)
	Get(ctx context.Context, request *GetExporterRequest) (*Exporter, error)
	Update() (*Exporter, error)
	Delete() (*Exporter, error)
}

type MetricExporter struct {
	provider string
	client   *Client
}

func (m *MetricExporter) Create(ctx context.Context, req *CreateExporterRequest) (*Exporter, error) {
	configName, err := m.getConfigName()
	if err != nil {
		return nil, err
	}
	if req.Name == "" {
		r, err := rand.Int(rand.Reader, big.NewInt(90000))
		if err != nil {
			return nil, err
		}
		req.Name = fmt.Sprintf("exporter-%d", 10000+r.Int64())
	}
	vars := map[string]any{
		"projectId":  m.client.projectID,
		"name":       req.Name,
		"regionCode": req.RegionCode,
		"config": map[string]any{
			configName: req.Config,
		},
	}
	request := map[string]interface{}{
		"operationName": "CreateMetricExporter",
		"query":         CreateMetricExporterMutation,
		"variables":     vars,
	}
	var resp Response[CreateMetricExporterResponse]
	err = m.client.do(ctx, request, &resp)
	if err = coalesceErrors(resp, err); err != nil {
		return nil, err
	}
	return resp.Data.Exporter, nil
}

func (m *MetricExporter) getConfigName() (string, error) {
	switch m.provider {
	case providerDatadog:
		return configNameDatadog, nil
	case providerCloudwatch:
		return configNameCloudwatch, nil
	default:
		return "", errors.New("unsupported metric provider " + m.provider)
	}
}

func (m *MetricExporter) Get(ctx context.Context, request *GetExporterRequest) (*Exporter, error) {
	tflog.Trace(ctx, "MetricExporter.Get")
	req := map[string]interface{}{
		"operationName": "GetAllMetricExporters",
		"query":         GetAllMetricExporters,
		"variables": map[string]string{
			"projectId": m.client.projectID,
		},
	}
	var resp Response[GetAllMetricExporterResponse]
	err := m.client.do(ctx, req, &resp)
	if err = coalesceErrors(resp, err); err != nil {
		return nil, err
	}
	exporter, ok := lo.Find(resp.Data.Exporters, func(e *Exporter) bool {
		return e.ID == request.ID
	})
	if !ok {
		return nil, errNotFound
	}
	return exporter, nil
}

func (m *MetricExporter) Update() (*Exporter, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MetricExporter) Delete() (*Exporter, error) {
	//TODO implement me
	panic("implement me")
}

type GenericMetricExporter struct {
}

func (c *Client) newExporterFactory(provider, data string) (exporterManager, error) {
	if data == "metrics" {
		return &MetricExporter{
			provider: provider,
			client:   c,
		}, nil
	}
	if provider == "cloudwatch" && data == "log" {

	}
	return nil, errors.New("unsupported exporter: " + provider + " " + data)
}

type CloudWatchMetricConfigInput struct {
	LogGroupName  string  `json:"logGroupName"`
	LogStreamName string  `json:"logStreamName"`
	Namespace     string  `json:"namespace"`
	AwsAccessKey  string  `json:"awsAccessKey"`
	AwsSecretKey  string  `json:"awsSecretKey"`
	AwsRegion     string  `json:"awsRegion"`
	AwsRoleArn    *string `json:"awsRoleArn,omitempty"`
}

type DatadogMetricConfigInput struct {
	ApiKey string  `json:"apiKey"`
	Site   *string `json:"site,omitempty"`
}

type CreateExporterRequest struct {
	Provider   string
	Type       string
	Name       string
	RegionCode string
	Config     json.RawMessage
}

type GetExporterRequest struct {
	ID       string
	Provider string
	Type     string
}

type CreateMetricExporterResponse struct {
	Exporter *Exporter `json:"createMetricExporter"`
}

type GetAllMetricExporterResponse struct {
	Exporters []*Exporter `json:"getAllMetricExporters"`
}

type Exporter struct {
	ID         string          `json:"id"`
	ProjectID  string          `json:"projectId"`
	Created    time.Time       `json:"created"`
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	RegionCode string          `json:"regionCode"`
	Config     json.RawMessage `json:"config"`
}

func (c *Client) CreateExporter(ctx context.Context, request *CreateExporterRequest) (*Exporter, error) {
	manager, err := c.newExporterFactory(request.Provider, request.Type)
	if err != nil {
		return nil, err
	}
	return manager.Create(ctx, request)
}

func (c *Client) GetExporter(ctx context.Context, request *GetExporterRequest) (*Exporter, error) {
	manager, err := c.newExporterFactory(request.Provider, request.Type)
	if err != nil {
		return nil, err
	}
	return manager.Get(ctx, request)
}
