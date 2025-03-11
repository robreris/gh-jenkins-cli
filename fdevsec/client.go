package fdevsec

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Client struct {
	Token string
}

func NewClient() *Client {
	return &Client{
		Token: os.Getenv("FDS_TOKEN"),
	}
}

type Org struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type App struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type AppResponse struct {
	Apps []App `json:"apps"`
}

func (c *Client) GetAppID(repoName string, fdevSecOrgID int) (int, error) {

	url := fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/get_apps?org_id=%d", fdevSecOrgID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var devsecapps AppResponse
	if err := json.NewDecoder(resp.Body).Decode(&devsecapps); err != nil {
		fmt.Println("Error processing get-apps api response", err)
		return 0, err
	}

	for _, app := range devsecapps.Apps {
		if app.Name == repoName {
			return app.ID, nil
		}
	}

	return 0, fmt.Errorf("app with name %s not found", repoName)
}

func (c *Client) GetOrgID() ([]Org, error) {
	req, err := http.NewRequest("GET", "https://fortidevsec.forticloud.com/api/v1/dashboard/get_orgs", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var orgs []Org
	err = json.NewDecoder(resp.Body).Decode(&orgs)
	if err != nil {
		return nil, err
	}
	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organizations found")
	}

	return orgs, nil
}

func (c *Client) CreateApp(orgID *int, appName string) (string, error) {
	url := fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/create_app?org_id=%d&app_name=%s", *orgID, appName)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	appID, ok := result["app_uuid"].(string)
	if !ok {
		return "", fmt.Errorf("invalid app_uuid format")
	}
	return appID, nil
}
