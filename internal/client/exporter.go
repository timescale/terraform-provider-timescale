package client

import (
	"context"
	"errors"
	"time"
)

const (
	configNameDatadog    = "configDatadog"
	configNameCloudwatch = "configCloudwatch"
	providerDatadog      = "datadog"
	providerCloudwatch   = "cloudwatch"
)

type exporterManager interface {
	Create(ctx context.Context, request CreateExporterRequest) (*Exporter, error)
	Read(ctx context.Context, id string) (*Exporter, error)
	Update() (*Exporter, error)
	Delete() (*Exporter, error)
}

type MetricExporter struct {
	provider string
	client   *Client
}

func (m *MetricExporter) Create(ctx context.Context, req CreateExporterRequest) (*Exporter, error) {
	configName, err := m.getConfigName()
	if err != nil {
		return nil, err
	}
	vars := map[string]any{
		"projectId":  m.client.projectID,
		"name":       req.Name,
		"regionCode": req.RegionCode,
		"config": map[string]interface{}{
			configName: req.Config,
		},
	}
	request := map[string]interface{}{
		"operationName": "CreateExporter",
		"query":         CreateMetricExporterMutation,
		"variables":     vars,
	}
	var resp Response[ExporterResponse]
	if err := m.client.do(ctx, request, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.Exporter, nil
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

func (m *MetricExporter) Read(ctx context.Context, id string) (*Exporter, error) {
	//TODO implement me
	panic("implement me")
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

func newExporterFactory(provider, data string, c *Client) (exporterManager, error) {
	if data == "METRIC" {
		return &MetricExporter{
			provider: provider,
			client:   c,
		}, nil
	}
	if provider == "cloudwatch" && data == "LOG" {

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
	Config     interface{}
}

type ExporterResponse struct {
	Exporter Exporter
}

type Exporter struct {
	ID         string
	ProjectID  string
	Created    time.Time
	Name       string
	Type       string
	RegionCode string
	Config     []byte
}

func (c *Client) CreateExporter(ctx context.Context, request CreateExporterRequest) (*Exporter, error) {
	manager, err := newExporterFactory(request.Provider, request.Type, c)
	if err != nil {
		return nil, err
	}
	return manager.Create(ctx, request)
}
