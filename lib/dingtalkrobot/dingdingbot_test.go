package dingtalkrobot

import (
	"fmt"
	"testing"
)

func TestDingdingBot(t *testing.T) {
	// 替换为你的Webhook URL和加签密钥
	webhookURL := "https://oapi.dingtalk.com/robot/send?access_token=TOKEN"
	secret := "secret"

	// 创建机器人实例
	robot := NewDingTalkRobot(webhookURL, secret)

	// 发送文本消息
	err := robot.SendTextMessage("Hello from Golang!", []string{""}, true)
	if err != nil {
		fmt.Printf("Send message error: %v\n", err)
	} else {
		fmt.Println("Message sent successfully!")
	}
}
