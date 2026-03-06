package jenkins

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type APIClient struct {
	JenkinsURL string
	Username   string
	APIToken   string
	httpClient *http.Client
}

func NewAPIClient() *APIClient {
	return &APIClient{
		JenkinsURL: os.Getenv("JENKINS_URL"),
		Username:   os.Getenv("JENKINS_USER_ID"),
		APIToken:   os.Getenv("JENKINS_API_TOKEN"),
		httpClient: &http.Client{},
	}
}

func (jc *APIClient) basicAuth() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(jc.Username+":"+jc.APIToken))
}

func (jc *APIClient) CreateJob(jobName string, configXMLPath string) error {
	// Read and update the job configuration XML
	configData, err := os.ReadFile(configXMLPath)
	if err != nil {
		return fmt.Errorf("failed to read XML file: %v", err)
	}

	updatedConfig := strings.ReplaceAll(string(configData), "REPO_NAME", jobName)

	// Construct the API URL
	apiURL := fmt.Sprintf("%s/createItem?name=%s", jc.JenkinsURL, jobName)

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer([]byte(updatedConfig)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", jc.basicAuth())

	// Make the request
	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Jenkins API error: %s, response: %s", resp.Status, string(body))
	}

	fmt.Printf("Job '%s' created successfully.\n", jobName)
	return nil
}

func (jc *APIClient) DeleteJob(jobName string) error {

	jenkinsURL := strings.TrimSuffix(jc.JenkinsURL, "/")
	apiURL := fmt.Sprintf("%s/job/%s/doDelete", jenkinsURL, jobName)

	req, err := http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", jc.basicAuth())

	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Jenkins API error: %s, response: %s", resp.Status, string(body))
	}

	fmt.Printf("Job '%s' deleted successfully.\n", jobName)
	return nil
}
