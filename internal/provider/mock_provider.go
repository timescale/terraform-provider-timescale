package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure MockProvider satisfies various provider interfaces.
var _ provider.Provider = &MockProvider{}

// MockProvider defines the provider implementation.
type MockProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// terraformVersion is the caller's terraform version.
	terraformVersion string
}

// MockProviderModel describes the provider data model.
type MockProviderModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	AccessToken types.String `tfsdk:"access_token"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
}

func (p *MockProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Trace(ctx, "MockProvider.Metadata")
	resp.Version = p.version
	resp.TypeName = "timescale"
}

// Schema defines the provider-level schema for configuration data.
func (p *MockProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Trace(ctx, "MockProvider.Schema")
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Terraform provider for [Timescale Cloud](https://console.cloud.timescale.com/).",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access Token",
				Optional:            true,
				Sensitive:           true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID",
				Optional:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "Access Key",
				Optional:            true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "Secret Key",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *MockProvider) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.Conflicting(
			path.MatchRoot("access_token"),
			path.MatchRoot("access_key"),
		),
		resourcevalidator.Conflicting(
			path.MatchRoot("access_token"),
			path.MatchRoot("secret_key"),
		),
	}
}

// Configure initializes a Timescale API client for data sources and resources.
func (p *MockProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Trace(ctx, "MockProvider.Configure")
	var data MockProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	p.terraformVersion = req.TerraformVersion
	client := tsClient.NewMockClient()
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *MockProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "MockProvider.Resources")
	return []func() resource.Resource{
		NewServiceResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *MockProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "MockProvider.DataSources")
	return []func() datasource.DataSource{
		NewProductsDataSource,
		NewServiceDataSource,
	}
}

func NewMock(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MockProvider{
			version: version,
		}
	}
}
