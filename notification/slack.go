package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"rm-server-slack/common"
	"rm-server-slack/storage"

	"github.com/sirupsen/logrus"
)

// alarm-app bot Token
var slackToken string
var receiverEmail string

func init() {
	slackToken = common.ConfInfo["slack.bot.token"]
	receiverEmail = common.ConfInfo["slack.receiver.email"]

	fmt.Printf("receiverEmail: %s\n", receiverEmail)
}

func SendSlackNotification(event storage.CloudEvent) {
	userID := getSlackUserIDByEmail(receiverEmail)
	if userID == "" {
		logrus.Errorf("Failed to get Slack user ID for email: %s", receiverEmail)
		return
	}

	message := fmt.Sprintf("New Event\nType: %s\nUser: %s\nMessage: %s\nDescription: %s\nNotes: %s", event.Type, event.Data.Assignee, event.Data.Subject, event.Data.Description, event.Data.Notes)
	payload := map[string]string{"channel": userID, "text": message}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logrus.Errorf("Failed to marshal Slack payload: %v", err)
		return
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewBuffer(payloadBytes))
	if err != nil {
		logrus.Errorf("Failed to create new request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // TLS 검증 비활성화
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Failed to send Slack notification: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Slack API responded with status %d: %s", resp.StatusCode, string(bodyBytes))
	} else {
		logrus.Info("Slack notification sent successfully")
	}
}

func getSlackUserIDByEmail(email string) string {
	url := fmt.Sprintf("https://slack.com/api/users.lookupByEmail?email=%s", email)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Errorf("Failed to create new request: %v", err)
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // TLS 검증 비활성화
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Failed to lookup user by email: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("Slack API responded with status %d: %s", resp.StatusCode, string(bodyBytes))
		return ""
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		logrus.Errorf("Failed to decode Slack API response: %v", err)
		return ""
	}

	if result["ok"].(bool) {
		user := result["user"].(map[string]interface{})
		return user["id"].(string)
	}

	logrus.Errorf("Failed to get user ID from Slack API response: %v", result)
	return ""
}
