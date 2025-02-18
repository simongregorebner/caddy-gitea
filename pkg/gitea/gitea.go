package gitea

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	gclient "code.gitea.io/sdk/gitea"
	"go.uber.org/zap"
)

type Client struct {
	serverURL  string
	token      string
	branchName string
	topicName  string
	gc         *gclient.Client
	logger     *zap.Logger
}

func NewClient(logger *zap.Logger, serverURL, token, requiredBranchName, requiredTopicName string) (*Client, error) {

	giteaClient, err := gclient.NewClient(serverURL, gclient.SetToken(token), gclient.SetGiteaVersion(""))
	if err != nil {
		return nil, err
	}

	return &Client{
		serverURL:  serverURL,
		token:      token,
		gc:         giteaClient,
		branchName: requiredBranchName,
		topicName:  requiredTopicName,
		logger:     logger,
	}, nil
}

func (c *Client) Get(organization, repository, path string) (fs.File, error) {

	// TODO need to cache this and return a good error message
	// if !c.hasRepoBranch(organization, repository, c.branchName) || !c.hasTopic(organization, repository, c.topicName) {
	// 	c.logger.Error("Branch or topic does not exist")
	// 	return nil, fs.ErrNotExist
	// }

	c.logger.Info(fmt.Sprintf("After path split: owner: %s repo: %s filepath: %s", organization, repository, path))

	content, err := c.getRawFileOrLFS(organization, repository, path, c.branchName)
	if err != nil {
		return nil, err
	}

	return &openFile{
		content: content,
		name:    path,
	}, nil
}

func (c *Client) Open(name string) (fs.File, error) {

	c.logger.Info(fmt.Sprintf("Retrieve path: %s", name))
	owner, repo, filepath := splitName(name)

	c.logger.Info(fmt.Sprintf("After path split: owner: %s repo: %s filepath: %s", owner, repo, filepath))

	// if repo is empty they want to have the gitea-pages repo
	if repo == "" {
		repo = c.branchName
		filepath = "index.html"
	}

	// we need to check if the repo exists (and allows access)
	limited, allowall := c.allowsPages(owner, repo)
	if !limited && !allowall {
		// if we're checking the gitea-pages and it doesn't exist, return 404
		if repo == c.branchName && !c.hasRepoBranch(owner, repo, c.branchName) {
			return nil, fs.ErrNotExist
		}

		// the repo didn't exist but maybe it's a filepath in the gitea-pages repo
		// so we need to check if the gitea-pages repo exists
		if filepath == "" {
			filepath = repo
		} else {
			filepath = repo + "/" + filepath
		}
		repo = c.branchName

		limited, allowall = c.allowsPages(owner, repo)
		if !limited && !allowall || !c.hasRepoBranch(owner, repo, c.branchName) {
			return nil, fs.ErrNotExist
		}
	}

	// If filepath is empty they want to have the index.html
	if filepath == "" {
		filepath = "index.html"
	}

	content, err := c.getRawFileOrLFS(owner, repo, filepath, c.branchName)
	if err != nil {
		return nil, err
	}

	return &openFile{
		content: content,
		name:    filepath,
	}, nil
}

// Retrieve specific file from gitea server
func (client *Client) getRawFileOrLFS(owner, repo, filepath, branch string) ([]byte, error) {

	client.logger.Info(fmt.Sprintf("Retrieve file - owner: %s repo: %s filepath: %s branch: %s", owner, repo, filepath, branch))

	// Assemble URL to query the gitea API
	// TODO: make pr for go-sdk
	// gitea sdk doesn't support "media" type for lfs/non-lfs
	giteaURL, err := url.JoinPath(client.serverURL+"/api/v1/repos/", owner, repo, "media", url.QueryEscape(filepath))
	if err != nil {
		return nil, err
	}

	// Add ref (branch) identifier
	giteaURL += "?ref=" + url.QueryEscape(branch)

	// Assemble request
	request, err := http.NewRequest(http.MethodGet, giteaURL, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "token "+client.token)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, fs.ErrNotExist
	case http.StatusOK:
	default:
		return nil, fmt.Errorf("unexpected status code '%d'", response.StatusCode)
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	return content, nil
}

// Read list of assigned topics for a given repo
func (client *Client) repoTopics(owner, repo string) ([]string, error) {
	topics, _, err := client.gc.ListRepoTopics(owner, repo, gclient.ListRepoTopicsOptions{})
	return topics, err
}

// Check if the repo has a specific branc
func (c *Client) hasRepoBranch(owner, repo, branchName string) bool {
	branchInfo, _, err := c.gc.GetRepoBranch(owner, repo, branchName)
	if err != nil {
		return false
	}

	return branchInfo.Name == branchName
}

func (client *Client) hasTopic(owner, repo, topicName string) bool {
	topics, _, err := client.gc.ListRepoTopics(owner, repo, gclient.ListRepoTopicsOptions{})
	if err != nil {
		return false
	}

	for _, topic := range topics {
		if topic == topicName {
			return true
		}
	}
	return false
}

// Check if repo has giteapages label attached
func (c *Client) allowsPages(owner, repo string) (bool, bool) {
	topics, err := c.repoTopics(owner, repo)
	if err != nil {
		return false, false
	}

	for _, topic := range topics {
		if topic == c.branchName {
			return true, false
		}
	}

	return false, false
}

func splitName(name string) (string, string, string) {
	parts := strings.Split(name, "/")

	// parts contains: ["owner", "repo", "filepath"]
	switch len(parts) {
	case 1:
		return parts[0], "", ""
	case 2:
		return parts[0], parts[1], ""
	default:
		return parts[0], parts[1], strings.Join(parts[2:], "/")
	}
}
