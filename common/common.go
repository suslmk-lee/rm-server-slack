package common

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

type AppConfigProperties map[string]string

var ConfInfo AppConfigProperties

// init() 함수는 패키지 내에서 가장먼저 실행되는 함수
// main() 함수 내에 포함된 패키지가 있을 경우 패키지내에 포함된 init() 함수가 먼저 실행된다.
func init() {
	path, _ := os.Getwd()
	println(path)
	_, err := ReadPropertiesFile("config.properties")
	if err != nil {
		path, _ := os.Getwd()
		println(path)
		return
	}
}

func ReadPropertiesFile(filename string) (AppConfigProperties, error) {
	ConfInfo = AppConfigProperties{}

	if len(filename) == 0 {
		return ConfInfo, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				ConfInfo[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return ConfInfo, nil
}

func RandomString(n int) string {
	var letterRunes = []rune("1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type KSTFormatter struct {
	// TimestampFormat은 포매팅할 타임스탬프의 형식입니다.
	TimestampFormat string
	// FullTimestamp가 true이면 전체 타임스탬프를 출력합니다.
	FullTimestamp bool
}

func (f *KSTFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// "Asia/Seoul" 타임존 로드
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return nil, err
	}
	// entry.Time을 KST로 변환
	entry.Time = entry.Time.In(loc)

	// 기본 TextFormatter 생성 (원하는 옵션 전달)
	formatter := &logrus.TextFormatter{
		TimestampFormat: f.TimestampFormat,
		FullTimestamp:   f.FullTimestamp,
	}

	return formatter.Format(entry)
}
