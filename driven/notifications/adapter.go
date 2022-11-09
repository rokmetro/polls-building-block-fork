package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"polls/core/model"
)

// Adapter implements the Notifications interface
type Adapter struct {
	internalAPIKey string
	baseURL        string
}

// NewNotificationsAdapter creates a new Notifications BB adapter instance
func NewNotificationsAdapter(config *model.Config) *Adapter {
	return &Adapter{internalAPIKey: config.InternalAPIKey, baseURL: config.NotificationsHost}
}

// SendNotification sends notification to a user
func (a *Adapter) SendNotification(recipients []model.NotificationRecipient, topic *string, title string, text string, data map[string]string) error {
	if len(recipients) > 0 {
		url := fmt.Sprintf("%s/api/int/message", a.baseURL)

		bodyData := model.NotificationMessage{
			Priority:   10,
			Topic:      topic,
			Recipients: recipients,
			Subject:    title,
			Body:       text,
			Data:       data,
		}
		bodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			log.Printf("error creating notification request - %s", err)
			return err
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			log.Printf("error creating load user data request - %s", err)
			return err
		}
		req.Header.Set("INTERNAL-API-KEY", a.internalAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error loading user data - %s", err)
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("error with response code - %d", resp.StatusCode)
			return fmt.Errorf("error with response code != 200")
		}
	}
	return nil
}

// SendMail sends email to a user
func (na *Adapter) SendMail(toEmail string, subject string, body string) {
	go na.sendMail(toEmail, subject, body)
}

func (na *Adapter) sendMail(toEmail string, subject string, body string) error {
	if len(toEmail) > 0 && len(subject) > 0 && len(body) > 0 {
		url := fmt.Sprintf("%s/api/int/mail", na.baseURL)

		bodyData := map[string]interface{}{
			"to_mail": toEmail,
			"subject": subject,
			"body":    body,
		}
		bodyBytes, err := json.Marshal(bodyData)
		if err != nil {
			log.Printf("error creating notification request - %s", err)
			return err
		}

		client := &http.Client{}
		req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
		if err != nil {
			log.Printf("error creating load user data request - %s", err)
			return err
		}
		req.Header.Set("INTERNAL-API-KEY", na.internalAPIKey)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("error sending an email - %s", err)
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("error with response code - %d", resp.StatusCode)
			return fmt.Errorf("error with response code != 200")
		}
	}
	return nil
}
