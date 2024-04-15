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
	configNameCloudwatch = "configCloudWatch"
	providerDatadog      = "datadog"
	providerCloudwatch   = "cloudwatch"
)

type exporterManager interface {
	Create(ctx context.Context, request *CreateExporterRequest) (*Exporter, error)
	GetAll(ctx context.Context) ([]*Exporter, error)
	Update(ctx context.Context, request *UpdateExporterRequest) error
	Delete() (*Exporter, error)
}

type metricExporter struct {
	provider string
	client   *Client
}

func (m *metricExporter) Create(ctx context.Context, req *CreateExporterRequest) (*Exporter, error) {
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

func (m *metricExporter) getConfigName() (string, error) {
	switch m.provider {
	case providerDatadog:
		return configNameDatadog, nil
	case providerCloudwatch:
		return configNameCloudwatch, nil
	default:
		return "", errors.New("unsupported metric provider " + m.provider)
	}
}

func (m *metricExporter) GetAll(ctx context.Context) ([]*Exporter, error) {
	tflog.Trace(ctx, "MetricExporter.GetAll")
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
	return resp.Data.Exporters, nil
}

func (m *metricExporter) Update(ctx context.Context, req *UpdateExporterRequest) error {
	tflog.Trace(ctx, "MetricExporter.Update")
	configName, err := m.getConfigName()
	if err != nil {
		return err
	}
	r := map[string]interface{}{
		"operationName": "UpdateMetricExporter",
		"query":         UpdateMetricExporterMutation,
		"variables": map[string]any{
			"projectId":  m.client.projectID,
			"exporterId": req.ExporterID,
			"name":       req.Name,
			"config": map[string]any{
				configName: req.Config,
			},
		},
	}
	var resp Response[any]
	err = m.client.do(ctx, r, &resp)
	if err = coalesceErrors(resp, err); err != nil {
		return err
	}
	return nil

}

func (m *metricExporter) Delete() (*Exporter, error) {
	//TODO implement me
	panic("implement me")
}

type GenericMetricExporter struct {
}

func (c *Client) newExporterManager(provider, data string) (exporterManager, error) {
	if data == "metrics" {
		return &metricExporter{
			provider: provider,
			client:   c,
		}, nil
	}
	if provider == "cloudwatch" && data == "log" {

	}
	return nil, errors.New("unsupported exporter: " + provider + " " + data)
}

type CloudWatchMetricConfig struct {
	LogGroupName  string  `json:"logGroupName"`
	LogStreamName string  `json:"logStreamName"`
	Namespace     string  `json:"namespace"`
	AwsAccessKey  string  `json:"awsAccessKey"`
	AwsSecretKey  string  `json:"awsSecretKey"`
	AwsRegion     string  `json:"awsRegion"`
	AwsRoleArn    *string `json:"awsRoleArn,omitempty"`
}

type DatadogMetricConfig struct {
	ApiKey string `json:"apiKey"`
	Site   string `json:"site,omitempty"`
}

type CreateExporterRequest struct {
	Provider   string
	Type       string
	Name       string
	RegionCode string
	Config     json.RawMessage
}

type UpdateExporterRequest struct {
	ExporterID string
	Provider   string
	Type       string
	Name       string
	Config     json.RawMessage
}

type GetExporterByIDRequest struct {
	ID       string
	Provider string
	Type     string
}

type GetExporterByNameRequest struct {
	Name     string
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

// GetConfig returns the exporter config as a JSON string
func (e *Exporter) GetConfig() (string, error) {
	var config interface{}
	switch e.Type {
	case "DATADOG":
		config = &DatadogMetricConfig{}
	case "CLOUDWATCH":
		config = &CloudWatchMetricConfig{}
	default:
		return "", fmt.Errorf("unsupported config type %s", e.Type)
	}
	if err := json.Unmarshal(e.Config, config); err != nil {
		return "", err
	}
	marshaled, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(marshaled), nil
}

func (c *Client) CreateExporter(ctx context.Context, request *CreateExporterRequest) (*Exporter, error) {
	manager, err := c.newExporterManager(request.Provider, request.Type)
	if err != nil {
		return nil, err
	}
	return manager.Create(ctx, request)
}

func (c *Client) UpdateExporter(ctx context.Context, request *UpdateExporterRequest) error {
	manager, err := c.newExporterManager(request.Provider, request.Type)
	if err != nil {
		return err
	}
	return manager.Update(ctx, request)
}

func (c *Client) GetExporterByID(ctx context.Context, request *GetExporterByIDRequest) (*Exporter, error) {
	manager, err := c.newExporterManager(request.Provider, request.Type)
	if err != nil {
		return nil, err
	}
	exporters, err := manager.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	exporter, ok := lo.Find(exporters, func(e *Exporter) bool {
		return e.ID == request.ID
	})
	if !ok {
		return nil, errNotFound
	}
	return exporter, nil
}

func (c *Client) GetExporterByName(ctx context.Context, request *GetExporterByNameRequest) (*Exporter, error) {
	manager, err := c.newExporterManager(request.Provider, request.Type)
	if err != nil {
		return nil, err
	}
	exporters, err := manager.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	e := lo.Filter(exporters, func(e *Exporter, _ int) bool {
		return e.Name == request.Name
	})
	if len(e) == 0 {
		return nil, errNotFound
	}
	if len(e) > 1 {
		return nil, errors.New("exporter names must be unique for importing")
	}
	return e[0], nil
}
