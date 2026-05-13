# terraform-provider-gcxtras

A companion Terraform provider for Genesys Cloud that manages resources not covered by the official provider.

Published by [gsyspslab](https://gsyspslab.com).

## Supported Resources

| Resource | Description |
|----------|-------------|
| `gcxtras_sta_program` | Speech and Text Analytics Program (full CRUD) |
| `gcxtras_sta_topic` | Speech and Text Analytics Topic (full CRUD) |

## Supported Data Sources

| Data Source | Description |
|-------------|-------------|
| `gcxtras_sta_program` | Look up an existing STA Program by name |
| `gcxtras_sta_topic` | Look up an existing STA Topic by name |

## Authentication

This provider uses the same OAuth configuration as the official Genesys Cloud Terraform Provider.

### Provider Configuration

```hcl
provider "gcxtras" {
  oauthclient_id     = "your-client-id"
  oauthclient_secret = "your-client-secret"
  aws_region         = "eu-west-1"
}
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GENESYSCLOUD_OAUTHCLIENT_ID` | OAuth Client ID |
| `GENESYSCLOUD_OAUTHCLIENT_SECRET` | OAuth Client Secret |
| `GENESYSCLOUD_REGION` | AWS Region (e.g. `eu-west-1`) |

## Building

```bash
make build    # Build the binary
make install  # Install to local Terraform plugin directory
```

## Usage

```hcl
terraform {
  required_providers {
    gcxtras = {
      source  = "gsyspslab/gcxtras"
      version = "0.1.0"
    }
  }
}

resource "gcxtras_sta_program" "example" {
  name        = "My Analytics Program"
  description = "Managed by Terraform"
  published   = false
  tags        = ["terraform"]
}

resource "gcxtras_sta_topic" "cancellation" {
  name         = "Account Cancellation"
  description  = "Detects cancellation intent"
  dialect      = "en-US"
  strictness   = "72"
  participants = "External"

  phrases {
    text = "I want to cancel my account"
  }
  phrases {
    text = "close out my account"
  }

  program_ids = [gcxtras_sta_program.example.id]
  tags        = ["churn"]
}

# Reference existing resources
data "gcxtras_sta_program" "default" {
  name = "Default Program"
}

data "gcxtras_sta_topic" "billing" {
  name = "Billing Inquiry"
}
```

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

Looks up an existing Speech and Text Analytics Topic by name.

### Arguments

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | The exact name of the topic to look up |

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
