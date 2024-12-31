package jenkins

import (
  "fmt"
  "os"
  "os/exec"
  "strings"
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
    Username:   os.Getenv("JENKINS_USER_ID"),
    APIToken:   os.Getenv("JENKINS_API_TOKEN"),
    CLIPath:    os.Getenv("JENKINS_CLI_PATH"),
  }
}

func (jc *CLIClient) CreateJob(jobName string, configXMLPath string) error {

  configData, err := os.ReadFile(configXMLPath)
  if err != nil {
    return fmt.Errorf("failed to read XML file: %v", err)
  }

  updatedConfig := string(configData)
  updatedConfig = strings.ReplaceAll(updatedConfig, "REPO_NAME", jobName)

  tempFile, err := os.CreateTemp("", "jenkins-config-*.xml")
  if err != nil {
    return fmt.Errorf("failed to create temporary file: %v", err)
  }
  defer os.Remove(tempFile.Name())

  if _, err := tempFile.WriteString(updatedConfig); err != nil {
    return fmt.Errorf("failed to write to temporary file: %v", err)
  }
  tempFile.Close()

  cmd := exec.Command("java", "-jar", jc.CLIPath, "-s", jc.JenkinsURL, "create-job", jobName)
  cmd.Env = append(os.Environ(),
    "JENKINS_USER_ID="+jc.Username,
    "JENKINS_API_TOKEN="+jc.APIToken,
  )

  tempFileHandle, err := os.Open(tempFile.Name())
  if err != nil {
    return fmt.Errorf("failed to open temporary file: %v", err)
  }
  defer tempFileHandle.Close()

  cmd.Stdin = tempFileHandle

  output, err := cmd.CombinedOutput()
  if err != nil {
    return fmt.Errorf("error creating job: %v, output: %s", err, string(output))
  } 

  fmt.Println(string(output))
  return nil
}
