package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure TimescaleProvider satisfies various provider interfaces.
var _ provider.Provider = &TimescaleProvider{}

// TimescaleProvider defines the provider implementation.
type TimescaleProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TimescaleProviderModel describes the provider data model.
type TimescaleProviderModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	AccessToken types.String `tfsdk:"access_token"`
}

func (p *TimescaleProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Trace(ctx, "TimescaleProvider.Metadata")
	resp.Version = p.version
	resp.TypeName = "timescale"
}

// Schema defines the provider-level schema for configuration data.
func (p *TimescaleProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Trace(ctx, "TimescaleProvider.Schema")
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access Token",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID",
				Required:            true,
			},
		},
	}
}

// Configure initializes a Timescale API client for data sources and resources.
func (p *TimescaleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Trace(ctx, "TimescaleProvider.Configure")
	var data TimescaleProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	//TODO: Validate the configuration

	client := tsClient.NewClient(data.AccessToken.ValueString(), data.ProjectID.ValueString(), p.version)
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *TimescaleProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "TimescaleProvider.Resources")
	return []func() resource.Resource{
		NewServiceResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *TimescaleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "TimescaleProvider.DataSources")
	return []func() datasource.DataSource{
		NewProductsDataSource,
		NewServiceDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TimescaleProvider{
			version: version,
		}
	}
}
