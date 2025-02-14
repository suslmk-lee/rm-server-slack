package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"rm-server-slack/common"
	"rm-server-slack/storage"
	"strconv"

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

	event.Data.Notes = replaceNotes(event.Data.Notes)
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

func SendSlackNotificationPrivate(event storage.CloudEvent) {
	userID := getSlackUserIDByEmail(event.Data.Email)
	if userID == "" {
		logrus.Errorf("Failed to get Slack user ID for email: %s", event.Data.Email)
		return
	}

	// Open DM conversation
	channelID, err := openConversation(userID)
	if err != nil {
		logrus.Errorf("Failed to open DM conversation: %v", err)
		return
	}

	event.Data.Notes = replaceNotes(event.Data.Notes)
	messageBlocks := createMessageBlocksPrivate(event)
	payload := map[string]interface{}{
		"channel": channelID,
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
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
		logrus.Infof("Slack DM sent successfully to user: %s", event.Data.Email)
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

func replaceNotes(notes string) string {
	notes = regexp.MustCompile(`\*\*\*`).ReplaceAllString(notes, "    -")
	notes = regexp.MustCompile(`\*\*`).ReplaceAllString(notes, "  -")
	return notes
}

func createMessageBlocks(event storage.CloudEvent) []map[string]interface{} {
	// 기본 헤더와 내용
	var messageText string
	if event.Data.DoneRatio > 0 {
		messageText = fmt.Sprintf("*:large_yellow_circle: %s :large_yellow_circle:*\n"+
			"*일감명:* %s(#%d)",
			event.Data.Assignee, event.Data.Subject, event.Data.JobID)
	} else {
		messageText = fmt.Sprintf("*:large_yellow_circle: %s :large_yellow_circle:*\n"+
			"*일감명:* %s(#%d)\n"+
			"*업무내용:* \n%s",
			event.Data.Assignee, event.Data.Subject, event.Data.JobID, event.Data.Description)
	}

	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": messageText,
			},
		},
	}

	// Notes가 있는 경우 추가
	if event.Data.Notes != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*작성내용:* \n```%s```", event.Data.Notes),
			},
		})
	}

	if event.Data.Property != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": formatPropertyChange(event.Data.PropKey, event.Data.OldValue, event.Data.Value),
			},
		})
	}

	// 공통 메타데이터 추가
	blocks = append(blocks, map[string]interface{}{
		"type": "context",
		"elements": []map[string]string{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Status:* %s", event.Data.Status),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Priority:* %s", event.Data.Priority),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Due Date:* %s", event.Data.DueDate.Format("2006-01-02")),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Created:* %s", event.Data.CreatedOn.Format("2006-01-02")),
			},
		},
	})

	return blocks
}

func createMessageBlocksPrivate(event storage.CloudEvent) []map[string]interface{} {
	// 기본 헤더와 내용
	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*:large_red_circle: %s :large_red_circle:*\n"+
					"*일감명:* %s(#%d)\n"+
					"*담당자:* %s",
					"금일완료예정", event.Data.Subject, event.Data.JobID, event.Data.Assignee),
			},
		},
	}

	if event.Data.Property != "" {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": formatPropertyChange(event.Data.PropKey, event.Data.OldValue, event.Data.Value),
			},
		})
	}

	// 공통 메타데이터 추가
	blocks = append(blocks, map[string]interface{}{
		"type": "context",
		"elements": []map[string]string{
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Status:* %s", event.Data.Status),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Priority:* %s", event.Data.Priority),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Due Date:* %s", event.Data.DueDate.Format("2006-01-02")),
			},
			{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Created:* %s", event.Data.CreatedOn.Format("2006-01-02")),
			},
		},
	})

	return blocks
}

func formatPropertyChange(propKey, oldValue, newValue string) string {
	propName := getPropName(propKey)

	if propKey == "done_ratio" {
		oldRatio, _ := strconv.Atoi(oldValue)
		newRatio, _ := strconv.Atoi(newValue)
		diff := newRatio - oldRatio

		var progressBar string
		if diff > 0 {
			// 증가: 기존 수치는 검은색, 증가분은 녹색
			progressBar = createProgressBarWithIncrease(oldRatio, newRatio)
			return fmt.Sprintf("*%s:* \n%s :: +%d%%", propName, progressBar, diff)
		} else {
			// 감소: 감소된 수치는 빨간색
			progressBar = createProgressBarWithDecrease(oldRatio, newRatio)
			return fmt.Sprintf("*%s:* \n%s :: %d%%", propName, progressBar, diff)
		}
	} else if propKey == "status_id" {
		oldID, _ := strconv.Atoi(oldValue)
		newID, _ := strconv.Atoi(newValue)
		return fmt.Sprintf("*%s:* \n`%s` => `%s`",
			propName,
			getStatusName(oldID),
			getStatusName(newID))
	}

	return fmt.Sprintf("*%s:* \n```%s => %s```", propName, oldValue, newValue)
}

func createProgressBarWithIncrease(oldRatio, newRatio int) string {
	const totalBlocks = 10
	oldBlocks := (oldRatio * totalBlocks) / 100
	newBlocks := (newRatio * totalBlocks) / 100

	var progressBar string
	for i := 0; i < totalBlocks; i++ {
		if i < oldBlocks {
			progressBar += "⬛" // 기존 진행률 (검은색)
		} else if i < newBlocks {
			progressBar += "🟩" // 증가분 (녹색)
		} else {
			progressBar += "⬜" // 남은 부분 (흰색)
		}
	}

	return progressBar
}

func createProgressBarWithDecrease(oldRatio, newRatio int) string {
	const totalBlocks = 10
	newBlocks := (newRatio * totalBlocks) / 100

	var progressBar string
	for i := 0; i < totalBlocks; i++ {
		if i < newBlocks {
			progressBar += "⬛" // 현재 진행률 (검은색)
		} else {
			progressBar += "🟥" // 감소분 (빨간색)
		}
	}

	return progressBar
}

func getPropName(propKey string) string {
	switch propKey {
	case "status_id":
		return "진행상태"
	case "due_date":
		return "마감일"
	case "done_ratio":
		return "완료율"
	case "tracker_id":
		return "트래커"
	case "parent_id":
		return "상위일감"
	case "child_id":
		return "하위일감"
	case "description":
		return "설명"
	case "priority_id":
		return "우선순위"
	case "precedes":
		return "이전"
	case "follows":
		return "팔로워"
	case "subject":
		return "일감명"
	case "start_date":
		return "시작일"
	case "estimated_hours":
		return "수행시간"
	case "assigned_to_id":
		return "담당자"
	default:
		return propKey
	}
}

func getStatusName(statusID int) string {
	switch statusID {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "의견(Opinion)"
	case 5:
		return "완료(Completetion)"
	case 6:
		return "6"
	case 7:
		return "중지(Pause)"
	default:
		return "unknown"
	}
}

func openConversation(userID string) (string, error) {
	payload := map[string]interface{}{
		"users": userID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/conversations.open", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+slackToken)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to open conversation: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Ok      bool `json:"ok"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
		Error string `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.Channel.ID, nil
}
