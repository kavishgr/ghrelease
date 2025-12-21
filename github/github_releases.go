package github

import (
	"context"
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/k0kubun/go-ansi"
	"github.com/kavishgr/ghrelease/utils"
	"github.com/schollz/progressbar/v3"
	"github.com/shurcooL/githubv4"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

/*
- Check if a given string is a valid URL
got the regex from chatgpt
*/
func isValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^(https?|ftp)://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

/*
	 	Takes a GitHub URL as input and returns two strings.
	 	It aims to standardize the URL format
	 	to match the API endpoint for fetching the latest release
	 	of a GitHub repository.

		- It initializes apiDomain and apiDomainSuffix variables with fixed parts
		of the API URL.

		- It parses the input githubUrl using the url.Parse function
		to extract path information.

		- If the input URL is valid (using the isValidURL function),
		it constructs the API URL by combining apiDomain, the parsed path,
		and apiDomainSuffix.

		- If the input URL is not valid,
		it assumes the input is a GitHub repository name
		and constructs the API URL accordingly.

		- It then returns the standardized API URL
		and the extracted repository path.
*/
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

/*
- Creates an authenticated HTTP GET request for the GitHub API
by setting the required headers,
including the GitHub token and user agent
*/
func craftGithubReq(ghtoken, url string) *http.Request {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %s", ghtoken))
	// req.Header.Add("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Add("User-Agent", "getghrel-cli")
	return req
}

/*
  - Downloads and processes files
    concurrently from a list of URLs provided through the urlsChan.

  - It uses a GitHub token for authentication
    saves the downloaded files to a temporary directory
    and optionally extracts the files if specified.
*/
func DownloadRelease(urlsChan chan string, job *sync.WaitGroup, ghtoken, tempdir string, skipextraction bool) {

	defer job.Done()

	// anonymous func() to handle file download and processing
	// so that defer() gets called upon each iteration
	// instead of waiting for DownloadRelease() to return
	downloadAndProcessFile := func(u string) {
		// get the assetname of each url -> e.g bat.tar.gz
		file := path.Base(u)
		src := filepath.Join(tempdir, file)

		req := craftGithubReq(ghtoken, u)
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		f, _ := os.OpenFile(src, os.O_CREATE|os.O_WRONLY, 0644)
		defer f.Close()

		// bar := progressbar.DefaultBytes(
		// 	resp.ContentLength,
		// 	file,
		// )

		// io.Copy(io.MultiWriter(f, bar), resp.Body)

		bar := progressbar.NewOptions64(resp.ContentLength,
			progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionClearOnFinish(),
			progressbar.OptionSetElapsedTime(true),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionSetDescription(fmt.Sprintf("%s", file)),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))

		io.Copy(io.MultiWriter(f, bar), resp.Body)
		bar.Reset()
		bar.Finish()

		if skipextraction {
			fmt.Printf("Downloaded: %s\n", file)
			bar.Close()
			return
		}

		fmt.Printf("Downloaded and Extracted: %s\n", file)
		bar.Close()
		utils.Extractor(src, tempdir)
	}

	// iterate over urls sent by stdin
	for u := range urlsChan {
		downloadAndProcessFile(u)
	}
}

/*
- Takes a string in the format "owner/repo" as input
and returns two strings.

- It extracts the owner and repository names
from the input string by splitting it at the '/' character.

- If the input starts with a '/',
it removes it before performing the split.

- The function then returns the extracted owner
and repository names as separate strings.
*/
func split(ownerNrepo string) (string, string) {
	var str string
	if strings.HasPrefix(ownerNrepo, "/") {
		str = strings.TrimPrefix(ownerNrepo, "/")
	}
	parts := strings.Split(str, "/")
	return parts[0], parts[1]
}

