package main

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"os"
	"rm-server-slack/common"
	"rm-server-slack/notification"
	"rm-server-slack/storage"
	"sync"
	"time"
)

const (
	checkInterval      = 10 * time.Second
	processedEventFile = "processed_event_time.json"
	movedPrefix        = "processed/"
)

var (
	bucketName        string
	region            string
	endpoint          string
	accessKey         string
	secretKey         string
	lastProcessedTime time.Time
	mutex             = &sync.Mutex{}
	kst               = time.FixedZone("KST", 9*60*60) // KST (UTC+9) 시간대
)

func init() {
	region = common.ConfInfo["nhn.region"]
	bucketName = common.ConfInfo["nhn.storage.bucket.name"]
	endpoint = common.ConfInfo["nhn.storage.endpoint.url"]
	accessKey = common.ConfInfo["nhn.storage.accessKey"]
	secretKey = common.ConfInfo["nhn.storage.secretKey"]

	lastProcessedTime = loadLastProcessedTime()
}

func main() {
	logrus.Infof("Starting application with bucket: %s, region: %s, endpoint: %s", bucketName, region, endpoint)

	s3Client, err := storage.NewS3Client(region, endpoint, accessKey, secretKey, bucketName)
	if err != nil {
		logrus.Fatalf("Failed to create session: %v", err)
	}

	// 바로 버킷을 확인
	processBucket(s3Client)

	// 주기적으로 버킷을 확인하기 위한 ticker 설정
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if isBusinessHour() {
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

	for _, event := range events {
		mutex.Lock()
		if event.Time.After(lastProcessedTime) {
			lastProcessedTime = event.Time
			saveLastProcessedTime()
			mutex.Unlock()

			if shouldSendNotification(event) {
				notification.SendSlackNotification(event)
				err = s3Client.MoveObject(event.ObjectKey, movedPrefix+event.ObjectKey)
				if err != nil {
					logrus.Errorf("Failed to move object %s: %v", event.ObjectKey, err)
				}
			}
		} else {
			mutex.Unlock()
		}
	}
}

func isBusinessHour() bool {
	now := time.Now().In(kst)
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

func loadLastProcessedTime() time.Time {
	file, err := os.Open(processedEventFile)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}
		}
		logrus.Fatalf("Failed to open processed event file: %v", err)
	}
	defer file.Close()

	var t time.Time
	err = json.NewDecoder(file).Decode(&t)
	if err != nil {
		if err.Error() == "EOF" {
			return time.Time{}
		}
		logrus.Fatalf("Failed to decode processed event file: %v", err)
	}

	return t
}

func saveLastProcessedTime() {
	file, err := os.Create(processedEventFile)
	if err != nil {
		logrus.Errorf("Failed to create processed event file: %v", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(lastProcessedTime)
	if err != nil {
		logrus.Errorf("Failed to encode processed event file: %v", err)
	}
}
