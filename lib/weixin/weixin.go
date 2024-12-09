package weixin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Constant for API URLs
const (
	tokenURL   = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	sendMsgURL = "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s"
)

// Token struct to hold access token
type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// Function to get access token
func GetAccessToken(corpID, agentSecret string) (string, error) {
	url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", tokenURL, corpID, agentSecret)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var tokenResponse Token
	if err := json.NewDecoder(res.Body).Decode(&tokenResponse); err != nil {
		return "", err
	}

	if tokenResponse.AccessToken == "" {
		return "", fmt.Errorf("获取 Access Token 失败")
	}

	return tokenResponse.AccessToken, nil
}

// Message struct for sending messages
type Message struct {
	ToParty string `json:"toparty"`
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// Function to send message
// Function to send message
func SendMessage(corpID string, agentSecret string, content string, agentID int) error {
	token, err := GetAccessToken(corpID, agentSecret)
	if err != nil {
		return err
	}

	// Prepare the message payload to send to all users, parties, and tags
	messagePayload := map[string]interface{}{
		"touser":  "@all", // 发送给所有用户
		"toparty": "@all", // 发送给所有部门
		"totag":   "@all", // 发送给所有标签
		"msgtype": "text",
		"text":    map[string]string{"content": content},
		"agentid": agentID,
	}

	data, err := json.Marshal(messagePayload)
	if err != nil {
		return err
	}

	// Make the HTTP request
	url := fmt.Sprintf(sendMsgURL, token)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return err
	}

	if response.ErrCode != 0 {
		return fmt.Errorf("发送消息失败: %s", response.ErrMsg)
	}

	return nil
}
