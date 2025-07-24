package fdevsec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	ID      int    `json:"id"`
	Name    string `json:"name"`
	NumApps int    `json:"applications"`
}

type App struct {
	ID   int    `json:"id"`
	UUID string `json:"app_uuid"`
	Name string `json:"name"`
}

type AppResponse struct {
	Apps []App `json:"apps"`
}

type AppInfo struct {
	AppId   int
	AppUUID string
}

func (c *Client) GetAppInfo(repoName string, fdevSecOrgID int) (*AppInfo, error) {

	var myAppInfo AppInfo

	//Get number of applications in Org
	url := fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/get_org?org_id=%d", fdevSecOrgID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &myAppInfo, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var numApps int
	var orgs []map[string]interface{}

	if err := json.Unmarshal(bodyBytes, &orgs); err == nil {
		for _, obj := range orgs {
			if obj["id"] == fdevSecOrgID {
				if val, ok := obj["applications"].(float64); ok {
					numApps = int(val)
				}
			}
		}
	}

	var org map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &org); err == nil {
		if val, ok := org["applications"].(float64); ok {
			numApps = int(val)
		}
	}

	// Get App ID
	url = fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/get_apps?org_id=%d&limit=%d", fdevSecOrgID, numApps)
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var devsecapps AppResponse
	if err := json.NewDecoder(resp.Body).Decode(&devsecapps); err != nil {
		fmt.Println("Error processing get-apps api response", err)
		return nil, err
	}

	//fmt.Println("devsecapps: ", devsecapps)

	for _, app := range devsecapps.Apps {
		if app.Name == repoName {
			myAppInfo.AppId = app.ID
			myAppInfo.AppUUID = app.UUID
			return &myAppInfo, nil
		}
	}

	fmt.Println("myAppInfo: ", myAppInfo)

	return &myAppInfo, nil
}

func (c *Client) DeleteApp(appName string, fdevSecOrgID int) error {

	//Get App ID
	appInfo, err := c.GetAppInfo(appName, fdevSecOrgID)
	if err != nil {
		return err
	}

	//Get App UUID
	appUUID := appInfo.AppUUID

	//Update App status to deactivated
	appInfoMap := map[string]string{
		"name":          appName,
		"app_uuid":      appUUID,
		"active_status": "deactivated",
	}
	jsonData, err := json.Marshal(appInfoMap)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "https://fortidevsec.forticloud.com/api/v1/dashboard/update_app", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Error sending update_app request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	url := fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/delete_app?app_id=%d", appInfo.AppId)
	req, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *Client) GetOrgs() ([]Org, error) {
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

/*
func (c *Client) DeleteApp(orgID *int, appName string) (string, error) {
	url := fmt.Sprintf("https://fortidevsec.forticloud.com/api/v1/dashboard/delete_app?app_id=%d", *orgID)
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
*/
