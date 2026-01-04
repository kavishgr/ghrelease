package github

import (
	"context"
	"fmt"
	"github.com/shurcooL/githubv4"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
		MaxIdleConnsPerHost: 10,
	},
}

// isValidURL checks if the provided string is a valid HTTP/HTTPS GitHub URL.
func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") &&
		u.Host != "" &&
		strings.Contains(u.Host, "github.com")
}

// fixUrl converts a GitHub repository reference into a GitHub API URL for releases.
// Accepts either a full URL (https://github.com/owner/repo) or short form (owner/repo).
// Returns the API URL and the repository path.
func fixUrl(githubUrl string) (string, string) {
	apiDomain := "https://api.github.com/repos"
	apiDomainSuffix := "/releases/latest"
	u, _ := url.Parse(githubUrl)
	fortag := fmt.Sprintf("%s", u.Path)

	if isValidURL(githubUrl) {
		result := fmt.Sprintf("%s%s%s", apiDomain, u.Path, apiDomainSuffix)
		return result, fortag
	}

	result := fmt.Sprintf("%s/%s%s", apiDomain, githubUrl, apiDomainSuffix)
	return result, fortag
}

// split extracts the owner and repository name from "owner/repo" format.
// Returns empty strings if the format is invalid.
func split(ownerNrepo string) (string, string) {
	ownerNrepo = strings.TrimPrefix(ownerNrepo, "/")
	parts := strings.Split(ownerNrepo, "/")

	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}

// getTagByName retrieves release information for the most recent tag of a repository.
// Used as a fallback when a repository doesn't have a "latest" release.
func (c *Client) getTagByName(ownerNrepo string) ([]byte, error) {
	// GitHub API token

	owner, name := split(ownerNrepo) // owner and name of the repo
	if owner == "" || name == "" {
		return nil, fmt.Errorf("invalid repository format: %s", ownerNrepo)
	}

	// Create an OAuth2 token source
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.token},
	)

	// Create an HTTP client with the token source
	oauthClient := oauth2.NewClient(context.Background(), src)

	// Create a new GitHub GraphQL client
	gqlClient := githubv4.NewClient(oauthClient)

	// Define the GraphQL query
	var query struct {
		Repository struct {
			Refs struct {
				Edges []struct {
					Node struct {
						Name string
					}
				}
			} `graphql:"refs(refPrefix: $refPrefix, first: $first, orderBy: $orderBy)"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	// Set the query variables
	variables := map[string]interface{}{
		"owner":     githubv4.String(owner),
		"name":      githubv4.String(name),
		"refPrefix": githubv4.String("refs/tags/"),
		"first":     githubv4.Int(1),
		"orderBy": githubv4.RefOrder{
			Field:     githubv4.RefOrderFieldTagCommitDate,
			Direction: githubv4.OrderDirectionDesc,
		},
	}

	// Execute the GraphQL query
	err := gqlClient.Query(context.Background(), &query, variables)
	if err != nil {
		return nil, fmt.Errorf("GraphQL query failed: %w", err)
	}

	// Access the query result
	if len(query.Repository.Refs.Edges) == 0 {
		return nil, fmt.Errorf("no tags found for %s", ownerNrepo)
	}
	tagname := query.Repository.Refs.Edges[0].Node.Name

	tagUrl := fmt.Sprintf("https://api.github.com/repos%s/releases/tags/%s", ownerNrepo, tagname)
	// fmt.Println(tagUrl)
	// fmt.Println("TAGURL:", tagUrl)

	req := c.craftRequest(tagUrl)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching tag release: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading tag response: %w", err)
	}
	return body, nil
}

// FetchGithubReleaseUrl fetches download URLs for release assets that match
// the OS/architecture regex pattern. Reads repository URLs from urlsChan and
// outputs matching asset URLs to stdout.
func (c *Client) FetchGithubReleaseURLs(ctx context.Context, urlsChan chan string, job *sync.WaitGroup) {
	defer job.Done()

	for u := range urlsChan {
		select {
		case <-ctx.Done():
			return
		default:
			c.fetchAndFilter(ctx, u)
		}
	}
}

func (c *Client) fetchAndFilter(ctx context.Context, u string) {
	github_releases := make(map[string]struct{})
	githubUrl, ownerNrepo := fixUrl(u) // fix url and return valid api url
	req := c.craftRequest(githubUrl)
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() == nil {
			log.Printf("Failed to fetch %s: %v", u, err)
		}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response for %s: %v", u, err)
	}

	// if the message is "Not Found"
	// release/asset section is EMPTY or is using tags instead of latest release
	message := gjson.Get(fmt.Sprintf("%s", body), "message")
	if message.Str == "Not Found" {
		// fetch assets for most recent tag
		body, err = c.getTagByName(ownerNrepo)
		if err != nil {
			body = []byte{}
		}
	}

	// fetch all the browser_download_url keys which contains the asset urls
	// results := gjson.Get(fmt.Sprintf("%s", body), "assets.#.browser_download_url")
	results := gjson.Get(string(body), "assets.#.browser_download_url")

	results.ForEach(func(key, value gjson.Result) bool {
		asset_url := value.String()
		isMatch, _ := c.regex.MatchString(asset_url)

		if isMatch {
			github_releases[asset_url] = struct{}{}
		}

		return true // keep iterating in case there are multiple urls that match
	})

	if len(github_releases) == 0 {
		fmt.Println("N/A:", u)
	} else {
		for release := range github_releases {
			// fmt.Println("URL found:", k)
			fmt.Println(release)
		}
	}
}
