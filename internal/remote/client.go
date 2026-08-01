package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/pratham-vishk/stratabench/internal/agentapi"
	"github.com/pratham-vishk/stratabench/internal/agentauth"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(host string) *Client {
	base := strings.TrimSpace(host)
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "http://" + base
	}
	base = strings.TrimRight(base, "/")
	return &Client{
		BaseURL: base,
		HTTPClient: &http.Client{Timeout: 2 * time.Hour},
	}
}

func (c *Client) Health(ctx context.Context) (*agentapi.HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/health", nil)
	if err != nil {
		return nil, err
	}
	agentauth.SetAuthHeader(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health %s: %s", resp.Status, string(body))
	}
	var out agentapi.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Run(ctx context.Context, p *profile.Profile, target string, mock, skipValidate, checkHardware bool, cacheBytes int64) (*schema.RunResult, error) {
	yamlBytes, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(agentapi.RunRequest{
		ProfileYAML:   string(yamlBytes),
		Target:        target,
		Mock:          mock,
		SkipValidate:  skipValidate,
		CheckHardware: checkHardware,
		CacheBytes:    cacheBytes,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/run", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	agentauth.SetAuthHeader(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("run %s: %s", resp.Status, string(raw))
	}
	var out agentapi.RunResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if !out.OK || out.Run == nil {
		if out.Error != "" {
			return nil, fmt.Errorf("%s", out.Error)
		}
		return nil, fmt.Errorf("agent run failed")
	}
	return out.Run, nil
}

func ParseHosts(csv string) []string {
	parts := strings.Split(csv, ",")
	var hosts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			hosts = append(hosts, p)
		}
	}
	return hosts
}
