package sta_program

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ara/terraform-provider-genesyscloud-ara-extras/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const basePath = "/api/v2/speechandtextanalytics/programs"
const publishJobsPath = "/api/v2/speechandtextanalytics/programs/publishjobs"

// Resource returns the Terraform resource schema for speechandtextanalytics programs.
func Resource() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Genesys Cloud Speech and Text Analytics Program.",
		CreateContext: createProgram,
		ReadContext:   readProgram,
		UpdateContext: updateProgram,
		DeleteContext: deleteProgram,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the program.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The description of the program.",
			},
			"published": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the program should be published. When set to true, a publish job is triggered after create/update.",
			},
			"topic_ids": {
				Type:        schema.TypeList,
				Optional:    true,
				Computed:    true,
				Description: "List of topic IDs to associate with the program.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:        schema.TypeList,
				Optional:    true,
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

// programRequest represents the request body for creating/updating a program.
// The Genesys Cloud API accepts topicIds as a flat array of strings.
type programRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TopicIDs    []string `json:"topicIds,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// publishJobRequest represents the body for triggering a program publish job.
type publishJobRequest struct {
	ProgramIDs []string `json:"programIds"`
}

type programRef struct {
	ID string `json:"id"`
}

// programResponse represents the API response for a program.
type programResponse struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	Published      bool         `json:"published"`
	Topics         []programRef `json:"topics"`
	Tags           []string     `json:"tags"`
	DefaultProgram bool         `json:"defaultProgram"`
}

func buildProgramRequest(d *schema.ResourceData) programRequest {
	req := programRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	if v, ok := d.GetOk("topic_ids"); ok {
		topicList := v.([]interface{})
		topicIDs := make([]string, len(topicList))
		for i, id := range topicList {
			topicIDs[i] = id.(string)
		}
		req.TopicIDs = topicIDs
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

// publishProgram triggers a publish job for the given program ID.
func publishProgram(c *client.GenesysCloudClient, programID string) diag.Diagnostics {
	req := publishJobRequest{
		ProgramIDs: []string{programID},
	}

	body, statusCode, err := c.DoRequest(http.MethodPost, publishJobsPath, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error publishing STA program: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error publishing STA program (HTTP %d): %s", statusCode, string(body))
	}

	// Give the publish job a moment to process
	time.Sleep(2 * time.Second)

	return nil
}

func createProgram(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildProgramRequest(d)

	body, statusCode, err := c.DoRequest(http.MethodPost, basePath, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating STA program: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error creating STA program (HTTP %d): %s", statusCode, string(body))
	}

	var resp programResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing create response: %w", err))
	}

	d.SetId(resp.ID)

	// Publish if requested
	if d.Get("published").(bool) {
		if diags := publishProgram(c, resp.ID); diags.HasError() {
			return diags
		}
	}

	return readProgram(ctx, d, meta)
}

func readProgram(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodGet, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading STA program: %w", err))
	}

	if statusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error reading STA program (HTTP %d): %s", statusCode, string(body))
	}

	var resp programResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing read response: %w", err))
	}

	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("published", resp.Published)
	d.Set("default_program", resp.DefaultProgram)
	d.Set("tags", resp.Tags)

	// Set topic_ids from the API response topics array
	topicIDs := make([]string, len(resp.Topics))
	for i, t := range resp.Topics {
		topicIDs[i] = t.ID
	}
	d.Set("topic_ids", topicIDs)

	return nil
}

func updateProgram(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildProgramRequest(d)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodPut, path, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating STA program: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error updating STA program (HTTP %d): %s", statusCode, string(body))
	}

	// Publish if requested (re-publish on any update to pick up changes)
	if d.Get("published").(bool) {
		if diags := publishProgram(c, d.Id()); diags.HasError() {
			return diags
		}
	}

	return readProgram(ctx, d, meta)
}

func deleteProgram(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodDelete, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting STA program: %w", err))
	}

	if statusCode == http.StatusNotFound {
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error deleting STA program (HTTP %d): %s", statusCode, string(body))
	}

	d.SetId("")
	return nil
}
