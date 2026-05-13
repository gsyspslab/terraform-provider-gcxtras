package sta_topic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/client"
	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/resources/sta_topic"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const basePath = "/api/v2/speechandtextanalytics/topics"

// DataSource returns the Terraform data source schema for looking up an existing STA topic.
func DataSource() *schema.Resource {
	return &schema.Resource{
		Description: "Looks up an existing Genesys Cloud Speech and Text Analytics Topic by name and dialect.",
		ReadContext: readTopicByName,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the topic to look up.",
			},
			"dialect": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The dialect of the topic (e.g. en-US, en-GB, de-DE). Required because multiple topics can share the same name across dialects.",
			},
			"state": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "latest",
				Description:  "Which version of the topic to retrieve. Valid values: `latest` (most recent version), `published` (only the published version). Defaults to `latest`.",
				ValidateFunc: validation.StringInSlice([]string{"latest", "published"}, false),
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the topic.",
			},
			"strictness": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The phrase strictness level.",
			},
			"participants": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Which participants are analyzed (External, Internal, All).",
			},
			"phrases": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of phrases configured for this topic.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"text": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The phrase text.",
						},
						"strictness": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The phrase-level strictness override.",
						},
					},
				},
			},
			"program_ids": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of program IDs this topic is associated with.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of tags associated with the topic.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// topicListResponse represents the paginated list response from the topics API.
type topicListResponse struct {
	Entities   []sta_topic.TopicResponse `json:"entities"`
	PageSize   int                       `json:"pageSize"`
	PageNumber int                       `json:"pageNumber"`
	Total      int                       `json:"total"`
}

func readTopicByName(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)
	name := d.Get("name").(string)
	dialect := d.Get("dialect").(string)
	state := d.Get("state").(string)

	pageNumber := 1
	pageSize := 100

	for {
		path := fmt.Sprintf("%s?pageSize=%d&pageNumber=%d&name=%s&state=%s",
			basePath, pageSize, pageNumber, url.QueryEscape(name), url.QueryEscape(state))

		body, statusCode, err := c.DoRequest(http.MethodGet, path, nil)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error listing STA topics: %w", err))
		}
		if statusCode < 200 || statusCode >= 300 {
			return diag.Errorf("error listing STA topics (HTTP %d): %s", statusCode, string(body))
		}

		var resp topicListResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("error parsing topic list response: %w", err))
		}

		// Search for exact name + dialect match
		for _, topic := range resp.Entities {
			if topic.Name == name && strings.EqualFold(topic.Dialect, dialect) {
				d.SetId(topic.ID)
				d.Set("name", topic.Name)
				d.Set("description", topic.Description)
				d.Set("strictness", topic.Strictness.String())
				d.Set("participants", topic.Participants)
				d.Set("dialect", topic.Dialect)

				phrases := make([]map[string]interface{}, len(topic.Phrases))
				for i, p := range topic.Phrases {
					phrases[i] = map[string]interface{}{
						"text":       p.Text,
						"strictness": p.Strictness.String(),
					}
				}
				d.Set("phrases", phrases)

				programIDs := make([]string, len(topic.Programs))
				for i, p := range topic.Programs {
					programIDs[i] = p.ID
				}
				d.Set("program_ids", programIDs)
				d.Set("tags", topic.Tags)

				return nil
			}
		}

		// Check if we've exhausted all pages
		if pageNumber*pageSize >= resp.Total {
			break
		}
		pageNumber++
	}

	return diag.Errorf("STA topic with name %q, dialect %q, state %q not found", name, dialect, state)
}
