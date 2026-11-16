package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

func backupRegionResource(t *testing.T, reply string) *serviceResource {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("TIMESCALE_DEV_URL", srv.URL)
	return &serviceResource{client: tsClient.NewClient("token", "proj", "test", "1.0.0")}
}

func TestReadBackupRegion(t *testing.T) {
	ctx := context.Background()

	t.Run("enabled", func(t *testing.T) {
		r := backupRegionResource(t, `{"data":{"crossRegionBackupsStatusV2":{"enabled":true,"regions":[{"region":"eu-central-1","createdAt":"x"}]}}}`)
		got, err := r.readBackupRegion(ctx, "svc", types.StringNull())
		require.NoError(t, err)
		require.Equal(t, types.StringValue("eu-central-1"), got)
	})

	t.Run("disabled is null", func(t *testing.T) {
		r := backupRegionResource(t, `{"data":{"crossRegionBackupsStatusV2":{"enabled":false,"regions":[]}}}`)
		got, err := r.readBackupRegion(ctx, "svc", types.StringValue("eu-central-1"))
		require.NoError(t, err)
		require.True(t, got.IsNull(), "disabled backups must show up as drift when a region is configured")
	})

	t.Run("error while unmanaged is tolerated", func(t *testing.T) {
		r := backupRegionResource(t, `{"errors":[{"message":"unsupported"}]}`)
		got, err := r.readBackupRegion(ctx, "svc", types.StringNull())
		require.NoError(t, err)
		require.True(t, got.IsNull())
	})

	t.Run("error while managed fails", func(t *testing.T) {
		r := backupRegionResource(t, `{"errors":[{"message":"unsupported"}]}`)
		_, err := r.readBackupRegion(ctx, "svc", types.StringValue("eu-central-1"))
		require.Error(t, err)
	})
}
