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

func (c *Client) CreateRepo(name string, private bool) (*github.Repository, error) {
  ctx := context.Background()
  repo := &github.Repository{
    Name: github.String(name),
    Private: github.Bool(private),
  }
  createdRepo, _, err := c.client.Repositories.Create(ctx, "", repo)
  return createdRepo, err
}
