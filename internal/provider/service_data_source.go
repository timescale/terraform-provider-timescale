package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	tsClient "github.com/timescale/terraform-provider-timescale/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ServiceDataSource{}
var _ datasource.DataSourceWithConfigure = &ServiceDataSource{}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

// ServiceDataSource defines the data source implementation.
type ServiceDataSource struct {
	client *tsClient.Client
}

// ServiceDataSourceModel describes the data source data model.
type ServiceDataSourceModel struct {
	ID         types.String    `tfsdk:"id"`
	Name       types.String    `tfsdk:"name"`
	RegionCode types.String    `tfsdk:"region_code"`
	Spec       SpecModel       `tfsdk:"spec"`
	Resources  []ResourceModel `tfsdk:"resources"`
	Created    types.String    `tfsdk:"created"`
}

type SpecModel struct {
	Hostname types.String `tfsdk:"hostname"`
	Username types.String `tfsdk:"username"`
	Port     types.Int64  `tfsdk:"port"`
}

type ResourceModel struct {
	ID   types.String      `tfsdk:"id"`
	Spec ResourceSpecModel `tfsdk:"spec"`
}

type ResourceSpecModel struct {
	MilliCPU  types.Int64 `tfsdk:"milli_cpu"`
	MemoryGB  types.Int64 `tfsdk:"memory_gb"`
	StorageGB types.Int64 `tfsdk:"storage_gb"`
}

func (d *ServiceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Service data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service ID",
				Description:         "service id",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service name",
				Description:         "service name",
				Computed:            true,
			},
			"region_code": schema.StringAttribute{
				Computed: true,
			},
			"spec": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Computed: true,
					},
					"username": schema.StringAttribute{
						Computed: true,
					},
					"port": schema.Int64Attribute{
						Computed: true,
					},
				},
				Computed: true,
			},
			"resources": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"spec": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"milli_cpu": schema.Int64Attribute{
									Computed: true,
								},
								"memory_gb": schema.Int64Attribute{
									Computed: true,
								},
								"storage_gb": schema.Int64Attribute{
									Computed: true,
								},
							},
						},
					},
				},
			},
			"created": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *ServiceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	tflog.Trace(ctx, "ServiceDataSource.Configure")

	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*tsClient.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Client Type",
			fmt.Sprintf("Expected *tsClient.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Trace(ctx, "ServiceDataSource.Read")

	var id string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("id"), &id)...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error reading terraform plan %v", resp.Diagnostics.Errors()))
		return
	}

	tflog.Info(ctx, "Getting Service: "+id)
	service, err := d.client.GetService(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create service, got error: %s", err))
		return
	}
	state := serviceToDataModel(service)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func serviceToDataModel(s *tsClient.Service) ServiceDataSourceModel {
	serviceModel := ServiceDataSourceModel{
		ID:         types.StringValue(s.ID),
		Name:       types.StringValue(s.Name),
		RegionCode: types.StringValue(s.RegionCode),
		Spec: SpecModel{
			Hostname: types.StringValue(s.ServiceSpec.Hostname),
			Username: types.StringValue(s.ServiceSpec.Username),
			Port:     types.Int64Value(s.ServiceSpec.Port),
		},
		Created: types.StringValue(s.Created),
	}
	for _, resource := range s.Resources {
		serviceModel.Resources = append(serviceModel.Resources, ResourceModel{
			ID: types.StringValue(resource.ID),
			Spec: ResourceSpecModel{
				MilliCPU:  types.Int64Value(resource.Spec.MilliCPU),
				MemoryGB:  types.Int64Value(resource.Spec.MemoryGB),
				StorageGB: types.Int64Value(resource.Spec.StorageGB),
			},
		})
	}
	return serviceModel
}