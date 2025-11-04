package cmd

import (
	"fmt"
	"log"

	"github.com/robreris/gh-jenkins-cli/github"
	"github.com/robreris/gh-jenkins-cli/jenkins"

	"github.com/spf13/cobra"
)

var (
	jenkinsXMLPath string
	collabNames    []string
)

var createProjectCmd = &cobra.Command{
	Use:   "create-project",
	Short: "Create a new project in FortinetCloudCSE org consisting of a GitHub repo and associated Jenkins pipeline",
	Run: func(cmd *cobra.Command, args []string) {

		jClient := jenkins.NewAPIClient()
		if err := jClient.CreateJob(repoName, jenkinsXMLPath); err != nil {
			log.Fatal("Error creating Jenkins job: ", err)
		}
		fmt.Printf("Jenkins job %s successfully created.", repoName)

		ghClient := github.NewClient()
		repo, err := ghClient.CreateRepo("FortinetCloudCSE", repoName, "UserRepo", private, true)
		if err != nil {
			fmt.Println("Error creating repository:", err)
			return
		}
		_ = ghClient.AddCollaborators("FortinetCloudCSE", repoName, collabNames, "push")
		if err != nil {
			fmt.Println("Error adding collaborators:", err)
			return
		}

		fmt.Printf("Repository '%s' created successfully at %s\n", repo.GetName(), repo.GetHTMLURL())
	},
}

func init() {
	rootCmd.AddCommand(createProjectCmd)
	createProjectCmd.Flags().StringVarP(&repoName, "project-name", "p", "", "Name of the project/repo")
	createProjectCmd.Flags().StringVarP(&jenkinsXMLPath, "jenkins-xml", "j", "jenkins/template-config.xml", "Path to Jenkins config XML file.")
	createProjectCmd.Flags().StringSliceVarP(&collabNames, "collab-names", "u", []string{}, "GitHub usernames to add as collaborators. Enter each username separated by a comma. e.g. -c user1,user2,user3.")
	createProjectCmd.Flags().BoolVarP(&private, "private", "r", false, "Make repository private")
	createProjectCmd.MarkFlagRequired("project-name")
}
