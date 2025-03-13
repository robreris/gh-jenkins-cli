package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/fdevsec"
	"github.com/spf13/cobra"
	"strconv"
)

var deleteFDSCmd = &cobra.Command{
	Use:   "delete-devsec",
	Short: "Delete an existing FortiDevSec app",
	Run: func(cmd *cobra.Command, args []string) {
		client := fdevsec.NewClient()
		if orgId == "" {
			fmt.Println("No Org Id supplied, assuming you want to retrieve it/them...")
			orgIds, err := client.GetOrgs()
			if err != nil {
				fmt.Println("Error retrieving Org ID: ", err)
			}

			fmt.Printf("%-20s %s\n", "Name", "ID")
			for _, org := range orgIds {
				fmt.Printf("%-20s %d\n", org.Name, org.ID)
			}

			fmt.Println("To create an app, run: ")
			fmt.Println("./gh-jenkins-cli create-fds-app --org-id <organization ID> --app-name <desired application name>")
		} else {
			orgVal, err := strconv.Atoi(orgId)
			err = client.DeleteApp(appName, orgVal)
			if err != nil {
				fmt.Println("Error deleting FortiDevSec Application:", err)
				return
			}
			fmt.Printf("FortiDevSec application '%s' deleted successfully.\n", appName)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteFDSCmd)
	deleteFDSCmd.Flags().StringVarP(&orgId, "org-id", "o", "", "FortiDevSec Organization ID (not what is showing in the console--need to get from the API.")
	deleteFDSCmd.Flags().StringVarP(&appName, "app-name", "a", "", "FortiDevSec Application Name.")
}
