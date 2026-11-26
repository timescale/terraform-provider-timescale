package client

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type CrossRegionBackupRegion struct {
	Region    string `json:"region"`
	CreatedAt string `json:"createdAt"`
}

type CrossRegionBackupsStatus struct {
	Enabled bool                      `json:"enabled"`
	Regions []CrossRegionBackupRegion `json:"regions"`
}

type crossRegionBackupsStatusResponse struct {
	Status CrossRegionBackupsStatus `json:"crossRegionBackupsStatusV2"`
}

// BackupRegion returns the region the service's backups are copied to, or "" when
// cross-region backups are disabled. A service has at most one backup region.
func (s CrossRegionBackupsStatus) BackupRegion() string {
	if !s.Enabled || len(s.Regions) == 0 {
		return ""
	}
	return s.Regions[0].Region
}

// GetCrossRegionBackupsStatus returns the cross-region backup status of a service.
func (c *Client) GetCrossRegionBackupsStatus(ctx context.Context, serviceID string) (*CrossRegionBackupsStatus, error) {
	tflog.Trace(ctx, "Client.GetCrossRegionBackupsStatus")
	req := map[string]interface{}{
		"operationName": "GetCrossRegionBackupsStatus",
		"query":         GetCrossRegionBackupsStatusQuery,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
		},
	}
	var resp Response[crossRegionBackupsStatusResponse]
	if err := c.do(ctx, req, &resp); err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, resp.Errors[0]
	}
	if resp.Data == nil {
		return nil, errors.New("no response found")
	}
	return &resp.Data.Status, nil
}

// ToggleCrossRegionBackups enables or disables copying the service's backups to region.
func (c *Client) ToggleCrossRegionBackups(ctx context.Context, serviceID, region string, enable bool) error {
	tflog.Trace(ctx, "Client.ToggleCrossRegionBackups")
	req := map[string]interface{}{
		"operationName": "ToggleCrossRegionBackups",
		"query":         ToggleCrossRegionBackupsMutation,
		"variables": map[string]any{
			"projectId": c.projectID,
			"serviceId": serviceID,
			"region":    region,
			"enable":    enable,
		},
	}
	var resp Response[any]
	if err := c.do(ctx, req, &resp); err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return resp.Errors[0]
	}
	if resp.Data == nil {
		return errors.New("no response found")
	}
	return nil
}
