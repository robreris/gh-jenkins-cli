package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/jenkins"
	"github.com/spf13/cobra"
	"log"
)

var (
	jobName       string
	configXMLPath string
)

var createJobCmd = &cobra.Command{
	Use:   "create-job",
	Short: "Create a new Jenkins job",
	Run: func(cmd *cobra.Command, args []string) {
		client := jenkins.NewAPIClient()

		if jobName == "" || configXMLPath == "" {
			log.Fatal("Missing some flags.")
		}

		if err := client.CreateJob(jobName, configXMLPath); err != nil {
			log.Fatal("Error creating Jenkins job: ", err)
		}

		fmt.Printf("Jenkins job '%s' created successfully.\n", jobName)
	},
}

func init() {
	rootCmd.AddCommand(createJobCmd)
	createJobCmd.Flags().StringVarP(&jobName, "name", "n", "", "Name of Jenkins job.")
	createJobCmd.Flags().StringVarP(&configXMLPath, "config-xml", "c", "jenkins/template-config.xml", "Path to config XML file.")
	createJobCmd.MarkFlagRequired("name")
}
