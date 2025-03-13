package github

import (
	"context"
	"errors"
	"fmt"
        "log"
        "strings"
	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"time"
)

type Client struct {
	client *github.Client
}

func NewClient() *Client {
	ctx := context.Background()
	var tc *http.Client

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable not set.")
		os.Exit(1)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc = oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)
	return &Client{client: ghClient}
}

func (c *Client) CreateRepo(orgName string, name string, templateRepo string, private bool, pipelineOpt ...string) (*github.Repository, error) {

	createdRepo, err := c.GenerateRepoFromTemplate(orgName, templateRepo, name, private)
	if err != nil {
		return nil, err
	}

	err = c.WaitForMainBranch(orgName, name)
	if err != nil {
		return nil, fmt.Errorf("main branch in repository '%s' not ready: %v", name, err)
	}

	pagesURL, err := c.EnableGitHubPages(orgName, name)
	if err != nil {
		return nil, fmt.Errorf("error enabling GitHub Pages: %v", err)
	}
	fmt.Printf("GitHub Pages URL: %s\n", pagesURL)

	readmeContent := fmt.Sprintf(`
# %s

To view the workshop, please go here: [GitHub Pages Link](%s)

---

For more information on creating these workshops, visit [FortinetCloudCSE User Repo](https://fortinetcloudcse.github.io/UserRepo/)
`, name, pagesURL)

	err = c.UpdateRepoFiles(orgName, name, readmeContent, pipelineOpt[0], pipelineOpt[1])
	if err != nil {
		return nil, fmt.Errorf("error updating README.md: %v", err)
	}

	webhookURL := "https://jenkins.fortinetcloudcse.com:8443/github-webhook/"
	err = c.CreateWebhook(orgName, name, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	statusCheck := "ci/jenkins/build-status"
	err = c.WaitForStatusCheck(orgName, name, "main", statusCheck)
	if err != nil {
		return nil, fmt.Errorf("error waiting for status check '%s', %v", statusCheck, err)
	}

	/*
	  err = c.AddBranchProtection(orgName, name)
	  if err != nil {
	    return nil, err
	  }
	*/

	return createdRepo, nil

}

func (c *Client) GenerateRepoFromTemplate(templateOwner, templateRepo, newRepoName string, private bool) (*github.Repository, error) {
	ctx := context.Background()

	payload := map[string]interface{}{
		"name":        newRepoName,
		"private":     private,
		"owner":       templateOwner,
		"description": "This repo was generated from " + templateRepo,
	}

	apiPath := fmt.Sprintf("repos/%s/%s/generate", templateOwner, templateRepo)
	req, err := c.client.NewRequest("POST", apiPath, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating request to generate repo from template %v", err)
	}

	var repo github.Repository
	_, err = c.client.Do(ctx, req, &repo)
	if err != nil {
		return nil, fmt.Errorf("error generating repository from template %v", err)
	}

	return &repo, nil
}

func (c *Client) DeleteRepo(templateOwner string, repoName string) error {
	ctx := context.Background()

	apiPath := fmt.Sprintf("repos/%s/%s", templateOwner, repoName)
	req, err := c.client.NewRequest("DELETE", apiPath, nil)
	if err != nil {
		return fmt.Errorf("error deleting repo %v", err)
	}

	_, err = c.client.Do(ctx, req, nil)
	if err != nil {
		return fmt.Errorf("error deleting repo %v", err)
	}

	return nil
}

func (c *Client) WaitForMainBranch(orgName string, repoName string) error {
	ctx := context.Background()

	maxRetries := 10
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		branch, _, err := c.client.Repositories.GetBranch(ctx, orgName, repoName, "main", 1)
		if err == nil && branch != nil {
			return nil
		}
		fmt.Printf("Waiting for main branch in repository '%s' to be ready (attempt %d/%d)...\n", repoName, i, maxRetries)
		time.Sleep(retryDelay)
	}

	return errors.New("wait for main branch timed out after multiple attempts")
}

