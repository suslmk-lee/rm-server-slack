package main

import (
	"fmt"
	"log"
	"regexp"
	"rm-server-slack/common"

	"github.com/slack-go/slack"
)

func main() {
	// 환경 변수에서 Slack Bot Token 읽기
	token := common.ConfInfo["slack.bot.token"]
	if token == "" {
		log.Fatal("SLACK_BOT_TOKEN 환경 변수가 설정되지 않았습니다.")
	}

	// Slack API 클라이언트 생성
	api := slack.New(token)

	// 테스트용 고정 텍스트 문자열 (여기에 원하는 텍스트를 입력)
	testText := `
		안녕하세요.
		문의사항이 있으시면 mctlmk@gmail.com 혹은 mctlmktk@gmail.com 으로 연락 주세요.
		감사합니다.
	`

	// 이메일 주소 추출을 위한 정규 표현식
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	emails := emailRegex.FindAllString(testText, -1)
	if len(emails) == 0 {
		log.Println("텍스트에서 이메일 주소를 찾지 못했습니다.")
		return
	}

	// 추출된 각 이메일에 대해 DM 전송
	for _, email := range emails {
		// 이메일로 Slack 사용자 정보 조회
		user, err := api.GetUserByEmail(email)
		if err != nil {
			log.Printf("이메일 '%s'에 해당하는 사용자를 찾을 수 없습니다: %v", email, err)
			continue
		}

		// 해당 사용자와 DM 채널 열기
		imChannel, _, _, err := api.OpenConversation(&slack.OpenConversationParameters{
			Users: []string{user.ID},
		})
		if err != nil {
			log.Printf("사용자 %s와 DM 채널을 열지 못했습니다: %v", user.Name, err)
			continue
		}

		// DM 메시지 작성
		messageText := fmt.Sprintf("안녕하세요 %s님,\n테스트 메시지입니다. 이 DM은 이메일 '%s'를 기반으로 전송되었습니다.", user.Name, email)

		// DM 전송
		_, _, err = api.PostMessage(imChannel.ID, slack.MsgOptionText(messageText, false))
		if err != nil {
			log.Printf("사용자 %s에게 DM 전송 실패: %v", user.Name, err)
			continue
		}

		log.Printf("사용자 %s(%s)에게 DM을 전송했습니다.", user.Name, email)
	}
}
