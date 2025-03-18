package cmd

import (
	"fmt"
	"github.com/robreris/gh-jenkins-cli/fdevsec"
	"github.com/spf13/cobra"
	"strconv"
)

var (
	orgId   string
	appName string
	appID   string
	orgVal  int
)

var createFDSCmd = &cobra.Command{
	Use:   "create-devsec",
	Short: "Create a new FortiDevSec app",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
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
			fmt.Println("./gh-jenkins-cli create-devsec --org-id <organization ID> --app-name <desired application name>")
		} else {
			orgVal, err = strconv.Atoi(orgId)
			appID, err = client.CreateApp(&orgVal, appName)
			if err != nil {
				fmt.Println("Error creating FortiDevSec Application:", err)
				return
			}
			fmt.Printf("FortiDevSec application '%s' created successfully with ID: %s\n", appName, appID)
		}
	},
}

func init() {
	rootCmd.AddCommand(createFDSCmd)
	createFDSCmd.Flags().StringVarP(&orgId, "org-id", "o", "", "FortiDevSec Organization ID (not what is showing in the console--need to get from the API.")
	createFDSCmd.Flags().StringVarP(&appName, "app-name", "a", "", "FortiDevSec Application Name.")
}
