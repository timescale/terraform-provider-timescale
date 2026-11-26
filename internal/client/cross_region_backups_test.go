package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCrossRegionBackupsStatus(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"crossRegionBackupsStatusV2":{"enabled":true,"regions":[{"region":"eu-central-1","createdAt":"2026-01-01T00:00:00Z"}]}}}`)

	st, err := c.GetCrossRegionBackupsStatus(context.Background(), "svc")
	require.NoError(t, err)
	require.Equal(t, "eu-central-1", st.BackupRegion())
	require.Equal(t, "GetCrossRegionBackupsStatus", (*calls)[0].Op)
	require.Equal(t, "proj", (*calls)[0].Variables["projectId"])
	require.Equal(t, "svc", (*calls)[0].Variables["serviceId"])
}

func TestCrossRegionBackupsStatus_BackupRegion(t *testing.T) {
	require.Equal(t, "", CrossRegionBackupsStatus{}.BackupRegion())
	require.Equal(t, "", CrossRegionBackupsStatus{Regions: []CrossRegionBackupRegion{{Region: "x"}}}.BackupRegion(), "disabled status must report no region")
	require.Equal(t, "x", CrossRegionBackupsStatus{Enabled: true, Regions: []CrossRegionBackupRegion{{Region: "x"}}}.BackupRegion())
}

func TestToggleCrossRegionBackups(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"toggleCrossRegionBackupsV2":true}}`)

	require.NoError(t, c.ToggleCrossRegionBackups(context.Background(), "svc", "eu-central-1", true))
	v := (*calls)[0].Variables
	require.Equal(t, "ToggleCrossRegionBackups", (*calls)[0].Op)
	require.Equal(t, "proj", v["projectId"])
	require.Equal(t, "svc", v["serviceId"])
	require.Equal(t, "eu-central-1", v["region"])
	require.Equal(t, true, v["enable"])
}

func TestToggleCrossRegionBackups_Error(t *testing.T) {
	c, _ := newRecordingClient(t, `{"errors":[{"message":"nope"}]}`)
	require.Error(t, c.ToggleCrossRegionBackups(context.Background(), "svc", "eu-central-1", true))
}
