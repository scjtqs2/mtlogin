package main

import (
	"errors"
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/scjtqs2/mtlogin/lib/cloudscraper"
	"github.com/scjtqs2/mtlogin/lib/dgoogauth"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"sync"
)

type Client struct {
	db    *leveldb.DB
	ua    string
	token string
	lock  sync.Mutex
	proxy *url.URL
}

func NewClient(dbPath, proxy string) (*Client, error) {
	var err error
	c := &Client{}
	c.db, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}
	c.proxy, err = url.Parse(proxy)
	return c, nil
}

func (c *Client) login(username, password, totpSecret string) error {
	if c.ua == "" {
		c.ua = ua
	}
	ck, _ := c.db.Get([]byte(dbKey), nil)
	var needLogin bool
	if ck != nil {
		c.token = string(ck)
	} else {
		needLogin = true
	}
	if needLogin {
		u := "https://kp.m-team.cc/api/login"
		// 二次验证
		tk, err := dgoogauth.GetTOTPToken(totpSecret)
		if err != nil {
			return err
		}

		body := url.Values{}
		body.Add("username", username)
		body.Add("password", password)
		if err != nil {
			return err
		}
		body.Add("otpCode", tk)

		client, _ := cloudscraper.Init(false, false)
		options := cycletls.Options{
			Headers:         make(map[string]string),
			Body:            body.Encode(),
			Timeout:         10,
			DisableRedirect: true,
			UserAgent:       c.ua,
		}
		if c.proxy != nil {
			options.Proxy = c.proxy.String()
		}
		options.Headers["User-Agent"] = c.ua
		options.Headers["referer"] = "https://kp.m-team.cc/index"
		// options.Headers["Content-Type"] = writer.FormDataContentType()
		options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
		options.Headers["Accept"] = "application/json;charset=UTF-8"
		res, err := client.Do(u, options, http.MethodPost)
		if err != nil {
			return err
		}
		fmt.Printf("body %s \r\n", res.Body)
		fmt.Printf("headers %+v \r\n", res.Headers)
		fmt.Printf("Cookies %s \r\n", res.Cookies)
		c.token = res.Headers["Authorization"]
		_ = c.db.Put([]byte(dbKey), []byte(c.token), nil)
	}
	return nil
}

func (c *Client) check() error {
	if c.ua == "" {
		c.ua = ua
	}
	u := "https://kp.m-team.cc/api/member/profile"
	client, _ := cloudscraper.Init(false, false)
	options := cycletls.Options{
		Headers:         make(map[string]string),
		Body:            "",
		Timeout:         10,
		DisableRedirect: true,
		UserAgent:       c.ua,
	}
	if c.proxy != nil {
		options.Proxy = c.proxy.String()
	}
	options.Headers["User-Agent"] = c.ua
	options.Headers["referer"] = "https://kp.m-team.cc/index"
	options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
	options.Headers["Accept"] = "application/json;charset=UTF-8"
	options.Headers["Authorization"] = fmt.Sprintf("%s", c.token)
	res, err := client.Do(u, options, http.MethodPost)
	if err != nil {
		return err
	}
	if res.Status != http.StatusOK {
		return errors.New(fmt.Sprintf("cookie已过期 status=%d;body=%s", res.Status, res.Body))
	}
	fmt.Printf("body %s \r\n", res.Body)
	fmt.Printf("headers %+v \r\n", res.Headers)
	fmt.Printf("Cookies %s \r\n", res.Cookies)
	user_info := gjson.Parse(res.Body)
	if user_info.Get("message").String() == "SUCCESS" {
		fmt.Printf("用户信息获取成功\r\n")
		// 更新最后访问时间
		uu := "https://kp.m-team.cc/api/member/updateLastBrowse"
		res, err = client.Do(uu, options, http.MethodPost)
		fmt.Printf("body %s \r\n", res.Body)
		fmt.Printf("headers %+v \r\n", res.Headers)
		fmt.Printf("Cookies %s \r\n", res.Cookies)
		if res.JSONBody()["message"] == "SUCCESS" {
			fmt.Printf("更新最后访问时间成功\r\n")
			return nil
		}
		// _ = c.db.Delete([]byte(dbKey), nil)
		return errors.New("连接成功，但更新状态失败")
	}
	_ = c.db.Delete([]byte(dbKey), nil)
	return errors.New("cookie已过期")
}
