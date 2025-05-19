package dingtalkrobot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// DingTalkRobot 钉钉机器人配置
type DingTalkRobot struct {
	WebhookURL string // Webhook地址
	Secret     string // 加签密钥
}

// TextMessage 文本消息结构
type TextMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
	At struct {
		AtMobiles []string `json:"atMobiles"` // 被@人的手机号
		IsAtAll   bool     `json:"isAtAll"`   // 是否@所有人
	} `json:"at"`
}

// NewDingTalkRobot 创建钉钉机器人实例
func NewDingTalkRobot(webhookURL, secret string) *DingTalkRobot {
	return &DingTalkRobot{
		WebhookURL: webhookURL,
		Secret:     secret,
	}
}

// generateSign 生成加签
func (d *DingTalkRobot) generateSign() (string, string) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/1e6, 10)
	secret := d.Secret

	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return timestamp, sign
}

// SendTextMessage 发送文本消息
func (d *DingTalkRobot) SendTextMessage(content string, atMobiles []string, isAtAll bool) error {
	// 生成签名和时间戳
	timestamp, sign := d.generateSign()

	// 构造消息体
	msg := TextMessage{
		MsgType: "text",
	}
	msg.Text.Content = content
	msg.At.AtMobiles = atMobiles
	msg.At.IsAtAll = isAtAll

	// 转换为JSON
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message error: %v", err)
	}

	// 构造请求URL
	reqURL := fmt.Sprintf("%s&timestamp=%s&sign=%s", d.WebhookURL, timestamp, sign)

	// 发送HTTP请求
	resp, err := http.Post(reqURL, "application/json", bytes.NewBuffer(jsonMsg))
	if err != nil {
		return fmt.Errorf("http post error: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response error: %v", err)
	}

	// 检查响应
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk robot return error: %s", string(body))
	}

	return nil
}