func (c *Client) AddBranchProtection(orgName string, repoName string) error {
	ctx := context.Background()

	protectionRequest := &github.ProtectionRequest{
		RequiredStatusChecks: &github.RequiredStatusChecks{
			Strict:   true,
			Contexts: &[]string{"ci/jenkins/build-status"},
		},
		EnforceAdmins: false,
		Restrictions:  nil,
		RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
			DismissStaleReviews:          true,
			RequireCodeOwnerReviews:      true,
			RequiredApprovingReviewCount: 1,
		},
	}

	fmt.Printf("Branch protection payload: %+v\n", protectionRequest)

	_, _, err := c.client.Repositories.UpdateBranchProtection(ctx, orgName, repoName, "main", protectionRequest)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) EnableGitHubPages(orgName string, repoName string) (string, error) {
	ctx := context.Background()
	opts := &github.Pages{
		Source: &github.PagesSource{
			Branch: github.String("main"),
			Path:   github.String("/docs"),
		},
	}
	_, _, err := c.client.Repositories.EnablePages(ctx, orgName, repoName, opts)
	if err != nil {
		return "", fmt.Errorf("error enabling GitHub Pages: %v", err)
	}

	apiURL := fmt.Sprintf("repos/%s/%s/pages", orgName, repoName)
	req, err := c.client.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request to fetch pages URL: %v", err)
	}

	var pagesResponse struct {
		HTMLURL string `json:"html_url"`
	}

	_, err = c.client.Do(ctx, req, &pagesResponse)
	if err != nil {
		return "", fmt.Errorf("error fetching GitHub Pages URL: %v", err)
	}

	return pagesResponse.HTMLURL, nil
}

func (c *Client) UpdateRepoFiles(orgName string, repoName string, readmeContent string, pipelineOpts ...string) error {
	ctx := context.Background()

	fdsAppId := pipelineOpts[0]
	jenkinsUpdate := pipelineOpts[1]

	// Get the latest commit and tree SHA from the main branch
	branch, _, err := c.client.Repositories.GetBranch(ctx, orgName, repoName, "main", 1)
	if err != nil {
		return fmt.Errorf("error fetching branch information: %v", err)
	}
	currentCommitSHA := branch.GetCommit().GetSHA()

	commit, _, err := c.client.Git.GetCommit(ctx, orgName, repoName, currentCommitSHA)
	if err != nil {
		return fmt.Errorf("error fetching commit information: %v", err)
	}
	baseTreeSHA := commit.GetTree().GetSHA()

	files := map[string]string{
		"README.md": readmeContent,
	}

	// Update FortiDevSec Application ID
	if fdsAppId != "" {
		fdsConfigData, err := os.ReadFile("github/fdevsec.yaml")
                if err != nil {
                      return fmt.Errorf("error finding fdevsec.yaml in UpdateRepoFiles function call")
                }
		fdsPlaceholder := "<insert app id here>"
		updatedFDSConfig := strings.ReplaceAll(string(fdsConfigData), fdsPlaceholder, fdsAppId)
		files["fdevsec.yaml"] = updatedFDSConfig
	}

	// Enable Jenkins
	if jenkinsUpdate == "yes" {
		jenkinsConfigData, err := os.ReadFile("github/Jenkinsfile")
                if err != nil {
                      return fmt.Errorf("error finding jenkinsfile in UpdateRepoFiles function call")
                }
		jenkinsPlaceholder := "when { expression { false } }"
		jenkinsPlaceholderReplace := "when { expression { true } }"
		updatedJenkinsConfig := strings.ReplaceAll(string(jenkinsConfigData), jenkinsPlaceholder, jenkinsPlaceholderReplace)
		files["Jenkinsfile"] = updatedJenkinsConfig
	}

	var treeEntries []*github.TreeEntry

	for path, content := range files {
		blob, _, err := c.client.Git.CreateBlob(ctx, orgName, repoName, &github.Blob{
			Content:  github.String(content),
			Encoding: github.String("utf-8"),
		})
		if err != nil {
			return fmt.Errorf("failed to create blob for %s: %v", path, err)
		}
		treeEntries = append(treeEntries, &github.TreeEntry{
			Path: github.String(path),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  blob.SHA,
		})
	}

	tree, _, err := c.client.Git.CreateTree(ctx, orgName, repoName, baseTreeSHA, treeEntries)
	if err != nil {
		return fmt.Errorf("failed to create tree: %v", err)
	}

	// Create a new commit pointing to the new tree
	commitMessage := "Update README.md with GitHub Pages link"
	newCommit := &github.Commit{
		Message: github.String(commitMessage),
		Tree:    tree,
		Parents: []*github.Commit{{SHA: github.String(currentCommitSHA)}},
	}
	commitOptions := &github.CreateCommitOptions{
		Signer: nil,
	}
	newCommitResponse, _, err := c.client.Git.CreateCommit(ctx, orgName, repoName, newCommit, commitOptions)
	if err != nil {
		return fmt.Errorf("error creating commit: %v", err)
	}

	// Update the main branch reference to point to the new commit
	_, _, err = c.client.Git.UpdateRef(ctx, orgName, repoName, &github.Reference{
		Ref: github.String("refs/heads/main"),
		Object: &github.GitObject{
			SHA: newCommitResponse.SHA,
		},
	}, false)
	if err != nil {
		return fmt.Errorf("error updating branch reference: %v", err)
	}

	fmt.Println("Updated README, fdevsec.yaml, and Jenkinsfile")
	return nil
}

