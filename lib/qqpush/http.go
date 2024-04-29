/**
* @Author: scjtqs
* @Date: 2022/7/18 13:45
* @Email: scjtqs@qq.com
 */
package qqpush

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/martian/log"
	"io"
	"net/http"
	"time"
)

type PostData map[string]interface{}

func Post(client http.Client, url string, header http.Header, body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		log.Errorf("创建post request失败 url： %s ,err: %v", url, err)
		return nil, err
	}
	req.Header = header
	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return io.ReadAll(res.Body)
}

func Qqpush(msg, cqq, token string) ([]byte, error) {
	posturl := fmt.Sprintf("https://wx.scjtqs.com/qq/push/pushMsg?token=%s", token)
	header := make(http.Header)
	header.Set("Users-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:32.0) Gecko/20100101 Firefox/32.0")
	header.Set("content-type", "application/json")
	postdata, _ := json.Marshal(PostData{
		"qq": cqq,
		"content": []PostData{
			{
				"msgtype": "text",
				"text":    msg,
			},
		},
		"token": token,
	})
	return Post(http.Client{Timeout: time.Second * 6}, posturl, header, postdata)
}
