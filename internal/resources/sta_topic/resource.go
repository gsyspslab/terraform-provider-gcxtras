package sta_topic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const basePath = "/api/v2/speechandtextanalytics/topics"

// Resource returns the Terraform resource schema for speechandtextanalytics topics.
func Resource() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Genesys Cloud Speech and Text Analytics Topic.",
		CreateContext: createTopic,
		ReadContext:   readTopic,
		UpdateContext: updateTopic,
		DeleteContext: deleteTopic,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the topic.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The description of the topic.",
			},
			"strictness": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "72",
				Description:  "The phrase strictness level (1-99). Controls how closely spoken words must match the defined phrases. Higher values require closer matches.",
				ValidateFunc: validation.StringInSlice([]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27", "28", "29", "30", "31", "32", "33", "34", "35", "36", "37", "38", "39", "40", "41", "42", "43", "44", "45", "46", "47", "48", "49", "50", "51", "52", "53", "54", "55", "56", "57", "58", "59", "60", "61", "62", "63", "64", "65", "66", "67", "68", "69", "70", "71", "72", "73", "74", "75", "76", "77", "78", "79", "80", "81", "82", "83", "84", "85", "86", "87", "88", "89", "90", "91", "92", "93", "94", "95", "96", "97", "98", "99"}, false),
			},
			"participants": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "All",
				Description:  "Which participants to analyze. Valid values: External, Internal, All.",
				ValidateFunc: validation.StringInSlice([]string{"External", "Internal", "All"}, false),
			},
			"dialect": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The dialect for the topic (e.g. en-US, en-GB, de-DE, fr-FR, etc.).",
			},
			"phrases": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of phrases to detect for this topic.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"text": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The phrase text to detect.",
						},
						"strictness": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Override strictness for this specific phrase (1-99). If not set, uses the topic-level strictness.",
						},
					},
				},
			},
			"program_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of program IDs this topic is associated with.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of tags associated with the topic.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// API request/response types

type phraseRequest struct {
	Text       string `json:"text"`
	Strictness string `json:"strictness,omitempty"`
}

type programRef struct {
	ID string `json:"id"`
}

type topicRequest struct {
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Strictness   string          `json:"strictness,omitempty"`
	Participants string          `json:"participants,omitempty"`
	Dialect      string          `json:"dialect"`
	Phrases      []phraseRequest `json:"phrases,omitempty"`
	Programs     []programRef    `json:"programs,omitempty"`
	Tags         []string        `json:"tags,omitempty"`
}

type phraseResponse struct {
	Text       string      `json:"text"`
	Strictness json.Number `json:"strictness"`
}

type TopicResponse struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Strictness   json.Number      `json:"strictness"`
	Participants string           `json:"participants"`
	Dialect      string           `json:"dialect"`
	Phrases      []phraseResponse `json:"phrases"`
	Programs     []programRef     `json:"programs"`
	Tags         []string         `json:"tags"`
}

func buildTopicRequest(d *schema.ResourceData) topicRequest {
	req := topicRequest{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		Strictness:   d.Get("strictness").(string),
		Participants: d.Get("participants").(string),
		Dialect:      d.Get("dialect").(string),
	}

	if v, ok := d.GetOk("phrases"); ok {
		phraseList := v.([]interface{})
		phrases := make([]phraseRequest, len(phraseList))
		for i, p := range phraseList {
			phraseMap := p.(map[string]interface{})
			phrases[i] = phraseRequest{
				Text: phraseMap["text"].(string),
			}
			if s, ok := phraseMap["strictness"]; ok && s.(string) != "" {
				phrases[i].Strictness = s.(string)
			}
		}
		req.Phrases = phrases
	}

	if v, ok := d.GetOk("program_ids"); ok {
		programIDs := v.([]interface{})
		programs := make([]programRef, len(programIDs))
		for i, id := range programIDs {
			programs[i] = programRef{ID: id.(string)}
		}
		req.Programs = programs
	}

	if v, ok := d.GetOk("tags"); ok {
		tagList := v.([]interface{})
		tags := make([]string, len(tagList))
		for i, t := range tagList {
			tags[i] = t.(string)
		}
		req.Tags = tags
	}

	return req
}

func createTopic(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildTopicRequest(d)

	body, statusCode, err := c.DoRequest(http.MethodPost, basePath, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating STA topic: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error creating STA topic (HTTP %d): %s", statusCode, string(body))
	}

	var resp TopicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing create response: %w", err))
	}

	d.SetId(resp.ID)

	return readTopic(ctx, d, meta)
}

func readTopic(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodGet, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading STA topic: %w", err))
	}

	if statusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error reading STA topic (HTTP %d): %s", statusCode, string(body))
	}

	var resp TopicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing read response: %w", err))
	}

	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("strictness", resp.Strictness.String())
	d.Set("participants", resp.Participants)
	d.Set("dialect", resp.Dialect)

	phrases := make([]map[string]interface{}, len(resp.Phrases))
	for i, p := range resp.Phrases {
		phrases[i] = map[string]interface{}{
			"text":       p.Text,
			"strictness": p.Strictness.String(),
		}
	}
	d.Set("phrases", phrases)

	programIDs := make([]string, len(resp.Programs))
	for i, p := range resp.Programs {
		programIDs[i] = p.ID
	}
	d.Set("program_ids", programIDs)
	d.Set("tags", resp.Tags)

	return nil
}

func updateTopic(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildTopicRequest(d)
	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodPut, path, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating STA topic: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error updating STA topic (HTTP %d): %s", statusCode, string(body))
	}

	return readTopic(ctx, d, meta)
}

func deleteTopic(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodDelete, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting STA topic: %w", err))
	}

	if statusCode == http.StatusNotFound {
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error deleting STA topic (HTTP %d): %s", statusCode, string(body))
	}

	d.SetId("")
	return nil
}
