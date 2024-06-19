package main

import (
	"encoding/json"
	"os"
	"rm-server-slack/common"
	"rm-server-slack/notification"
	"rm-server-slack/storage"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	checkInterval       = 10 * time.Second
	processedEventsFile = "processed_events.json"
)

var (
	bucketName      string
	region          string
	endpoint        string
	accessKey       string
	secretKey       string
	processedEvents map[string]bool
	mutex           = &sync.Mutex{}
)

func init() {
	region = common.ConfInfo["nhn.region"]
	bucketName = common.ConfInfo["nhn.storage.bucket.name"]
	endpoint = common.ConfInfo["nhn.storage.endpoint.url"]
	accessKey = common.ConfInfo["nhn.storage.accessKey"]
	secretKey = common.ConfInfo["nhn.storage.secretKey"]

	logrus.Printf("init() processedEvents calls")
	processedEvents = loadProcessedEvents()
}

func main() {
	logrus.Printf("main() start!")
	s3Client, err := storage.NewS3Client(region, endpoint, accessKey, secretKey, bucketName)
	if err != nil {
		logrus.Fatalf("Failed to create session: %v", err)
	}

	// 주기적으로 버킷을 확인하기 위한 ticker 설정
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if isBusinessHour() {
				logrus.Printf("processBucket(s3Client) called!")
				processBucket(s3Client)
			}
		}
	}
}

func processBucket(s3Client *storage.S3Client) {
	events, err := s3Client.GetEvents("issues/")
	if err != nil {
		logrus.Errorf("Failed to get events: %v", err)
		return
	}
	logrus.Printf("Received %d events", len(events))

	for _, event := range events {
		mutex.Lock()
		if !processedEvents[event.ID] {
			processedEvents[event.ID] = true
			saveProcessedEvents()
			mutex.Unlock()

			if shouldSendNotification(event) {
				logrus.Printf("Send Slack notification: %s", event.ID)
				notification.SendSlackNotification(event)
			}
		} else {
			mutex.Unlock()
		}
	}
}

func isBusinessHour() bool {
	now := time.Now()
	day := now.Weekday()
	hour := now.Hour()

	if day >= time.Monday && day <= time.Friday && hour >= 9 && hour < 18 {
		return true
	}
	return false
}

func shouldSendNotification(event storage.CloudEvent) bool {
	// 필터링 정책을 정의합니다. 예를 들어, 특정 이벤트 유형에 대해서만 알림을 보냅니다.
	if event.Type == "com.example.issue" && event.Data.Status == "접수(Receipt)" {
		return true
	}
	return false
}

func loadProcessedEvents() map[string]bool {
	file, err := os.Open(processedEventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]bool)
		}
		logrus.Fatalf("Failed to open processed events file: %v", err)
	}
	defer file.Close()

	var events map[string]bool
	err = json.NewDecoder(file).Decode(&events)
	if err != nil {
		if err.Error() == "EOF" {
			return make(map[string]bool)
		}
		logrus.Fatalf("Failed to decode processed events file: %v", err)
	}

	return events
}

func saveProcessedEvents() {
	file, err := os.Create(processedEventsFile)
	if err != nil {
		logrus.Errorf("Failed to create processed events file: %v", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(processedEvents)
	if err != nil {
		logrus.Errorf("Failed to encode processed events file: %v", err)
	}
}
