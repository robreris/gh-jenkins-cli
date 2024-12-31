package github

import (
  "context"
  "fmt"
  "os"
  "net/http"

  "golang.org/x/oauth2"
  "github.com/google/go-github/v50/github"
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

func (c *Client) CreateRepo(orgName string, name string, private bool) (*github.Repository, error) {
  ctx := context.Background()
  repo := &github.Repository{
    Name: github.String(name),
    Private: github.Bool(private),
  }
  createdRepo, _, err := c.client.Repositories.Create(ctx, orgName, repo)
  if err != nil {
    return nil, err
  }

  err = c.CreateEmptyMainBranch(orgName, name)
  if err != nil {
    return nil, fmt.Errorf("error creating main branch: %v", err)
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

  err = c.CreateInitialCommitWithREADME(orgName, name, readmeContent)
  if err != nil {
    return nil, fmt.Errorf("error updating README.md: %v", err)
  }

  err = c.AddBranchProtection(orgName, name)
  if err != nil {
    return nil, err
  }

  return createdRepo, nil

}

func (c *Client) CreateEmptyMainBranch(orgName string, repoName string) error {
  ctx := context.Background()

  zeroSHA := "0000000000000000000000000000000000000000"
  ref := &github.Reference{
    Ref:	github.String("refs/heads/main"),
    Object:	&github.GitObject{SHA: github.String(zeroSHA)},
  }

  _, _, err := c.client.Git.CreateRef(ctx, orgName, repoName, ref)
  if err != nil {
    return fmt.Errorf("error creating main branch: %v", err)
  }

  return nil
}

func (c *Client) AddBranchProtection(orgName string, repoName string) error {
  ctx := context.Background()

  protectionRequest := &github.ProtectionRequest{
    RequiredStatusChecks: &github.RequiredStatusChecks{
      Strict: true,
      Contexts: []string{"ci/jenkins/build-status"},
    },
    EnforceAdmins: false,
    Restrictions: nil,
    RequiredPullRequestReviews: &github.PullRequestReviewsEnforcementRequest{
      DismissStaleReviews:		true,
      RequireCodeOwnerReviews:		true,
      RequiredApprovingReviewCount: 	1,
    },
  }

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

func (c *Client) CreateInitialCommitWithREADME(orgName string, repoName string, readmeContent string) error {
	ctx := context.Background()

	// Create a new blob for the README content
	blob := &github.Blob{
		Content:  github.String(readmeContent),
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
	tree, _, err := c.client.Git.CreateTree(ctx, orgName, repoName, "", []*github.TreeEntry{treeEntry})
	if err != nil {
		return fmt.Errorf("error creating tree: %v", err)
	}

	// Create a new commit pointing to the new tree
	commitMessage := "initial commit with README.md and GitHub Pages URL"
	firstCommit := &github.Commit{
		Message: github.String(commitMessage),
		Tree:    tree,
	}
	commitResponse, _, err := c.client.Git.CreateCommit(ctx, orgName, repoName, firstCommit)
	if err != nil {
		return fmt.Errorf("error creating commit: %v", err)
	}

        ref := &github.Reference{
          Ref: github.String("/refs/heads/main"),
          Object: &github.GitObject{SHA: commitResponse.SHA},
        }
        _, _, err = c.client.Git.CreateRef(ctx, orgName, repoName, ref)
        if err != nil {
          return fmt.Errorf("error creating branch reference: %v", err)
        }

	return nil
}
