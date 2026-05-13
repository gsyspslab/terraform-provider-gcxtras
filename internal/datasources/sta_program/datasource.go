package sta_program

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const basePath = "/api/v2/speechandtextanalytics/programs"

// DataSource returns the Terraform data source schema for looking up an existing STA program.
func DataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up an existing Genesys Cloud Speech and Text Analytics Program by name.",
		ReadContext: readProgramByName,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the program to look up.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the program.",
			},
			"published": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the program is published.",
			},
			"topic_ids": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of topic IDs associated with the program.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of tags associated with the program.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"default_program": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether this is the default program.",
			},
		},
	}
}

type topicRef struct {
	ID string `json:"id"`
}

type programResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Published      bool       `json:"published"`
	Topics         []topicRef `json:"topics"`
	Tags           []string   `json:"tags"`
	DefaultProgram bool       `json:"defaultProgram"`
}

type programListResponse struct {
	Entities   []programResponse `json:"entities"`
	PageSize   int               `json:"pageSize"`
	PageNumber int               `json:"pageNumber"`
	Total      int               `json:"total"`
}

func readProgramByName(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)
	name := d.Get("name").(string)

	pageNumber := 1
	pageSize := 100

	for {
		path := fmt.Sprintf("%s?pageSize=%d&pageNumber=%d&name=%s", basePath, pageSize, pageNumber, url.QueryEscape(name))

		body, statusCode, err := c.DoRequest(http.MethodGet, path, nil)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error listing STA programs: %w", err))
		}
		if statusCode < 200 || statusCode >= 300 {
			return diag.Errorf("error listing STA programs (HTTP %d): %s", statusCode, string(body))
		}

		var resp programListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("error parsing program list response: %w", err))
		}

		for _, program := range resp.Entities {
			if program.Name == name {
				d.SetId(program.ID)
				d.Set("name", program.Name)
				d.Set("description", program.Description)
				d.Set("published", program.Published)
				d.Set("default_program", program.DefaultProgram)

				topicIDs := make([]string, len(program.Topics))
				for i, t := range program.Topics {
					topicIDs[i] = t.ID
				}
				d.Set("topic_ids", topicIDs)
				d.Set("tags", program.Tags)

				return nil
			}
		}

		if pageNumber*pageSize >= resp.Total {
			break
		}
		pageNumber++
	}

	return diag.Errorf("STA program with name %q not found", name)
}
