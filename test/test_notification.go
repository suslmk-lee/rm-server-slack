package main

import (
	"rm-server-slack/notification"
	"rm-server-slack/storage"
	"time"

	"github.com/sirupsen/logrus"
)

func main() {
	// 테스트용 이벤트 생성
	event := storage.CloudEvent{
		SpecVersion:     "1.0",
		ID:              "test-event-001",
		Source:          "test",
		Type:            "test.notification",
		DataContentType: "application/json",
		Time:            time.Now(),
		ObjectKey:       "test/object/key",
		Data: storage.EventData{
			ID:             123,
			JobID:          456,
			Status:         "In Progress",
			Email:          "mctlmktk@gmail.com", // 테스트할 이메일 주소
			Assignee:       "이민규",
			StartDate:      time.Now(),
			DueDate:        time.Now().Add(24 * time.Hour),
			DoneRatio:      50,
			EstimatedHours: 8,
			Priority:       "High",
			Author:         "테스터",
			Subject:        "DM 테스트",
			Description:    "Slack DM 발송 테스트입니다.",
			Commentor:      "테스터",
			Notes:          "테스트 노트\n* 첫 번째 항목\n* 두 번째 항목",
			CreatedOn:      time.Now(),
		},
	}

	logrus.Info("Sending test notification...")
	notification.SendSlackNotificationPrivate(event)
	logrus.Info("Test completed")
}
