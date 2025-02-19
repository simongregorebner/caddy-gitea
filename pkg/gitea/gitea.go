package gitea

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"

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

	content, err := c.getRawFileOrLFS(organization, repository, path, c.branchName)
	if err != nil {
		return nil, err
	}

	return &openFile{
		content: content,
		name:    path,
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

// Check if the repo has a specific branch
func (c *Client) RepoBranchExists(organization, repository, branch string) bool {
	branchInfo, _, err := c.gc.GetRepoBranch(organization, repository, branch)
	if err != nil {
		return false
	}
	return branchInfo.Name == branch
}

// Check if a repo has a specific topic assigned
func (client *Client) TopicExists(organization, repository, topicName string) bool {
	topics, _, err := client.gc.ListRepoTopics(organization, repository, gclient.ListRepoTopicsOptions{})
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
