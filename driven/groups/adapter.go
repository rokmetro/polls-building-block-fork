package groups

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"polls/core/model"

	"github.com/rokwire/rokwire-building-block-sdk-go/services/core/auth"
)

// Adapter groups adapter
type Adapter struct {
	internalAPIKey        string
	baseURL               string
	serviceAccountManager *auth.ServiceAccountManager
}

// NewGroupsAdapter creates a new Groups BB adapter instance
func NewGroupsAdapter(config *model.Config, serviceAccountManager *auth.ServiceAccountManager) *Adapter {
	return &Adapter{internalAPIKey: config.InternalAPIKey, baseURL: config.GroupsHost, serviceAccountManager: serviceAccountManager} // pragma: allowlist secret
}

type userGroup struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Privacy          string `json:"privacy"`
	MembershipStatus string `json:"membership_status"`
}

// GroupMembership mapping. Better to access map entry by key instead of iterating for check purpose.
type GroupMembership struct {
	GroupIDsAsAdmin  []string
	GroupIDsAsMember []string
}

// GetGroupsMembership retrieves all groups that a user is a member
func (a *Adapter) GetGroupsMembership(userToken string) (*GroupMembership, error) {
	if userToken != "" {

		url := fmt.Sprintf("%s/api/user/group-memberships", a.baseURL)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", userToken))
		if err != nil {
			log.Printf("error GetGroupsMembership: request - %s", err)
			return nil, fmt.Errorf("error GetGroupsMembership: request - %s", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error GetGroupsMembership: request - %s", err)
			return nil, fmt.Errorf("error GetGroupsMembership: request - %s", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			errorBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("error GetGroupsMembership: request - %s", err)
				return nil, fmt.Errorf("error GetGroupsMembership: request - %s", err)
			}

			log.Printf("error GetGroupsMembership: request - %d. Error: %s, Body: %s", resp.StatusCode, err, string(errorBody))
			return nil, fmt.Errorf("error GetGroupsMembership: request - %d. Error: %s, Body: %s", resp.StatusCode, err, string(errorBody))
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("error GetGroupsMembership: request - %s", err)
			return nil, fmt.Errorf("error GetGroupsMembership: request - %s", err)
		}

		var groups []userGroup
		err = json.Unmarshal(data, &groups)
		if err != nil {
			log.Printf("error GetGroupsMembership: request - %s", err)
			return nil, fmt.Errorf("error GetGroupsMembership: request - %s", err)
		}

		membership := GroupMembership{}
		if len(groups) > 0 {
			for _, group := range groups {
				if group.MembershipStatus == "member" {
					membership.GroupIDsAsMember = append(membership.GroupIDsAsMember, group.ID)
				} else if group.MembershipStatus == "admin" {
					membership.GroupIDsAsAdmin = append(membership.GroupIDsAsAdmin, group.ID)
				}
			}
		}

		return &membership, nil
	}
	return nil, nil
}

// GetGroupDetails retrieves group details
func (a *Adapter) GetGroupDetails(userToken string, groupID string) (*model.Group, error) {
	if groupID != "" {

		url := fmt.Sprintf("%s/api/v2/groups/%s", a.baseURL, groupID)
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("error GetGroupDetails: request - %s", err)
			return nil, fmt.Errorf("error GetGroupDetails: request - %s", err)
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", userToken))

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error GetGroupDetails: request - %s", err)
			return nil, fmt.Errorf("error GetGroupDetails: request - %s", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			errorBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("error GetGroupDetails: request - %s", err)
				return nil, fmt.Errorf("error GetGroupDetails: request - %s", err)
			}

			log.Printf("error GetGroupDetails: request - %d. Error: %s, Body: %s", resp.StatusCode, err, string(errorBody))
			return nil, fmt.Errorf("error GetGroupDetails: request - %d. Error: %s, Body: %s", resp.StatusCode, err, string(errorBody))
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("error GetGroupDetails: request - %s", err)
			return nil, fmt.Errorf("error GetGroupDetails: request - %s", err)
		}

		var group model.Group
		err = json.Unmarshal(data, &group)
		if err != nil {
			log.Printf("error GetGroupDetails: request - %s", err)
			return nil, fmt.Errorf("error GetGroupDetails: request - %s", err)
		}

		return &group, nil
	}
	return nil, nil
}

// SendGroupNotification Sends a notification to members of a group
func (a *Adapter) SendGroupNotification(groupID string, notification model.GroupNotification) {
	go a.sendGroupNotification(groupID, notification)
}

// SendGroupNotification Sends a group notification
func (a *Adapter) sendGroupNotification(groupID string, notification model.GroupNotification) {
	if groupID != "" && notification.Subject != "" && notification.Body != "" {

		bodyBytes, err := json.Marshal(notification)
		if err != nil {
			log.Printf("error creating group notification request body - %s", err)
			return
		}

		url := fmt.Sprintf("%s/api/int/group/%s/notification", a.baseURL, groupID)
		client := &http.Client{}
		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		req.Header.Set("INTERNAL-API-KEY", a.internalAPIKey)
		if err != nil {
			log.Printf("error SendGroupNotification: request - %s", err)
			return
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error SendGroupNotification: request - %s", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Printf("error SendGroupNotification: request - %d. Error: %s", resp.StatusCode, err)
		}
	}
}

// UpdateGroupDateUpdated Updates group date updated
func (a *Adapter) UpdateGroupDateUpdated(groupID string) error {
	if groupID != "" {

		url := fmt.Sprintf("%s/api/bbs/groups/%s/date-updated", a.baseURL, groupID)

		bodyData := map[string]interface{}{
			"operation": "poll_update",
		}
		bodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			log.Printf("UpdateGroupDateUpdated:error creating notification request - %s", err)
			return err
		}

		req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
		if err != nil {
			log.Printf("UpdateGroupDateUpdated:error creating load user data request - %s", err)
			return err
		}

		resp, err := a.serviceAccountManager.MakeRequest(req, "all", "all")
		if err != nil {
			log.Printf("UpdateGroupDateUpdated: error sending request - %s", err)
			return err
		}

		defer resp.Body.Close()

		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("UpdateGroupDateUpdated: unable to read response json: %s", err)
			return fmt.Errorf("UpdateGroupDateUpdated: unable to parse response json: %s", err)
		}

		if resp.StatusCode != 200 {
			log.Printf("UpdateGroupDateUpdated: error with response code - %d, Response: %s", resp.StatusCode, responseData)
			return fmt.Errorf("UpdateGroupDateUpdated: error with response code != 200")
		}

	}
	return nil
}
