package processautomation_scheduled_trigger

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

const basePath = "/api/v2/processautomation/scheduledtriggers"

// Resource returns the Terraform resource schema for process automation scheduled triggers.
func Resource() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages a Genesys Cloud Process Automation Scheduled Trigger.",
		CreateContext: createScheduledTrigger,
		ReadContext:   readScheduledTrigger,
		UpdateContext: updateScheduledTrigger,
		DeleteContext: deleteScheduledTrigger,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the scheduled trigger. Maximum 162 characters.",
				ValidateFunc: validation.StringLenBetween(1, 162),
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				Description:  "The description of the scheduled trigger. Maximum 512 characters.",
				ValidateFunc: validation.StringLenBetween(0, 512),
			},
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether the scheduled trigger is enabled.",
			},
			"target": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "The target workflow to invoke when the trigger fires.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The ID of the Architect workflow to invoke.",
						},
						"type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The type of target. Must be `Workflow`.",
							ValidateFunc: validation.StringInSlice([]string{
								"Workflow",
							}, false),
						},
					},
				},
			},
			"schedule": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "The cron-based schedule configuration for the trigger.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"minutes": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "The minutes at which the trigger runs (0-59). Maximum of 2 values per hour.",
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(0, 59),
							},
						},
						"hours": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "The hours at which the trigger runs (0-23). Use `[\"*\"]` for every hour. Specify as integers.",
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(0, 23),
							},
						},
						"days_of_month": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The days of the month on which the trigger runs (1-31). Leave empty or omit for every day.",
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(1, 31),
							},
						},
						"months": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The months in which the trigger runs (1-12). Leave empty or omit for every month.",
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(1, 12),
							},
						},
						"days_of_week": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "The days of the week on which the trigger runs (1-7, starting with Sunday=1). Leave empty or omit for every day.",
							Elem: &schema.Schema{
								Type:         schema.TypeInt,
								ValidateFunc: validation.IntBetween(1, 7),
							},
						},
						"timezone": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The timezone for the schedule (e.g. `Europe/London`, `America/New_York`, `UTC`).",
						},
					},
				},
			},
		},
	}
}

// --- Request/Response types ---

type targetRequest struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type scheduleRequest struct {
	Minutes     string `json:"minutes"`
	Hours       string `json:"hours"`
	DaysOfMonth string `json:"daysOfMonth"`
	Months      string `json:"months"`
	DaysOfWeek  string `json:"daysOfWeek"`
	Timezone    string `json:"timezone"`
}

type scheduledTriggerRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Enabled     bool            `json:"enabled"`
	Target      targetRequest   `json:"target"`
	Schedule    scheduleRequest `json:"schedule"`
}

type targetResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type scheduleResponse struct {
	Minutes     string `json:"minutes"`
	Hours       string `json:"hours"`
	DaysOfMonth string `json:"daysOfMonth"`
	Months      string `json:"months"`
	DaysOfWeek  string `json:"daysOfWeek"`
	Timezone    string `json:"timezone"`
}

type scheduledTriggerResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Enabled     bool             `json:"enabled"`
	Target      targetResponse   `json:"target"`
	Schedule    scheduleResponse `json:"schedule"`
}

// --- Helpers to convert between Terraform state and API format ---

// intListToCron converts a list of integers to a cron field string.
// An empty list means "*" (every value).
func intListToCron(vals []interface{}) string {
	if len(vals) == 0 {
		return "*"
	}
	result := ""
	for i, v := range vals {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%d", v.(int))
	}
	return result
}

// cronToIntList parses a cron field string back to a list of integers.
// "*" or "?" returns an empty list.
func cronToIntList(s string) []int {
	if s == "*" || s == "?" || s == "" {
		return []int{}
	}
	var result []int
	parts := splitCronField(s)
	for _, p := range parts {
		var v int
		fmt.Sscanf(p, "%d", &v)
		result = append(result, v)
	}
	return result
}

