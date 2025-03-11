package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/github"
	"github.com/spf13/cobra"
)

var deleteRepoCmd = &cobra.Command{
	Use:   "delete-repo",
	Short: "Delete an existing repo in FortinetCloudCSE org",
	Run: func(cmd *cobra.Command, args []string) {
		client := github.NewClient()
		err := client.DeleteRepo("FortinetCloudCSE", repoName)
		if err != nil {
			fmt.Println("Error deleting repository:", err)
			return
		}
		fmt.Printf("Repository '%s' deleted successfully.", repoName)
	},
}

func init() {
	rootCmd.AddCommand(deleteRepoCmd)
	deleteRepoCmd.Flags().StringVarP(&repoName, "name", "n", "", "Name of the repo")
	deleteRepoCmd.MarkFlagRequired("name")
}
