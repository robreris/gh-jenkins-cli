package github

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

type Client struct {
	client     *github.Client
	JenkinsUrl string
}

func NewClient() *Client {
	ctx := context.Background()
	var tc *http.Client

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("GITHUB_TOKEN environment variable not set.")
		os.Exit(1)
	}

	jenkinsUrl := os.Getenv("JENKINS_URL")
	if jenkinsUrl == "" {
		fmt.Println("Warning: JENKINS_URL environment variable not set.")
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc = oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)
	return &Client{
		client:     ghClient,
		JenkinsUrl: jenkinsUrl,
	}
}

func (c *Client) CreateRepo(orgName string, name string, templateRepo string, private bool, enablePipeline bool) (*github.Repository, error) {

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

        //Need UpdateRepo in both blocks since order of execution is important here
	if enablePipeline == true {
		//webhookURL := "https://jenkins.fortinetcloudcse.com:8443/github-webhook/"
		webhookURL := c.JenkinsUrl + "/github-webhook/"
		err = c.CreateWebhook(orgName, name, webhookURL)
		if err != nil {
			return nil, fmt.Errorf("error creating webhook: %v", err)
		}

	        err = c.UpdateRepoFiles(orgName, name, readmeContent, enablePipeline)
	        if err != nil {
		        return nil, fmt.Errorf("error updating repo files: %v", err)
	        }

		statusCheck := "ci/jenkins/build-status"
		err = c.WaitForStatusCheck(orgName, name, "main", statusCheck)
		if err != nil {
			return nil, fmt.Errorf("error waiting for status check '%s', %v", statusCheck, err)
		}
	} else {
	        err = c.UpdateRepoFiles(orgName, name, readmeContent, enablePipeline)
	        if err != nil {
		        return nil, fmt.Errorf("error updating repo files: %v", err)
	        }
        }

	err = c.AddBranchProtection(orgName, name)
	if err != nil {
		return nil, err
	}

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
			DismissStaleReviews:          false,
			RequireCodeOwnerReviews:      false,
			RequiredApprovingReviewCount: 0,
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
		BuildType: github.String("workflow"),
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

	fmt.Println("in enable pages func, req:", req)

	return pagesResponse.HTMLURL, nil
}

func (c *Client) UpdateRepoFiles(orgName string, repoName string, readmeContent string, enablePipeline bool) error {
	ctx := context.Background()

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

	// Enable Jenkins
	if enablePipeline {
		jenkinsConfigData, err := os.ReadFile("github/Jenkinsfile")
		if err != nil {
			return fmt.Errorf("error reading Jenkinsfile in UpdateRepoFiles function call")
		}
		content := string(jenkinsConfigData)

		// Define which "when" expressions to replace: 1-based index (e.g., []int{2} to replace only the second one)
		indicesToReplace := map[int]bool{
			2: true, // only replace the 2nd instance
		}

		// Match all occurrences of the when-expression block
		re := regexp.MustCompile(`when\s*\{\s*expression\s*\{\s*false\s*\}\s*\}`)
		matches := re.FindAllStringIndex(content, -1)

		if len(matches) == 0 {
			fmt.Println("No 'when { expression { false } }' blocks found.")
			files["Jenkinsfile"] = content
			return nil
		}

		// Replace only the specified indices
		var updatedContent string
		lastIndex := 0
		for i, match := range matches {
			start, end := match[0], match[1]
			updatedContent += content[lastIndex:start]
			if indicesToReplace[i+1] { // 1-based index
				updatedContent += "when { expression { true } }"
			} else {
				updatedContent += content[start:end]
			}
			lastIndex = end
		}
		updatedContent += content[lastIndex:] // add the rest

		files["Jenkinsfile"] = updatedContent
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

	fmt.Println("README (and Jenkinsfile if create-project invoked) updated successfully.")
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
