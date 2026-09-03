# Private Link examples

Two complete, runnable setups. Each one provisions a small Tiger Cloud service, a private endpoint claimed as a private link connection, a small VM to test from, and an output command that runs `SELECT 1` over the private path.

| Directory | Cloud | Claim identifier |
| --------- | ----- | ---------------- |
| [`aws/`](./aws) | AWS | VPC endpoint ID (`vpce-…`), straight from `aws_vpc_endpoint.example.id` |
| [`azure/`](./azure) | Azure | Private endpoint's `resourceGuid`, read via the `azapi` provider |

## How private link works here

Tiger Cloud's endpoint service is open. Anyone can create a private endpoint against it, every connection is accepted, and the connection is left **unowned**. An unowned connection reaches nothing — it matches no routing rule and cannot be bound to a service.

Ownership is established by **claiming** the connection: you read an identifier from your own cloud resource and submit it while authenticated to a project. Authentication to the project is the proof of ownership; nothing about the cloud account is trusted. That is what makes a shared third-party provider account work — several customers can connect through one provider account, each claiming their own connection.

```
aws_vpc_endpoint / azurerm_private_endpoint   you create the endpoint
              ↓
timescale_privatelink_connection              you claim it
              ↓
timescale_service.private_endpoint_connection_ids   you attach services
              ↓
aws_route53_zone / azurerm_private_dns_zone   you point DNS at the endpoint
```

## You own the DNS

Tiger Cloud does not publish DNS records for private link. You create a private zone in your own VPC or VNet and point the service hostname at your endpoint. Both examples do this for you.

The hostname is **the same one the public endpoint uses**, so the certificate Tiger Cloud already issues validates with `sslmode=verify-full`. Only two things differ from the public path:

- **Resolution** — your private zone shadows the public name inside your network. AWS uses an alias A record to the endpoint's regional DNS name, which balances across every healthy ENI, giving AZ failover. Azure uses a plain A record to the endpoint IP, because an Azure private endpoint has one IP for its lifetime.
- **The port** — allocated per binding, and not always 5432. Read it from `timescale_service.example.port`, the Tiger Cloud console, or the API. Both examples output it.

  The allocation differs by cloud. On **AWS** a binding takes a reserved port by role: 5432 primary, 5433 replica, 6432 pooler. On **Azure** there are no reserved ports: every binding draws from the backend's pool, so expect a port in 50000-51000. The security group and NSG rules in these examples reflect that split.

A wildcard record will not work: Tiger Cloud issues per-service certificates, so only exact hostnames validate. Add one record per service.

## Availability Zones (AWS only)

Tiger Cloud serves PrivateLink from exactly **two** Availability Zones per region, and an endpoint interface in any other zone cannot reach your service. The AWS example creates a subnet in both and spans the endpoint across them, so it survives losing a zone. The zone pairs are hardcoded in `privatelink_availability_zone_ids` in `aws/main.tf`.

The pairs are **Availability Zone IDs** (`use1-az1`), not names (`us-east-1a`). AWS maps names to different physical zones in every account, so matching on the name puts the endpoint in the wrong place — the example translates IDs to this account's names via `aws_availability_zones`. This also means the test instance is placed explicitly: not every instance type is offered in every zone.

Azure needs none of this. A private endpoint there is a single NIC with one IP, and Microsoft handles zone resilience below it.

## Running an example

```bash
cd aws   # or: cd azure

cp terraform.tfvars.example terraform.tfvars
# fill in your Tiger Cloud credentials and project ID

terraform init
terraform apply

# test the private path
eval "$(terraform output -raw test_command)"

# confirm the hostname resolves to the endpoint, not the public IP
eval "$(terraform output -raw dns_check_command)"
```

`terraform output test_command` is marked sensitive because it embeds the service password. `terraform output -raw` prints it.

Cloud credentials come from the usual environment for each provider: `AWS_PROFILE` or `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` for AWS, `az login` or `ARM_*` variables for Azure.

## Destroying

```bash
terraform destroy
```

By default this releases the claim from Terraform state and **leaves the connection in place**, because rejecting a connection is irreversible: a rejected connection cannot be re-claimed, and you would have to recreate the private endpoint. Re-applying claims the same connection again.

If you want `terraform destroy` to reject the connection in your cloud account too, set `reject_on_destroy = true` on the resource. Note that `terraform destroy` also deletes the endpoint itself in these examples, and once your endpoint is gone Tiger Cloud's sync marks the connection removed on its own.

## Troubleshooting

**`Unable to claim Private Link connection` / connection not found.** The connection only exists in Tiger Cloud once a sync has picked it up from the cloud provider. The resource asks for a sync and retries for 10 minutes by default, which is normally plenty. If it times out, check that `claim_identifier` matches your endpoint: the VPC endpoint ID on AWS, the `resourceGuid` on Azure — not the endpoint name.

**`connection is already claimed`.** A claim identifier can be owned by exactly one project. Re-claiming a connection *your* project already owns is idempotent and succeeds; this error means another project holds it.

**The hostname resolves to a public IP from the VM.** The private zone is not linked to the network, or the record name does not match the service hostname exactly. Run the `dns_check_command` output and compare against `service_hostname`.

**Connection times out.** Check the port. It is allocated per binding and is often not 5432 — use `terraform output service_port`.

**`root certificate file "~/.postgresql/root.crt" does not exist`.** Add `sslrootcert=system` to the connection string, as the example's `test_command` does. Tiger Cloud certificates come from a public CA, so the OS trust store validates them and `sslmode=verify-full` works without downloading anything. Requires libpq 16 or newer.

**`SSL error: certificate verify failed` on a service you just created.** Wait a few minutes and retry. A new service starts with a self-signed certificate so it is available immediately, and a CA-signed one replaces it shortly after — usually within 30 minutes. `sslmode=verify-full` cannot succeed until that swap lands, because the system trust store does not contain the temporary self-signed issuer. Check which certificate is being served with:

```bash
openssl s_client -starttls postgres -connect <hostname>:<port> </dev/null 2>&1 | grep -E 'i:|Verify return code'
```

An issuer of `CN = ca.timescale.com` means the swap has not happened yet; a public CA such as `Google Trust Services` means it has. See [Strict SSL mode](https://www.tigerdata.com/docs/deploy/tiger-cloud/tiger-cloud-aws/security/strict-ssl).
