package jenkins

import (
  "fmt"
  "os"
  "os/exec"
)

type CLIClient struct {
  JenkinsURL string
  Username   string
  APIToken   string
  CLIPath    string
}

func NewCLIClient() *CLIClient {
  return &CLIClient{
    JenkinsURL: os.Getenv("JENKINS_URL"),
    Username:   os.Getenv("JENKINS_USERNAME"),
    APIToken:   os.Getenv("JENKINS_API_TOKEN"),
    CLIPath:    os.Getenv("JENKINS_CLI_PATH"),
  }
}

func (jc *CLIClient) CreateJob(jobName string, configXMLPath string) error {
  cmd := exec.Command("java", "-jar", jc.CLIPath, "-s", jc.JenkinsURL, "create-job", jobName)
  cmd.Env = append(os.Environ(),
    "JENKINS_USER="+jc.Username,
    "JENKINS_API_TOKEN="+jc.APIToken,
  )

  configFile, err := os.Open(configXMLPath)
  if err != nil {
    return err
  }
  defer configFile.Close()

  cmd.Stdin = configFile

  output, err := cmd.CombinedOutput()
  if err != nil {
    return fmt.Errorf("error creating job: %v, output: %s", err, string(output))
  } 

  fmt.Println(string(output))
  return nil
}
