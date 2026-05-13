package provider

import (
	"context"
	"fmt"

	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/client"
	ds_sta_program "github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/datasources/sta_program"
	ds_sta_topic "github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/datasources/sta_topic"
	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/resources/sta_program"
	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/resources/sta_topic"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// New returns the provider schema and resource map.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"oauthclient_id": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("GENESYSCLOUD_OAUTHCLIENT_ID", nil),
				Description: "OAuthClient ID found on the OAuth page of Admin UI. Can be set with the `GENESYSCLOUD_OAUTHCLIENT_ID` environment variable.",
			},
			"oauthclient_secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("GENESYSCLOUD_OAUTHCLIENT_SECRET", nil),
				Description: "OAuthClient secret found on the OAuth page of Admin UI. Can be set with the `GENESYSCLOUD_OAUTHCLIENT_SECRET` environment variable.",
			},
			"aws_region": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("GENESYSCLOUD_REGION", nil),
				Description:  "AWS region where the Genesys Cloud org exists (e.g. us-east-1). Can be set with the `GENESYSCLOUD_REGION` environment variable.",
				ValidateFunc: validation.StringInSlice(client.AllowedRegions(), true),
			},
			"sdk_debug": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enables debug logging of all API requests and responses to `gcxtras_debug.log` in the working directory.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"gcxtras_sta_program": sta_program.Resource(),
			"gcxtras_sta_topic":   sta_topic.Resource(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"gcxtras_sta_topic":   ds_sta_topic.DataSource(),
			"gcxtras_sta_program": ds_sta_program.DataSource(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	clientID := d.Get("oauthclient_id").(string)
	clientSecret := d.Get("oauthclient_secret").(string)
	region := d.Get("aws_region").(string)
	debug := d.Get("sdk_debug").(bool)

	if clientID == "" {
		return nil, diag.Errorf("oauthclient_id must be set (or GENESYSCLOUD_OAUTHCLIENT_ID env var)")
	}
	if clientSecret == "" {
		return nil, diag.Errorf("oauthclient_secret must be set (or GENESYSCLOUD_OAUTHCLIENT_SECRET env var)")
	}
	if region == "" {
		return nil, diag.Errorf("aws_region must be set (or GENESYSCLOUD_REGION env var)")
	}

	c, err := client.NewGenesysCloudClient(clientID, clientSecret, region, debug)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to configure Genesys Cloud client: %w", err))
	}

	return c, nil
}
