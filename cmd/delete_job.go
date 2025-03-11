package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/jenkins"
	"github.com/spf13/cobra"
	"log"
)

var deleteJobCmd = &cobra.Command{
	Use:   "delete-job",
	Short: "Delete an existing Jenkins job",
	Run: func(cmd *cobra.Command, args []string) {
		client := jenkins.NewAPIClient()

		if jobName == "" || configXMLPath == "" {
			log.Fatal("Missing some flags.")
		}

		if err := client.DeleteJob(jobName); err != nil {
			log.Fatal("Error deleting Jenkins job: ", err)
		}

		fmt.Printf("Jenkins job '%s' deleted successfully.\n", jobName)
	},
}

func init() {
	rootCmd.AddCommand(deleteJobCmd)
	deleteJobCmd.Flags().StringVarP(&jobName, "name", "n", "", "Name of Jenkins job.")
	deleteJobCmd.MarkFlagRequired("name")
}
