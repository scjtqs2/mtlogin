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

// Client http的请求处理合类
type Client struct {
	db        *leveldb.DB
	ua        string
	token     string
	lock      sync.Mutex
	proxy     *url.URL
	MTeamAuth string
	cfg       *Config
}

func NewClient(dbPath, proxy string, cfg *Config) (*Client, error) {
	var err error
	c := &Client{cfg: cfg}
	c.db, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}
	c.proxy, err = url.Parse(proxy)
	return c, nil
}

// login 通过账号密码+otp秘钥登录来获取auth
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
		u := fmt.Sprintf("https://%s/api/login", apiHost)
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
			Timeout:         timeOut,
			DisableRedirect: true,
			UserAgent:       c.ua,
		}
		if c.proxy != nil {
			options.Proxy = c.proxy.String()
		}
		options.Headers["User-Agent"] = c.ua
		options.Headers["referer"] = c.cfg.Referer
		// options.Headers["Content-Type"] = writer.FormDataContentType()
		options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
		options.Headers["Accept"] = "application/json;charset=UTF-8"
		fmt.Println("==================login start======================== ")
		defer fmt.Println("==================login end========================")
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

// check 校验auth是否有效，有效的话再进行签到更新
func (c *Client) check() error {
	if c.ua == "" {
		c.ua = ua
	}
	// 使用外部给的token
	if c.MTeamAuth != "" {
		c.token = c.MTeamAuth
	}
	u := fmt.Sprintf("https://%s/api/member/profile", apiHost)
	client, _ := cloudscraper.Init(false, false)
	options := cycletls.Options{
		Headers:         make(map[string]string),
		Body:            "",
		Timeout:         timeOut,
		DisableRedirect: true,
		UserAgent:       c.ua,
	}
	if c.proxy != nil {
		options.Proxy = c.proxy.String()
	}
	options.Headers["User-Agent"] = c.ua
	options.Headers["referer"] = c.cfg.Referer
	options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
	options.Headers["Accept"] = "application/json;charset=UTF-8"
	options.Headers["Authorization"] = fmt.Sprintf("%s", c.token)
	res, err := client.Do(u, options, http.MethodPost)
	fmt.Println("==================check start======================== ")
	if err != nil {
		fmt.Println("==================check end======================== ")
		return err
	}
	if res.Status != http.StatusOK {
		fmt.Println("==================check end======================== ")
		return errors.New(fmt.Sprintf("cookie已过期 status=%d;body=%s", res.Status, res.Body))
	}
	fmt.Printf("body %s \r\n", res.Body)
	fmt.Printf("headers %+v \r\n", res.Headers)
	fmt.Printf("Cookies %s \r\n", res.Cookies)
	fmt.Println("==================check end======================== ")
	user_info := gjson.Parse(res.Body)
	if user_info.Get("message").String() == "SUCCESS" {
		fmt.Printf("用户信息获取成功\r\n")
		// 更新最后访问时间
		uu := fmt.Sprintf("https://%s/api/member/updateLastBrowse", apiHost)
		res, err = client.Do(uu, options, http.MethodPost)
		fmt.Println("==================update start======================== ")
		defer fmt.Println("==================update end======================== ")
		fmt.Printf("body %s \r\n", res.Body)
		fmt.Printf("headers %+v \r\n", res.Headers)
		fmt.Printf("Cookies %s \r\n", res.Cookies)
		if res.JSONBody()["message"] == "SUCCESS" {
			failedCount = 0
			fmt.Printf("更新最后访问时间成功\r\n")
			return nil
		}
		// _ = c.db.Delete([]byte(dbKey), nil)
		return errors.New("连接成功，但更新状态失败")
	}
	// 连续失败5次
	if failedCount >= 5 {
		_ = c.db.Delete([]byte(dbKey), nil)
	}
	return errors.New("cookie已过期")
}
