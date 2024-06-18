package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

type S3Client struct {
	svc      *s3.S3
	endpoint string
	bucket   string
}

type CloudEvent struct {
	EventID   string `json:"eventID"`
	EventType string `json:"eventType"`
	User      string `json:"user"`
	Message   string `json:"message"`
	Email     string `json:"email"`
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
		svc:      s3.New(sess),
		endpoint: endpoint,
		bucket:   bucket,
	}, nil
}

func (client *S3Client) GetEvents() ([]CloudEvent, error) {
	result, err := client.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(client.bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %v", err)
	}

	var events []CloudEvent
	for _, item := range result.Contents {
		// 각 파일을 다운로드합니다.
		objURL := fmt.Sprintf("%s/%s/%s", client.endpoint, client.bucket, *item.Key)
		data, err := client.getObjectFromURL(objURL)
		if err != nil {
			logrus.Errorf("Failed to get object %s: %v", *item.Key, err)
			continue
		}

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

func (client *S3Client) getObjectFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get object from URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get object from URL %s: %s", url, resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from URL %s: %v", url, err)
	}

	return data, nil
}
