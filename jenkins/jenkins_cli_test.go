package jenkins

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jobConfigFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.xml")
	if err := os.WriteFile(path, []byte(`<flow-definition><description>REPO_NAME</description></flow-definition>`), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCreateJobRequest(t *testing.T) {
	const jobName = "repo & team+test"
	for _, suffix := range []string{"", "/", "///"} {
		t.Run("trailing slashes="+suffix, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/jenkins/createItem" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				if got := r.URL.Query().Get("name"); got != jobName || len(r.URL.Query()) != 1 {
					t.Errorf("query = %q, want one name parameter for %q", r.URL.RawQuery, jobName)
				}
				user, token, ok := r.BasicAuth()
				if !ok || user != "user" || token != "token" {
					t.Error("missing basic authentication")
				}
				if r.Header.Get("Content-Type") != "application/xml" {
					t.Error("missing XML content type")
				}
				var config struct {
					Description string `xml:"description"`
				}
				if err := xml.NewDecoder(r.Body).Decode(&config); err != nil {
					t.Errorf("invalid XML: %v", err)
				} else if config.Description != jobName {
					t.Errorf("description = %q, want %q", config.Description, jobName)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			client := &APIClient{JenkinsURL: server.URL + "/jenkins" + suffix, Username: "user", APIToken: "token", httpClient: server.Client()}
			if err := client.CreateJob(jobName, jobConfigFile(t)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateJobRejectsLoginRedirectAndServerErrors(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusBadRequest, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/login" {
					t.Error("followed login redirect")
					return
				}
				w.Header().Set("Location", "/login")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "Jenkins error details")
			}))
			defer server.Close()
			client := &APIClient{JenkinsURL: server.URL, httpClient: server.Client()}
			if err := client.CreateJob("repo", jobConfigFile(t)); err == nil || !strings.Contains(err.Error(), "Jenkins error details") {
				t.Fatalf("expected Jenkins response details, got %v", err)
			}
		})
	}
}

func TestCreateJobRejectsInvalidBaseURL(t *testing.T) {
	for _, base := range []string{"", "jenkins.example.com", "ftp://jenkins.example.com", "https://jenkins.example.com?name=other", "https://jenkins.example.com/#login", "https://user:password@jenkins.example.com", "http://["} {
		client := &APIClient{JenkinsURL: base}
		if err := client.CreateJob("repo", jobConfigFile(t)); err == nil || !strings.Contains(err.Error(), "JENKINS_URL") {
			t.Errorf("base URL %q: expected validation error, got %v", base, err)
		}
	}
}

func TestXMLTextEscapesCredentialPayload(t *testing.T) {
	got := xmlText(`id<&>"token"`)
	want := `id&lt;&amp;&gt;&#34;token&#34;`

	if got != want {
		t.Fatalf("xmlText() = %q, want %q", got, want)
	}
}

func TestGroovySingleQuotedStringEscapesCredentialID(t *testing.T) {
	got := groovySingleQuotedString(`jenkins\git's-token`)
	want := `'jenkins\\git\'s-token'`

	if got != want {
		t.Fatalf("groovySingleQuotedString() = %q, want %q", got, want)
	}
}

func TestGitHubServerCredentialScriptRebuildsServerConfigs(t *testing.T) {
	script := githubServerCredentialScript("github-token")

	for _, want := range []string{
		"new GitHubServerConfig('github-token')",
		"replacement.setName(server.getName())",
		"replacement.setApiUrl(server.getApiUrl())",
		"replacement.setManageHooks(server.isManageHooks())",
		"replacement.setClientCacheSize(server.getClientCacheSize())",
		"config.setConfigs(replacements)",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q:\n%s", want, script)
		}
	}

	if strings.Contains(script, "getDeclaredField('credentialsId')") {
		t.Fatalf("script still mutates the private credentialsId field:\n%s", script)
	}
}
