package cmd

import (
  "fmt"
  "os"

  "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
  Use:		"gh-jenkins-cli",
  Short:	"A CLI tool for working with GitHub and Jenkins.",
  Long:		"A CLI tool to work cross-platform for use building and working with FortinetCloudCSE repos and Jenkins pipelines.",
}

func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}
