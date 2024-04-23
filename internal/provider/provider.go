package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/providervalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure TimescaleProvider satisfies various provider interfaces.
var _ provider.ProviderWithConfigValidators = &TimescaleProvider{}

// TimescaleProvider defines the provider implementation.
type TimescaleProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// terraformVersion is the caller's terraform version.
	terraformVersion string
}

// TimescaleProviderModel describes the provider data model.
type TimescaleProviderModel struct {
	ProjectID   types.String `tfsdk:"project_id"`
	AccessToken types.String `tfsdk:"access_token"`
	AccessKey   types.String `tfsdk:"access_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
}

func (p *TimescaleProvider) Metadata(ctx context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	tflog.Trace(ctx, "TimescaleProvider.Metadata")
	resp.Version = p.version
	resp.TypeName = "timescale"
}

// Schema defines the provider-level schema for configuration data.
func (p *TimescaleProvider) Schema(ctx context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	tflog.Trace(ctx, "TimescaleProvider.Schema")
	resp.Schema = schema.Schema{
		MarkdownDescription: "The Terraform provider for [Timescale](https://console.cloud.timescale.com/).",
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access Token",
				Optional:            true,
				Sensitive:           true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "Project ID",
				Required:            true,
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

func (p *TimescaleProvider) ConfigValidators(_ context.Context) []provider.ConfigValidator {
	return []provider.ConfigValidator{
		providervalidator.Conflicting(
			path.MatchRoot("access_token"),
			path.MatchRoot("access_key"),
		),
		providervalidator.Conflicting(
			path.MatchRoot("access_token"),
			path.MatchRoot("secret_key"),
		),
		providervalidator.AtLeastOneOf(
			path.MatchRoot("access_token"),
			path.MatchRoot("access_key"),
		),
		providervalidator.RequiredTogether(
			path.MatchRoot("access_key"),
			path.MatchRoot("secret_key"),
		),
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

	p.terraformVersion = req.TerraformVersion
	client := tsClient.NewClient(data.AccessToken.ValueString(), data.ProjectID.ValueString(),
		p.version, p.terraformVersion)
	if !data.AccessKey.IsNull() && !data.SecretKey.IsNull() {
		err := tsClient.JWTFromCC(client, data.AccessKey.ValueString(), data.SecretKey.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get JWT from CC, got error: %s", err))
		}
	}
	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources defines the resources implemented in the provider.
func (p *TimescaleProvider) Resources(ctx context.Context) []func() resource.Resource {
	tflog.Trace(ctx, "TimescaleProvider.Resources")
	return []func() resource.Resource{
		NewServiceResource,
		NewVpcsResource,
		NewPeeringConnectionResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *TimescaleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	tflog.Trace(ctx, "TimescaleProvider.DataSources")
	return []func() datasource.DataSource{
		NewProductsDataSource,
		NewServiceDataSource,
		NewVpcsDataSource,
		NewExporterDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TimescaleProvider{
			version: version,
		}
	}
}
