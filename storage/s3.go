package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Client struct {
	svc *s3.S3
}

type CloudEvent struct {
	EventID   string `json:"eventID"`
	EventType string `json:"eventType"`
	User      string `json:"user"`
	Message   string `json:"message"`
	Email     string `json:"email"`
}

func NewS3Client(region, endpoint, accessKey, secretKey string) (*S3Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Endpoint:    aws.String(endpoint),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, err
	}

	return &S3Client{
		svc: s3.New(sess),
	}, nil
}

func (client *S3Client) GetEvents(bucketName string) ([]CloudEvent, error) {
	result, err := client.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %v", err)
	}

	var events []CloudEvent
	for _, item := range result.Contents {
		obj, err := client.svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucketName),
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
