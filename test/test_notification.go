package main

import (
	"rm-server-slack/notification"
	"rm-server-slack/storage"
	"time"

	"github.com/sirupsen/logrus"
)

func createTestEvent(withNotes bool) storage.CloudEvent {
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
			Email:          "mctlmk@gmail.com",
			Assignee:       "이민규",
			StartDate:      time.Now(),
			DueDate:        time.Now().Add(24 * time.Hour),
			DoneRatio:      50,
			EstimatedHours: 8,
			Priority:       "High",
			Author:         "테스터",
			Subject:        "DM 테스트",
			Description:    "이것은 테스트를 위한 설명입니다.\n* 첫 번째 설명\n* 두 번째 설명",
			Commentor:      "테스터",
			CreatedOn:      time.Now(),
		},
	}

	if withNotes {
		event.Data.Notes = "이것은 테스트를 위한 노트입니다.\n* 첫 번째 노트\n* 두 번째 노트"
	}

	return event
}

func main() {
	// Notes가 있는 케이스 테스트
	logrus.Info("Testing notification with notes...")
	notification.SendSlackNotificationPrivate(createTestEvent(true))

	// 3초 대기
	time.Sleep(3 * time.Second)

	// Notes가 없는 케이스 테스트
	logrus.Info("Testing notification without notes...")
	notification.SendSlackNotificationPrivate(createTestEvent(false))

	logrus.Info("Test completed")
}
