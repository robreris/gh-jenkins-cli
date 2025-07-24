package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/robreris/gh-jenkins-cli/github"
	"github.com/spf13/cobra"
)

// Flags
var (
	collaborators string // Comma-separated list of collaborators
	permission    string
)

// addCollabCmd represents the add_collab command
var addCollabCmd = &cobra.Command{
	Use:   "add-collab",
	Short: "Add collaborators to a GitHub repoNamesitory",
	Long: `This command allows you to add one or more collaborators to a specified GitHub repoNamesitory.
	
Example usage:
  mycli add-collab --org myorg --repo-name myrepoName --collaborators user1,user2 --permission push
	`,
	Run: func(cmd *cobra.Command, args []string) {
		// Convert comma-separated collaborators string into a slice
		collabList := strings.Split(collaborators, ",")

		// Initialize the GitHub client using your existing NewClient function
		client := github.NewClient()

		// Call the AddCollaborators function
		err := client.AddCollaborators(orgName, repoName, collabList, permission)
		if err != nil {
			log.Fatalf("Error adding collaborators: %v", err)
		}

		fmt.Println("All collaborators added successfully.")
	},
}

func init() {
	rootCmd.AddCommand(addCollabCmd)
	addCollabCmd.Flags().StringVarP(&orgName, "org", "o", "FortinetCloudCSE", "GitHub repository organization (required)")
	addCollabCmd.Flags().StringVarP(&repoName, "repo-name", "r", "", "GitHub repository name (required)")
	addCollabCmd.Flags().StringVarP(&collaborators, "collaborators", "c", "", "Comma-separated list of collaborators to add (required)")
	addCollabCmd.Flags().StringVarP(&permission, "permission", "p", "push", "Permission level (pull, push, admin, maintain, triage)")
	addCollabCmd.MarkFlagRequired("repo-name")
	addCollabCmd.MarkFlagRequired("collaborators")
}
