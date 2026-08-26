# terraform-provider-gcxtras

A companion Terraform provider for Genesys Cloud that manages resources not covered by the official provider.

Published by [gsyspslab](https://gsyspslab.com).

## Supported Resources

| Resource | Description |
|----------|-------------|
| `gcxtras_sta_program` | Speech and Text Analytics Program (full CRUD) |
| `gcxtras_sta_topic` | Speech and Text Analytics Topic (full CRUD) |
| `gcxtras_processautomation_scheduled_trigger` | Process Automation Scheduled Trigger (full CRUD) |

## Supported Data Sources

| Data Source | Description |
|-------------|-------------|
| `gcxtras_sta_program` | Look up an existing STA Program by name |
| `gcxtras_sta_topic` | Look up an existing STA Topic by name and dialect |

## Authentication

This provider uses the same OAuth configuration as the official Genesys Cloud Terraform Provider.

### Provider Configuration

```hcl
provider "gcxtras" {
  oauthclient_id     = "your-client-id"
  oauthclient_secret = "your-client-secret"
  aws_region         = "eu-west-1"
  sdk_debug          = true  # optional: logs API calls to gcxtras_debug.log
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GENESYSCLOUD_OAUTHCLIENT_ID` | OAuth Client ID |
| `GENESYSCLOUD_OAUTHCLIENT_SECRET` | OAuth Client Secret |
| `GENESYSCLOUD_REGION` | AWS Region (e.g. `eu-west-1`) |

## Usage

```hcl
terraform {
  required_providers {
    gcxtras = {
      source  = "gsyspslab/gcxtras"
      version = "~> 0.1"
    }
  }
}

resource "gcxtras_sta_topic" "cancellation" {
  name         = "Account Cancellation"
  description  = "Detects cancellation intent"
  dialect      = "en-GB"
  strictness   = "72"
  participants = "External"
  tags         = ["churn"]
}

data "gcxtras_sta_topic" "billing" {
  name    = "Billing Inquiry"
  dialect = "en-GB"
}

resource "gcxtras_sta_program" "example" {
  name        = "My Analytics Program"
  description = "Managed by Terraform"
  published   = true
  tags        = ["terraform"]
  topic_ids = [
    gcxtras_sta_topic.cancellation.id,
    data.gcxtras_sta_topic.billing.id,
  ]
}

data "gcxtras_sta_program" "default" {
  name = "General"
}
```

## Development

### Building

```bash
make build    # Build the binary
make install  # Install to local Terraform plugin directory
```

After `make install`, run `terraform init -upgrade` in your Terraform project to pick up the new binary.

### Regenerating Documentation

Documentation is auto-generated from schema descriptions using `tfplugindocs`:

```bash
# Install (one-time)
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Generate (run from project root)
~/go/bin/tfplugindocs generate --provider-name gcxtras
```

This reads the schema `Description` fields and example files from `examples/` to produce the `docs/` directory. The Terraform Registry renders these automatically.

### Project Structure for Documentation

`tfplugindocs` expects examples in this layout:

```
examples/
  provider/provider.tf
  resources/gcxtras_sta_program/resource.tf
  resources/gcxtras_sta_topic/resource.tf
  data-sources/gcxtras_sta_program/data-source.tf
  data-sources/gcxtras_sta_topic/data-source.tf
templates/
  index.md.tmpl              # custom provider page template
docs/                        # generated output (commit this)
```

## Release Process

Releases are published to the [Terraform Registry](https://registry.terraform.io/providers/gsyspslab/gcxtras) via GitHub Actions.

### Steps to release a new version:

```bash
# 1. Make your changes and commit
git add .
git commit -m "Description of changes"
git push origin main

# 2. Tag with a semver version
git tag v0.1.1
git push origin v0.1.1
```

Pushing the tag triggers the `.github/workflows/release.yml` workflow which:
1. Builds binaries for linux/darwin/windows (amd64 + arm64)
2. Signs the SHA256SUMS file with the GPG key
3. Creates a GitHub Release with all artifacts
4. The Terraform Registry webhook detects the release and publishes it

### GitHub Secrets Required

| Secret | Description |
|--------|-------------|
| `GPG_PRIVATE_KEY` | ASCII-armored GPG private key (`gpg --armor --export-secret-keys gsyspslab`) |

### GPG Key Management

The release artifacts are signed with a GPG key. The public key must be registered in:
- **Terraform Registry** — User Settings → Signing Keys
- **GitHub** — Account Settings → SSH and GPG keys (for verified tag signatures)

```bash
# Export public key (for registry/GitHub)
gpg --armor --export gsyspslab

# Export private key (for GitHub Actions secret)
gpg --armor --export-secret-keys gsyspslab
```

### GoReleaser

The `.goreleaser.yml` config handles cross-compilation and release artifact creation. It also includes the `terraform-registry-manifest.json` file in the release, which tells the registry the provider uses protocol version 5.0.

## Resource: gcxtras_sta_program

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The name of the program |
| `description` | string | no | The description of the program |
| `published` | bool | no | Whether the program is published (default: false) |
| `topic_ids` | list(string) | no | List of topic IDs associated with the program |
| `tags` | list(string) | no | List of tags associated with the program |

### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The program ID |
| `default_program` | bool | Whether this is the default program |

### Import

```bash
terraform import gcxtras_sta_program.example <program-id>
```

## Resource: gcxtras_sta_topic

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The name of the topic |
| `description` | string | no | The description of the topic |
| `dialect` | string | yes | The dialect (e.g. en-US, en-GB, de-DE). Forces replacement if changed. |
| `strictness` | string | no | Phrase strictness level 1-99 (default: 72) |
| `participants` | string | no | Which participants to analyze: External, Internal, All (default: All) |
| `phrases` | list(object) | no | List of phrases to detect (see below) |
| `program_ids` | list(string) | no | List of program IDs to associate with |
| `tags` | list(string) | no | List of tags |

#### Phrase Object

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | yes | The phrase text to detect |
| `strictness` | string | no | Per-phrase strictness override (1-99) |

### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The topic ID |

### Import

```bash
terraform import gcxtras_sta_topic.example <topic-id>
```

## Resource: gcxtras_processautomation_scheduled_trigger

Manages a Genesys Cloud Process Automation Scheduled Trigger. These triggers invoke Architect workflows on a cron-based schedule. The official Genesys Cloud provider only supports event-based triggers — this resource fills that gap.

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The name of the trigger (max 162 characters) |
| `description` | string | no | The description (max 512 characters) |
| `enabled` | bool | no | Whether the trigger is enabled (default: true) |
| `target` | block | yes | The workflow target (see below) |
| `schedule` | block | yes | The cron-based schedule (see below) |

#### Target Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | yes | The ID of the Architect workflow to invoke |
| `type` | string | yes | Must be `Workflow` |

#### Schedule Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `minutes` | list(number) | yes | Minutes at which the trigger runs (0-59). Max 2 values per hour. |
| `hours` | list(number) | yes | Hours at which the trigger runs (0-23). Empty list means every hour. |
| `days_of_month` | list(number) | no | Days of the month (1-31). Omit for every day. |
| `months` | list(number) | no | Months (1-12). Omit for every month. |
| `days_of_week` | list(number) | no | Days of the week (1-7, Sunday=1). Omit for every day. |
| `timezone` | string | yes | Timezone (e.g. `Europe/London`, `America/New_York`, `UTC`) |

### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The scheduled trigger ID |

### Example Usage

```hcl
resource "gcxtras_processautomation_scheduled_trigger" "daily_report" {
  name        = "Daily Report Generator"
  description = "Runs every weekday at 9:00 AM London time"
  enabled     = true

  target {
    id   = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    type = "Workflow"
  }

  schedule {
    minutes      = [0]
    hours        = [9]
    days_of_week = [2, 3, 4, 5, 6]
    timezone     = "Europe/London"
  }
}
```

### Import

```bash
terraform import gcxtras_processautomation_scheduled_trigger.example <trigger-id>
```

## Data Source: gcxtras_sta_program

Looks up an existing Speech and Text Analytics Program by name.

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The exact name of the program to look up |

### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The program ID |
| `description` | string | The program description |
| `published` | bool | Whether the program is published |
| `topic_ids` | list(string) | Associated topic IDs |
| `tags` | list(string) | Associated tags |
| `default_program` | bool | Whether this is the default program |

## Data Source: gcxtras_sta_topic

Looks up an existing Speech and Text Analytics Topic by name and dialect.

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The exact name of the topic to look up |
| `dialect` | string | yes | The dialect (e.g. en-GB). Required to disambiguate topics with the same name. |
| `state` | string | no | Which version to retrieve: `latest` or `published` (default: `latest`) |

### Attributes

| Name | Type | Description |
|------|------|-------------|
| `id` | string | The topic ID |
| `description` | string | The topic description |
| `dialect` | string | The topic dialect |
| `strictness` | string | The topic strictness level |
| `participants` | string | Which participants are analyzed |
| `phrases` | list(object) | The configured phrases |
| `program_ids` | list(string) | Associated program IDs |
| `tags` | list(string) | Associated tags |

## Supported Regions

- `us-east-1` (US East / Virginia)
- `us-east-2` (US East / Ohio - FedRAMP)
- `us-west-2` (US West / Oregon)
- `ca-central-1` (Canada)
- `sa-east-1` (São Paulo)
- `eu-west-1` (Ireland)
- `eu-west-2` (London)
- `eu-central-1` (Frankfurt)
- `eu-central-2` (Zurich)
- `ap-south-1` (Mumbai)
- `ap-southeast-2` (Sydney)
- `ap-northeast-1` (Tokyo)
- `ap-northeast-2` (Seoul)
- `ap-northeast-3` (Osaka)
- `me-central-1` (UAE)
- `af-south-1` (Cape Town)
