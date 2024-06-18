package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Client struct {
	svc    *s3.S3
	bucket string
}

type CloudEvent struct {
	SpecVersion     string    `json:"specversion"`
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	Type            string    `json:"type"`
	DataContentType string    `json:"datacontenttype"`
	Time            time.Time `json:"time"`
	Data            EventData `json:"data"`
}

type EventData struct {
	ID             int       `json:"id"`
	JobID          int       `json:"job_id"`
	Status         string    `json:"status"`
	Assignee       string    `json:"assignee"`
	StartDate      time.Time `json:"start_date"`
	DueDate        time.Time `json:"due_date"`
	DoneRatio      int       `json:"done_ratio"`
	EstimatedHours int       `json:"estimated_hours"`
	Priority       string    `json:"priority"`
	Author         string    `json:"author"`
	Subject        string    `json:"subject"`
	Description    string    `json:"description"`
	Commentor      string    `json:"commentor"`
	Notes          string    `json:"notes"`
	CreatedOn      time.Time `json:"created_on"`
}

func NewS3Client(region, endpoint, accessKey, secretKey, bucket string) (*S3Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true), // 경로 스타일을 강제 설정
	})
	if err != nil {
		return nil, err
	}

	return &S3Client{
		svc:    s3.New(sess),
		bucket: bucket,
	}, nil
}

func (client *S3Client) GetEvents(prefix string) ([]CloudEvent, error) {
	result, err := client.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(client.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %v", err)
	}

	var events []CloudEvent
	for _, item := range result.Contents {
		obj, err := client.svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(client.bucket),
			Key:    item.Key,
		})
		if err != nil {
			logrus.Errorf("Failed to get object %s: %v", *item.Key, err)
			continue
		}

		data, err := ioutil.ReadAll(obj.Body)
		if err != nil {
			logrus.Errorf("Failed to read object data %s: %v", *item.Key, err)
			continue
		}
		obj.Body.Close()

		var event CloudEvent
		err = json.Unmarshal(data, &event)
		if err != nil {
			logrus.Errorf("Failed to unmarshal json data %s: %v", *item.Key, err)
			continue
		}

		events = append(events, event)
	}

	return events, nil
}