func (c *Client) CreateWebhook(orgName string, repoName string, webhookURL string) error {

	ctx := context.Background()

	webhookConfig := &github.HookConfig{
		URL:         github.String(webhookURL),
		ContentType: github.String("json"),
	}

	hook := &github.Hook{
		Name:   github.String("web"),
		Active: github.Bool(true),
		Events: []string{"push"},
		Config: webhookConfig,
	}

	_, _, err := c.client.Repositories.CreateHook(ctx, orgName, repoName, hook)
	if err != nil {
		return fmt.Errorf("error creating webhook for repository '%s': %v", repoName, err)
	}

	fmt.Printf("Webhook created successfully for repository '%s' with URL '%s'\n", repoName, webhookURL)
	return nil

}

func (c *Client) WaitForStatusCheck(orgName, repoName, branch, statusCheck string) error {
	ctx := context.Background()

	maxRetries := 60
	retryDelay := 2 * time.Second

	for i := 0; i < maxRetries; i++ {

		statuses, _, err := c.client.Repositories.ListStatuses(ctx, orgName, repoName, branch, nil)
		if err != nil {
			return fmt.Errorf("error fetching status checks for branch '%s': %v", branch, err)
		}

		for _, status := range statuses {
			if status.GetContext() == statusCheck {
				return nil // status check available
			}
		}

		fmt.Printf("Waiting for status check '%s' to be reported (attempt %d/%d)...\n", statusCheck, i+1, maxRetries)
		time.Sleep(retryDelay)
	}

	return fmt.Errorf("status check '%s' not reported after multiple attempts", statusCheck)
}


func (c *Client) AddCollaborators(owner, repo string, collaborators []string, permission string) error {
	ctx := context.Background()

        collabOpts := &github.RepositoryAddCollaboratorOptions{
               Permission: permission,
        }

	for _, collaborator := range collaborators {
		_, _, err := c.client.Repositories.AddCollaborator(ctx, owner, repo, collaborator, collabOpts)
		if err != nil {
			log.Printf("Failed to add collaborator %s: %v", collaborator, err)
			return err
		}
		fmt.Printf("Successfully added %s to %s/%s with %s permission\n", collaborator, owner, repo, permission)
	}
	return nil
}
