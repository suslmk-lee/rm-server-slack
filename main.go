package main

import (
	"rm-server-slack/common"
	"rm-server-slack/notification"
	"rm-server-slack/storage"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	checkInterval = 10 * time.Second
)

var (
	bucketName string
	region     string
	endpoint   string
	accessKey  string
	secretKey  string
)

func init() {
	region = common.ConfInfo["nhn.region"]
	bucketName = common.ConfInfo["nhn.storage.bucket.name"]
	endpoint = common.ConfInfo["nhn.storage.endpoint.url"]
	accessKey = common.ConfInfo["nhn.storage.accessKey"]
	secretKey = common.ConfInfo["nhn.storage.secretKey"]
}

func main() {
	s3Client, err := storage.NewS3Client(region, endpoint, accessKey, secretKey)
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
				processBucket(s3Client)
			}
		}
	}
}

func processBucket(s3Client *storage.S3Client) {
	events, err := s3Client.GetEvents(bucketName)
	if err != nil {
		logrus.Errorf("Failed to get events: %v", err)
		return
	}

	for _, event := range events {
		notification.SendSlackNotification(event)
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