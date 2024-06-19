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
	"time"

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

	messageBlocks := createMessageBlocks(event)
	payload := map[string]interface{}{
		"channel": userID,
		"blocks":  messageBlocks,
	}
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

func createMessageBlocks(event storage.CloudEvent) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": "*------------------------------*",
			},
		},
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*New Event*\n*Type:* %s\n*User:* %s\n*Message:* %s\n*Description:* %s\n*Notes:* %s", event.Type, event.Data.Assignee, event.Data.Subject, event.Data.Description, event.Data.Notes),
			},
		},
		{
			"type": "section",
			"fields": []map[string]string{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Status:*\n%s", event.Data.Status),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Priority:*\n%s", event.Data.Priority),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Start Date:*\n%s", event.Data.StartDate.Format(time.RFC3339)),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Due Date:*\n%s", event.Data.DueDate.Format(time.RFC3339)),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Author:*\n%s", event.Data.Author),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Created On:*\n%s", event.Data.CreatedOn.Format(time.RFC3339)),
				},
			},
		},
	}
}
