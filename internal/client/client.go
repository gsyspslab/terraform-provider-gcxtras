package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// regionMapping maps AWS region identifiers to Genesys Cloud API base domains.
// Reference: https://help.mypurecloud.com/articles/aws-regions-for-genesys-cloud-deployment
var regionMapping = map[string]string{
	"us-east-1":      "mypurecloud.com",
	"us-east-2":      "use2.us-gov-pure.cloud",
	"us-west-2":      "usw2.pure.cloud",
	"ca-central-1":   "cac1.pure.cloud",
	"sa-east-1":      "sae1.pure.cloud",
	"eu-west-1":      "mypurecloud.ie",
	"eu-west-2":      "euw2.pure.cloud",
	"eu-central-1":   "mypurecloud.de",
	"eu-central-2":   "euc2.pure.cloud",
	"ap-south-1":     "aps1.pure.cloud",
	"ap-southeast-2": "mypurecloud.com.au",
	"ap-northeast-1": "mypurecloud.jp",
	"ap-northeast-2": "apne2.pure.cloud",
	"ap-northeast-3": "apne3.pure.cloud",
	"me-central-1":   "mec1.pure.cloud",
	"af-south-1":     "afs1.pure.cloud",
}

// AllowedRegions returns the list of valid AWS region strings for the provider.
func AllowedRegions() []string {
	regions := make([]string, 0, len(regionMapping))
	for r := range regionMapping {
		regions = append(regions, r)
	}
	return regions
}

// GenesysCloudClient is an authenticated HTTP client for the Genesys Cloud Platform API.
type GenesysCloudClient struct {
	httpClient   *http.Client
	baseDomain   string
	accessToken  string
	tokenExpiry  time.Time
	clientID     string
	clientSecret string
	debug        bool
	debugLogger  *log.Logger
	mu           sync.Mutex
}

// NewGenesysCloudClient creates a new client and authenticates using OAuth2 client credentials.
func NewGenesysCloudClient(clientID, clientSecret, awsRegion string, debug bool) (*GenesysCloudClient, error) {
	domain, ok := regionMapping[strings.ToLower(awsRegion)]
	if !ok {
		return nil, fmt.Errorf("unsupported AWS region: %s", awsRegion)
	}

	c := &GenesysCloudClient{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseDomain:   domain,
		clientID:     clientID,
		clientSecret: clientSecret,
		debug:        debug,
	}

	if debug {
		f, err := os.OpenFile("gcxtras_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open gcxtras_debug.log: %w", err)
		}
		c.debugLogger = log.New(f, "", log.LstdFlags)
		c.debugLog("Initializing client for region %s (domain: %s)", awsRegion, domain)
	}

	if err := c.authenticate(); err != nil {
		return nil, err
	}

	return c, nil
}

// debugLog writes a message to the debug log file if debug is enabled.
func (c *GenesysCloudClient) debugLog(format string, args ...interface{}) {
	if c.debug && c.debugLogger != nil {
		c.debugLogger.Printf("[GCXTRAS] "+format, args...)
	}
}

// authenticate obtains an OAuth2 access token using client credentials grant.
func (c *GenesysCloudClient) authenticate() error {
	tokenURL := fmt.Sprintf("https://login.%s/oauth/token", c.baseDomain)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	c.debugLog("AUTH >>> POST %s", tokenURL)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.clientID, c.clientSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing auth request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.debugLog("AUTH <<< %d BODY: %s", resp.StatusCode, string(body))
		return fmt.Errorf("authentication failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	c.debugLog("AUTH <<< success, token expires in %d seconds", tokenResp.ExpiresIn)

	return nil
}

// ensureAuthenticated re-authenticates if the token is expired or about to expire.
func (c *GenesysCloudClient) ensureAuthenticated() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-authenticate if token expires within 60 seconds
	if time.Now().Add(60 * time.Second).After(c.tokenExpiry) {
		c.debugLog("Token expired or expiring soon, re-authenticating")
		return c.authenticate()
	}
	return nil
}

// apiURL constructs the full API URL for a given path.
func (c *GenesysCloudClient) apiURL(path string) string {
	return fmt.Sprintf("https://api.%s%s", c.baseDomain, path)
}

// DoRequest performs an authenticated HTTP request against the Genesys Cloud API.
func (c *GenesysCloudClient) DoRequest(method, path string, body interface{}) ([]byte, int, error) {
	if err := c.ensureAuthenticated(); err != nil {
		return nil, 0, fmt.Errorf("authentication error: %w", err)
	}

	var reqBody io.Reader
	var reqBodyBytes []byte
	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(reqBodyBytes)
	}

	fullURL := c.apiURL(path)

	c.debugLog(">>> %s %s", method, fullURL)
	if reqBodyBytes != nil {
		c.debugLog(">>> BODY: %s", string(reqBodyBytes))
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	c.debugLog("<<< %d %s %s", resp.StatusCode, method, path)
	if c.debug {
		respStr := string(respBody)
		if len(respStr) > 4000 {
			respStr = respStr[:4000] + "\n... [truncated]"
		}
		c.debugLog("<<< BODY: %s", respStr)
	}

	return respBody, resp.StatusCode, nil
}
