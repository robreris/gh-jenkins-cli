package github

import (
  "context"
  "fmt"
  "os"
  "net/http"
  "errors"
  "time"

  "golang.org/x/oauth2"
  "github.com/google/go-github/v68/github"
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

func (c *Client) CreateRepo(orgName string, name string, templateRepo string, private bool) (*github.Repository, error) {

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

  err = c.AddOrUpdateREADME(orgName, name, readmeContent)
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

  err = c.AddBranchProtection(orgName, name)
  if err != nil {
    return nil, err
  }

  return createdRepo, nil

}

func (c *Client) GenerateRepoFromTemplate(templateOwner, templateRepo, newRepoName string, private bool) (*github.Repository, error) {
  ctx := context.Background()

  payload := map[string]interface{}{
    "name": newRepoName,
    "private": private,
    "owner":   templateOwner,
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

func (c *Client) WaitForMainBranch(orgName string, repoName string) error {
  ctx := context.Background()

  maxRetries := 10
  retryDelay := 2 * time.Second

  for i := 0; i < maxRetries; i++ {
    branch, _, err := c.client.Repositories.GetBranch(ctx, orgName, repoName, "main", 1)
    if err == nil && branch != nil {
      return nil
    }
    fmt.Printf("Waiting for main branch in repository '%s' to be ready (attempt %d/%d)...\n")
    time.Sleep(retryDelay)
  }
  
  return errors.New("wait for main branch timed out after multiple attempts")
}

func (c *Client) AddBranchProtection(orgName string, repoName string) error {
  ctx := context.Background()

  protectionRequest := &github.ProtectionRequest{
    RequiredStatusChecks: &github.RequiredStatusChecks{
      Strict: true,
      Contexts: &[]string{"ci/jenkins/build-status"},
    },
    EnforceAdmins: false,
    Restrictions: nil,
    RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
      DismissStaleReviews:		true,
      RequireCodeOwnerReviews:		true,
      RequiredApprovingReviewCount: 	1,
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

func (c *Client) AddOrUpdateREADME(orgName string, repoName string, content string) error {
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

	// Create a new blob for the README content
	blob := &github.Blob{
		Content:  github.String(content),
		Encoding: github.String("utf-8"),
	}
	blobResponse, _, err := c.client.Git.CreateBlob(ctx, orgName, repoName, blob)
	if err != nil {
		return fmt.Errorf("error creating blob: %v", err)
	}

	// Create a new tree including the README blob
	treeEntry := &github.TreeEntry{
		Path: github.String("README.md"),
		Mode: github.String("100644"),
		Type: github.String("blob"),
		SHA:  blobResponse.SHA,
	}
	tree, _, err := c.client.Git.CreateTree(ctx, orgName, repoName, baseTreeSHA, []*github.TreeEntry{treeEntry})
	if err != nil {
		return fmt.Errorf("error creating tree: %v", err)
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

	return nil
}

func (c *Client) CreateWebhook(orgName string, repoName string, webhookURL string) error {

  ctx := context.Background()

  webhookConfig := &github.HookConfig{
    URL: 	    github.String(webhookURL),
    ContentType:    github.String("json"),
  }

  hook := &github.Hook{
    Name: github.String("web"),
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

  maxRetries := 10
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
