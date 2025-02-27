package cmd

import (
  "fmt"
  "github.com/spf13/cobra"
  "github.com/robreris/gh-jenkins-cli/github"
)

var deleteRepoCmd = &cobra.Command{
  Use:	"delete-repo",
  Short: "Delete an existing repo in FortinetCloudCSE org",
  Run: func(cmd *cobra.Command, args []string) {
    client := github.NewClient()
    err := client.DeleteRepo("FortinetCloudCSE", repoName)
    if err != nil {
      fmt.Println("Error deleting repository:", err)
      return
    }
    fmt.Println("Repository '%s' deleted successfully.", repoName)
  },
}

func init() {
  rootCmd.AddCommand(deleteRepoCmd)
  deleteRepoCmd.Flags().StringVarP(&repoName, "name", "n", "", "Name of the repo")
  deleteRepoCmd.MarkFlagRequired("name")
}
