package github

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/kavishgr/ghrelease/utils"
)

type Client struct {
	token      string
	httpClient *http.Client
	regex      *regexp2.Regexp
}

func NewClient(token, os, arch string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	regexPattern := utils.SetRegex(os, arch)
	regex, err := regexp2.Compile(regexPattern, 0)
	if err != nil {
		return nil, fmt.Errorf("compiling regex pattern: %w", err)

	}

	return &Client{
		token: token,
		httpClient: &http.Client{
			// Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  true,
				MaxIdleConnsPerHost: 10,
			},
		},
		regex: regex,
	}, nil
}

func (c *Client) craftRequest(url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// This should never fail for valid URLs, but handle it anyway
		panic(fmt.Sprintf("creating request: %v", err))
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", c.token))
	req.Header.Add("User-Agent", "ghrelease-cli")
	return req
}

func (c *Client) ValidateToken() error {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Add("Authorization", "token "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach GitHub API: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		return nil
	case 401:
		return fmt.Errorf("invalid or expired token (401 Unauthorized)")
	case 403:
		return fmt.Errorf("token is valid but lacks required permissions (403 Forbidden)")
	default:
		return fmt.Errorf("unexpected response from GitHub API: %d %s", resp.StatusCode, resp.Status)
	}
}