/*
- Retrieves information about a GitHub repository's latest release tag
using the GitHub GraphQL API.
It takes a GitHub API token (`ghtoken`)
and a string in the format "owner/repo" (`ownerNrepo`)
representing the repository's owner and name.

- The function uses the `split` function to separate
the owner and repository names from the input string.
It then sets up an OAuth2 token source
and an HTTP client to create a GitHub GraphQL client.

- Next, the function defines a GraphQL query to fetch
the latest release tag for the given repository.
The query specifies the required fields,
such as the name of the tag and sorting based on commit date.

- After executing the GraphQL query,
the function extracts the latest tag name
from the query result.
It constructs the URL for the GitHub API
endpoint related to the retrieved tag.

- Using the `craftGithubReq` function,
it creates an HTTP GET request
with the provided GitHub token and tag URL.
The function then sends the request,
reads the response body,
and returns it as a byte slice containing
the information about the latest release tag.
*/
func getTagByName(ghtoken, ownerNrepo string) []byte {
	// GitHub API token

	var tagname string
	owner, name := split(ownerNrepo) // owner and name of the repo

	// Create an OAuth2 token source
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
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
		log.Fatal(err)
	}

	// Access the query result
	for _, edge := range query.Repository.Refs.Edges {
		// fmt.Println("Tag Name:", edge.Node.Name)
		tagname = edge.Node.Name
		// fmt.Println(tagname)
	}

	tagUrl := fmt.Sprintf("https://api.github.com/repos%s/releases/tags/%s", ownerNrepo, tagname)
	// fmt.Println(tagUrl)
	// fmt.Println("TAGURL:", tagUrl)

	req := craftGithubReq(ghtoken, tagUrl)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	return body
}

// verify if a map is empty or not
func mapIsEmpty(m map[string]int) bool {
	return len(m) == 0 // returns true if map is empty
}

/* - fetch the asset urls from latest release for each url or username/repo (used by -list)
   - the regex is used to find the required asset url for your os/arch
   - return/print found urls for each asset that matched
   - repos that do not have a release for the os and arch will be printed like so:
   - N/A: https://github.com/user/repo
*/

/*
- Fetches the download URLs for specific assets from GitHub releases.
It uses a regular expression (regex) to filter URLs
based on the target OS/architecture.
The function takes URLs from the urlsChan channel
and uses a provided GitHub API token (ghtoken)
to make API requests to fetch release information.

- The function starts by defining an inner function fetch responsible
for handling the URL processing.
Within this function, it prepares the API URL
using fixUrl and constructs an HTTP GET request
with the provided GitHub token using craftGithubReq.
It then sends the request and reads the response body
containing release information.

- The function checks if the response contains a "Not Found" message,
indicating that the repository may be using tags
instead of the latest release. In such cases,
it fetches the assets for the most recent tag
using the getTagByName function.

- Next, the function uses gjson to parse the response body
and extract the URLs of the assets.
It applies the provided regular expression
to filter the URLs based on the target OS/architecture.
URLs that match the regex are stored in the github_release map.

- If there are matching URLs, they are printed to the console.
If there are no matching URLs, "N/A" is printed to
indicate that no relevant assets were found.

- The main loop of the function continuously receives URLs
from urlsChan and processes them using the fetch function.
*/
func FetchGithubReleaseUrl(urlsChan chan string, job *sync.WaitGroup, regex, ghtoken string) {

	defer job.Done()
	// var github_release []string
	// github_release := make(map[string]int)

	fetch := func(u string) {
		github_release := make(map[string]int)
		// fmt.Println(u)
		// map to keep assets
		// sometimes there are multiple assets for same os/architecture
		// for e.g gnu and musl for linux
		re2 := regexp2.MustCompile(regex, 0) // regex for os/arch
		// fmt.Println("Regex: ", re)
		githubUrl, ownerNrepo := fixUrl(u) // fix url and return valid api url
		// fmt.Println(githubUrl)

		// HTTP client starts
		req := craftGithubReq(ghtoken, githubUrl) // craft request with token and valid api url
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		// HTTP client ends

		message := gjson.Get(fmt.Sprintf("%s", body), "message")
		// if the message is "Not Found"
		// release/asset section is EMPTY or is using tags instead of latest release
		if message.Str == "Not Found" {
			// fetch assets for most recent tag
			body = getTagByName(ghtoken, ownerNrepo)
		}

		// fetch all the browser_download_url keys which contains the asset urls
		// results := gjson.Get(fmt.Sprintf("%s", body), "assets.#.browser_download_url")
		results := gjson.Get(fmt.Sprintf("%s", body), "assets.#.browser_download_url")

		// fmt.Println("Matching values for URL:", u)
		results.ForEach(func(key, value gjson.Result) bool {
			// fmt.Println(key.String())
			asset_url := value.String()
			// fmt.Println(asset_url)
			isMatch, _ := re2.MatchString(asset_url)

			if isMatch == true {
				github_release[asset_url] = 1
			}

			return true // keep iterating in case there are multiple urls that match
		})

		if mapIsEmpty(github_release) {
			fmt.Println("N/A:", u)
		} else {
			for k, _ := range github_release {
				// fmt.Println("URL found:", k)
				fmt.Println(k)
			}
		}

	}

	for u := range urlsChan {
		fetch(u)
	}
}
