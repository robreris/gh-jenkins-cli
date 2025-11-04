package cmd

import (
	"fmt"
	"log"

	"github.com/robreris/gh-jenkins-cli/github"
	"github.com/robreris/gh-jenkins-cli/jenkins"
	"github.com/spf13/cobra"
)

var jenkinsJob string

var deleteProjectCmd = &cobra.Command{
	Use:   "delete-project",
	Short: "Delete a GitHub repo and its associated Jenkins job",
	Run: func(cmd *cobra.Command, args []string) {
		if repoName == "" {
			log.Fatal("Project name is required.")
		}

		jobName := jenkinsJob
		if jobName == "" {
			jobName = repoName
		}

		jClient := jenkins.NewAPIClient()
		if err := jClient.DeleteJob(jobName); err != nil {
			log.Fatalf("Error deleting Jenkins job '%s': %v", jobName, err)
		}
		fmt.Printf("Jenkins job '%s' deleted successfully.\n", jobName)

		ghClient := github.NewClient()
		if err := ghClient.DeleteRepo("FortinetCloudCSE", repoName); err != nil {
			log.Fatalf("Error deleting repository '%s': %v", repoName, err)
		}
		fmt.Printf("Repository '%s' deleted successfully.\n", repoName)
	},
}

func init() {
	rootCmd.AddCommand(deleteProjectCmd)

	deleteProjectCmd.Flags().StringVarP(&repoName, "project-name", "p", "", "Name of the project/repo to delete.")
	deleteProjectCmd.Flags().StringVarP(&jenkinsJob, "jenkins-job", "j", "", "Name of the Jenkins job to delete. Defaults to the project name.")
	deleteProjectCmd.MarkFlagRequired("project-name")
}
