package jenkins

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type APIClient struct {
	JenkinsURL string
	Username   string
	APIToken   string
}

func NewAPIClient() *APIClient {
	return &APIClient{
		JenkinsURL: os.Getenv("JENKINS_URL"),
		Username:   os.Getenv("JENKINS_USER_ID"),
		APIToken:   os.Getenv("JENKINS_API_TOKEN"),
	}
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

	// Encode authentication credentials (Basic Auth)
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", jc.Username, jc.APIToken)))

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer([]byte(updatedConfig)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", "Basic "+auth)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
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

        // Check to ensure job exists
        apiURL := fmt.Sprintf("%s/job/%s/api/json", jenkinsURL, jobName)
        req, err := http.NewRequest("GET", apiURL, nil)
        if err != nil {
                return fmt.Errorf("error finding existing Jenkins pipeline with that name: %v", err)
        }
        

	// Construct the API URL
	apiURL = fmt.Sprintf("%s/job/%s/doDelete", jenkinsURL, jobName)

	// Encode authentication credentials (Basic Auth)
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", jc.Username, jc.APIToken)))

	// Create the HTTP request
	req, err = http.NewRequest("POST", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Authorization", "Basic "+auth)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Jenkins API error: %s, response: %s", resp.Status, string(body))
	}

	fmt.Printf("Job '%s' deleted successfully.\n", jobName)
	return nil
}
