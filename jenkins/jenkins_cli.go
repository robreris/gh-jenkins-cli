package jenkins

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	updatedConfig := strings.ReplaceAll(string(configData), "REPO_NAME", xmlText(jobName))

	// Construct the API URL
	baseURL, err := url.Parse(strings.TrimRight(strings.TrimSpace(jc.JenkinsURL), "/"))
	if err != nil || baseURL == nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" || baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return fmt.Errorf("JENKINS_URL must be an absolute HTTP(S) Jenkins base URL without credentials, query, or fragment")
	}
	if strings.TrimSpace(jobName) == "" {
		return fmt.Errorf("Jenkins job name must not be empty")
	}
	apiURL := baseURL.String() + "/createItem?" + url.Values{"name": {jobName}}.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer([]byte(updatedConfig)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", jc.basicAuth())

	// Make the request
	// A login redirect must not be mistaken for successful job creation.
	client := *jc.httpClient
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
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

func (jc *APIClient) getCSRFCrumb() (string, string, error) {
	apiURL := fmt.Sprintf("%s/crumbIssuer/api/json", strings.TrimSuffix(jc.JenkinsURL, "/"))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create crumb request: %v", err)
	}
	req.Header.Set("Authorization", jc.basicAuth())

	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch crumb: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return "", "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read crumb response: %v", err)
	}

	var crumbData struct {
		CrumbRequestField string `json:"crumbRequestField"`
		Crumb             string `json:"crumb"`
	}
	if err := json.Unmarshal(body, &crumbData); err != nil {
		return "", "", fmt.Errorf("failed to parse crumb response: %v", err)
	}
	return crumbData.CrumbRequestField, crumbData.Crumb, nil
}

func (jc *APIClient) credentialExists(credentialID string) (bool, error) {
	apiURL := fmt.Sprintf("%s/credentials/store/system/domain/_/credential/%s/api/json",
		strings.TrimSuffix(jc.JenkinsURL, "/"), credentialID)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", jc.basicAuth())

	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check credential: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	body, _ := io.ReadAll(resp.Body)
	return false, fmt.Errorf("unexpected status checking credential: %s, response: %s", resp.Status, string(body))
}

func xmlText(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func groovySingleQuotedString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return "'" + s + "'"
}

func githubServerCredentialScript(credentialID string) string {
	credentialLiteral := groovySingleQuotedString(credentialID)
	return fmt.Sprintf(`
import jenkins.model.Jenkins
import org.jenkinsci.plugins.github.config.GitHubPluginConfig
import org.jenkinsci.plugins.github.config.GitHubServerConfig

def config = Jenkins.get().getDescriptorByType(GitHubPluginConfig)
def servers = config.getConfigs()
if (servers == null || servers.isEmpty()) {
    println "ERROR: no GitHub servers found in Jenkins configuration"
} else {
    def replacements = []
    servers.each { server ->
        def replacement = new GitHubServerConfig(%s)
        replacement.setName(server.getName())
        replacement.setApiUrl(server.getApiUrl())
        replacement.setManageHooks(server.isManageHooks())
        replacement.setClientCacheSize(server.getClientCacheSize())
        replacements.add(replacement)
        println "UPDATED: server '${server.getName()}' now uses credential %s"
    }
    config.setConfigs(replacements)
    config.save()
    println "OK"
}
`, credentialLiteral, credentialLiteral)
}

func (jc *APIClient) UpsertSecretTextCredential(credentialID, secret, description string) error {
	crumbField, crumbValue, err := jc.getCSRFCrumb()
	if err != nil {
		return err
	}

	exists, err := jc.credentialExists(credentialID)
	if err != nil {
		return err
	}

	base := strings.TrimSuffix(jc.JenkinsURL, "/")
	credXML := fmt.Sprintf(`<org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl>
  <scope>GLOBAL</scope>
  <id>%s</id>
  <description>%s</description>
  <secret>%s</secret>
</org.jenkinsci.plugins.plaincredentials.impl.StringCredentialsImpl>`, xmlText(credentialID), xmlText(description), xmlText(secret))

	var apiURL string
	if exists {
		apiURL = fmt.Sprintf("%s/credentials/store/system/domain/_/credential/%s/config.xml", base, credentialID)
	} else {
		apiURL = fmt.Sprintf("%s/credentials/store/system/domain/_/createCredentials", base)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(credXML))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Authorization", jc.basicAuth())
	if crumbField != "" {
		req.Header.Set(crumbField, crumbValue)
	}

	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusFound {
		return fmt.Errorf("Jenkins API error: %s, response: %s", resp.Status, string(body))
	}
	return nil
}

func (jc *APIClient) UpdateGitHubServerCredential(credentialID string) error {
	crumbField, crumbValue, err := jc.getCSRFCrumb()
	if err != nil {
		return err
	}

	script := githubServerCredentialScript(credentialID)

	apiURL := fmt.Sprintf("%s/scriptText", strings.TrimSuffix(jc.JenkinsURL, "/"))
	body := url.Values{"script": {script}}.Encode()

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", jc.basicAuth())
	if crumbField != "" {
		req.Header.Set(crumbField, crumbValue)
	}

	resp, err := jc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Jenkins: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	output := strings.TrimSpace(string(respBody))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Jenkins script console error: %s, response: %s", resp.Status, output)
	}
	if strings.Contains(output, "Exception") || strings.Contains(output, "ERROR:") {
		return fmt.Errorf("Jenkins script error: %s", output)
	}
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
