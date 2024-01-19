package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	VpcID      types.Int64     `tfsdk:"vpc_id"`
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
	MilliCPU        types.Int64 `tfsdk:"milli_cpu"`
	MemoryGB        types.Int64 `tfsdk:"memory_gb"`
	EnableHAReplica types.Bool  `tfsdk:"enable_ha_replica"`
}

func (d *ServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Service data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Service ID is the unique identifier for this service",
				Description:         "service id",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Service Name is the configurable name assigned to this resource. If none is provided, a default will be generated by the provider.",
				Description:         "service name",
				Computed:            true,
			},
			"region_code": schema.StringAttribute{
				MarkdownDescription: "Region Code is the physical data center where this service is located.",
				Computed:            true,
			},
			"spec": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						MarkdownDescription: "Hostname is the hostname of this service.",
						Description:         "hostname is the hostname of this service",
						Computed:            true,
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "Username is the Postgres username.",
						Description:         "username is the Postgres username",
						Computed:            true,
					},
					"port": schema.Int64Attribute{
						MarkdownDescription: "Port is the port assigned to this service.",
						Description:         "port is the port assigned to this service",
						Computed:            true,
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
									MarkdownDescription: "MilliCPU is the cpu allocated for this service.",
									Description:         "MilliCPU is the cpu allocated for this service.",
									Computed:            true,
								},
								"memory_gb": schema.Int64Attribute{
									MarkdownDescription: "MemoryGB is the memory allocated for this service.",
									Description:         "MemoryGB is the memory allocated for this service.",
									Computed:            true,
								},
								"enable_ha_replica": schema.BoolAttribute{
									MarkdownDescription: "EnableHAReplica defines if a replica will be provisioned for this service.",
									Description:         "EnableHAReplica defines if a replica will be provisioned for this service.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Created is the time this service was created.",
				Description:         "Created is the time this service was created.",
				Computed:            true,
			},
			"vpc_id": schema.Int64Attribute{
				MarkdownDescription: "VPC ID this service is linked to.",
				Description:         "VPC ID this service is linked to.",
				Optional:            true,
				Computed:            true,
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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service, got error: %s", err))
		return
	}
	state := serviceToDataModel(resp.Diagnostics, service)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, fmt.Sprintf("error updating terraform state %v", resp.Diagnostics.Errors()))
		return
	}
}

func serviceToDataModel(diag diag.Diagnostics, s *tsClient.Service) ServiceDataSourceModel {
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
	if s.VPCEndpoint != nil {
		if vpcID, err := strconv.ParseInt(s.VPCEndpoint.VPCId, 10, 64); err != nil {
			diag.AddError("Parse Error", "could not parse vpcID")
		} else {
			serviceModel.VpcID = types.Int64Value(vpcID)
		}
		serviceModel.Spec.Hostname = types.StringValue(s.VPCEndpoint.Host)
		serviceModel.Spec.Port = types.Int64Value(s.VPCEndpoint.Port)
	}
	for _, resource := range s.Resources {
		serviceModel.Resources = append(serviceModel.Resources, ResourceModel{
			ID: types.StringValue(resource.ID),
			Spec: ResourceSpecModel{
				MilliCPU:        types.Int64Value(resource.Spec.MilliCPU),
				MemoryGB:        types.Int64Value(resource.Spec.MemoryGB),
				EnableHAReplica: types.BoolValue(s.ReplicaStatus != ""),
			},
		})
	}
	return serviceModel
}
