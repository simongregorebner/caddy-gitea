package gitea

import (
	"fmt"

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

func (c *Client) Get(organization, repository, path string) ([]byte, error) {
	return c.GetRawFileOrLFS(organization, repository, path, c.branchName)
}

// Retrieve specific file from gitea server
func (client *Client) GetRawFileOrLFS(owner, repo, filepath, branch string) ([]byte, error) {

	client.logger.Info(fmt.Sprintf("Retrieve file - owner: %s repo: %s filepath: %s branch: %s", owner, repo, filepath, branch))

	content, _, err := client.gc.GetFile(owner, repo, branch, filepath)
	return content, err
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
