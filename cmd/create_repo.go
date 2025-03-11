package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/github"
	"github.com/spf13/cobra"
)

var (
	orgName  string
	repoName string
	private  bool
)

var createRepoCmd = &cobra.Command{
	Use:   "create-repo",
	Short: "Create a new repo in FortinetCloudCSE org",
	Run: func(cmd *cobra.Command, args []string) {
		client := github.NewClient()
		repo, err := client.CreateRepo("FortinetCloudCSE", repoName, "UserRepo", private)
		if err != nil {
			fmt.Println("Error creating repository:", err)
			return
		}
		fmt.Printf("Repository '%s' created successfully at %s\n", repo.GetName(), repo.GetHTMLURL())
	},
}

func init() {
	rootCmd.AddCommand(createRepoCmd)
	createRepoCmd.Flags().StringVarP(&repoName, "name", "n", "", "Name of the repo")
	createRepoCmd.Flags().BoolVarP(&private, "private", "p", false, "Make repository private")
	createRepoCmd.MarkFlagRequired("name")
}
