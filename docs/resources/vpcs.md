---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "timescale_vpcs Resource - timescale"
subcategory: ""
description: |-
  Schema for a VPC. Import can be done using your VPCs name
---

# timescale_vpcs (Resource)

Schema for a VPC. Import can be done using your VPCs name



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cidr` (String) The IPv4 CIDR block
- `region_code` (String) The region for this VPC.

### Optional

- `name` (String) VPC Name is the configurable name assigned to this vpc. If none is provided, a default will be generated by the provider.
- `timeouts` (Attributes) (see [below for nested schema](#nestedatt--timeouts))

### Read-Only

- `created` (String)
- `error_message` (String)
- `id` (Number) The ID of this resource.
- `project_id` (String)
- `provisioned_id` (String)
- `status` (String)
- `updated` (String)

<a id="nestedatt--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
