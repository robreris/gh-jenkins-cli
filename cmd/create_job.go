package cmd

import (
  "fmt"
  "github.com/spf13/cobra"
  "github.com/robreris/gh-jenkins-cli/jenkins"
  "io/ioutil"
)

var jobName string
var configFile string

var createJobCmd = &cobra.Command{
  Use: "create-job",
  Short: "Create a new Jenkins job",
  Run: func(cmd *cobra.Command, args []string) {
    jc := jenkins.NewCLIClient()

    configXML, err := ioutil.ReadFile(configFile)
    if err != nil {
      fmt.Println("Error reading config file: ", err)
      return
    }

    err = jc.CreateJob(jobName, string(configXML))
    if err != nil {
      fmt.Println("Error creating Jenkins job: ", err)
      return
    }

    fmt.Printf("Jenkins job '%s' created successfully.\n", jobName)
  },
}

func init() {
  rootCmd.AddCommand(createJobCmd)
  createJobCmd.Flags().StringVarP(&jobName, "name", "n", "", "Name of Jenkins job.")
  createJobCmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to config XML file.")
  createJobCmd.MarkFlagRequired("name")
  createJobCmd.MarkFlagRequired("config") 
}
