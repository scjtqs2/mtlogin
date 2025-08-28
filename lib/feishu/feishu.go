package feishu

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// FeishuBot 飞书机器人结构体
type FeishuBot struct {
	WebhookURL string
	Secret     string
}

// TextMessage 文本消息
type TextMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// CardMessage 卡片消息
type CardMessage struct {
	MsgType string `json:"msg_type"`
	Card    Card   `json:"card"`
}

// Card 卡片结构
type Card struct {
	Config   Config    `json:"config"`
	Header   Header    `json:"header"`
	Elements []Element `json:"elements"`
}

// Config 卡片配置
type Config struct {
	WideScreenMode bool `json:"wide_screen_mode"`
	EnableForward  bool `json:"enable_forward"`
}

// Header 卡片头部
type Header struct {
	Title    Title  `json:"title"`
	Template string `json:"template"`
}

// Title 标题
type Title struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

// Element 卡片元素
type Element struct {
	Tag  string `json:"tag"`
	Text Text   `json:"text"`
}

// Text 文本元素
type Text struct {
	Content string `json:"content"`
	Tag     string `json:"tag"`
}

// NewFeishuBot 创建飞书机器人实例
func NewFeishuBot(webhookURL, secret string) *FeishuBot {
	return &FeishuBot{WebhookURL: webhookURL, Secret: secret}
}

// SendText 发送文本消息
func (bot *FeishuBot) SendText(text string) error {
	message := TextMessage{
		MsgType: "text",
	}
	message.Content.Text = text

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	return bot.sendRequest(jsonData)
}

// SendCard 发送卡片消息
func (bot *FeishuBot) SendCard(title, content string) error {
	message := CardMessage{
		MsgType: "interactive",
		Card: Card{
			Config: Config{
				WideScreenMode: true,
				EnableForward:  true,
			},
			Header: Header{
				Title: Title{
					Content: title,
					Tag:     "plain_text",
				},
				Template: "blue",
			},
			Elements: []Element{
				{
					Tag: "div",
					Text: Text{
						Content: content,
						Tag:     "lark_md",
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	return bot.sendRequest(jsonData)
}

// sendRequest 发送HTTP请求到飞书Webhook
func (bot *FeishuBot) sendRequest(data []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", bot.WebhookURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 如果设置了密钥，添加签名
	if bot.Secret != "" {
		sign, timestamp := bot.generateSign()
		req.Header.Set("X-Lark-Signature", sign)
		req.Header.Set("X-Lark-Request-Timestamp", strconv.FormatInt(timestamp, 10))
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP请求失败: %v", err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误: %s", string(body))
	}

	fmt.Println("消息发送成功:", string(body))
	return nil
}

// generateSign 生成签名
func (bot *FeishuBot) generateSign() (string, int64) {
	if bot.Secret == "" {
		return "", 0
	}

	timestamp := time.Now().Unix()
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, bot.Secret)

	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature, timestamp
}