func splitCronField(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func buildRequest(d *schema.ResourceData) scheduledTriggerRequest {
	req := scheduledTriggerRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Enabled:     d.Get("enabled").(bool),
	}

	// Target
	targetList := d.Get("target").([]interface{})
	if len(targetList) > 0 {
		t := targetList[0].(map[string]interface{})
		req.Target = targetRequest{
			ID:   t["id"].(string),
			Type: t["type"].(string),
		}
	}

	// Schedule
	scheduleList := d.Get("schedule").([]interface{})
	if len(scheduleList) > 0 {
		s := scheduleList[0].(map[string]interface{})

		minutes := s["minutes"].([]interface{})
		hours := s["hours"].([]interface{})
		daysOfMonth := s["days_of_month"].([]interface{})
		months := s["months"].([]interface{})
		daysOfWeek := s["days_of_week"].([]interface{})

		// If daysOfMonth is specified and daysOfWeek is not (or vice versa),
		// the unspecified one should be "?" per the API docs
		domStr := intListToCron(daysOfMonth)
		dowStr := intListToCron(daysOfWeek)

		// Apply the mutual exclusion rule: if one is specified, the other is "?"
		if domStr != "*" && dowStr == "*" {
			dowStr = "?"
		} else if dowStr != "*" && domStr == "*" {
			domStr = "?"
		}

		req.Schedule = scheduleRequest{
			Minutes:     intListToCron(minutes),
			Hours:       intListToCron(hours),
			DaysOfMonth: domStr,
			Months:      intListToCron(months),
			DaysOfWeek:  dowStr,
			Timezone:    s["timezone"].(string),
		}
	}

	return req
}

func flattenResponse(resp scheduledTriggerResponse, d *schema.ResourceData) {
	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("enabled", resp.Enabled)

	d.Set("target", []interface{}{
		map[string]interface{}{
			"id":   resp.Target.ID,
			"type": resp.Target.Type,
		},
	})

	d.Set("schedule", []interface{}{
		map[string]interface{}{
			"minutes":       cronToIntList(resp.Schedule.Minutes),
			"hours":         cronToIntList(resp.Schedule.Hours),
			"days_of_month": cronToIntList(resp.Schedule.DaysOfMonth),
			"months":        cronToIntList(resp.Schedule.Months),
			"days_of_week":  cronToIntList(resp.Schedule.DaysOfWeek),
			"timezone":      resp.Schedule.Timezone,
		},
	})
}

// --- CRUD operations ---

func createScheduledTrigger(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildRequest(d)

	body, statusCode, err := c.DoRequest(http.MethodPost, basePath, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error creating scheduled trigger: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error creating scheduled trigger (HTTP %d): %s", statusCode, string(body))
	}

	var resp scheduledTriggerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing create response: %w", err))
	}

	d.SetId(resp.ID)

	return readScheduledTrigger(ctx, d, meta)
}

func readScheduledTrigger(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodGet, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading scheduled trigger: %w", err))
	}

	if statusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error reading scheduled trigger (HTTP %d): %s", statusCode, string(body))
	}

	var resp scheduledTriggerResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("error parsing read response: %w", err))
	}

	flattenResponse(resp, d)

	return nil
}

func updateScheduledTrigger(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	req := buildRequest(d)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodPut, path, req)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error updating scheduled trigger: %w", err))
	}
	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error updating scheduled trigger (HTTP %d): %s", statusCode, string(body))
	}

	return readScheduledTrigger(ctx, d, meta)
}

func deleteScheduledTrigger(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client.GenesysCloudClient)

	path := fmt.Sprintf("%s/%s", basePath, d.Id())

	body, statusCode, err := c.DoRequest(http.MethodDelete, path, nil)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error deleting scheduled trigger: %w", err))
	}

	if statusCode == http.StatusNotFound {
		return nil
	}

	if statusCode < 200 || statusCode >= 300 {
		return diag.Errorf("error deleting scheduled trigger (HTTP %d): %s", statusCode, string(body))
	}

	d.SetId("")
	return nil
}
