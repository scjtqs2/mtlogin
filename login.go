package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
	"github.com/google/martian/log"
	"github.com/scjtqs2/mtlogin/lib/cloudscraper"
	"github.com/scjtqs2/mtlogin/lib/dgoogauth"
	"github.com/scjtqs2/mtlogin/lib/utls"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/tidwall/gjson"
	"golang.org/x/net/http2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

var (
	// 需要重新登录
	authFaildErr = errors.New("Full authentication is required to access this resource")
)

// Client http的请求处理合类
type Client struct {
	db         *leveldb.DB
	ua         string
	token      string
	did        string
	visitorid  string
	lock       sync.Mutex
	proxy      *url.URL
	MTeamAuth  string
	cfg        *Config
	Uploaded   string // 新增字段
	Downloaded string // 新增字段
	Bonus      string // 新增字段
	g_Username string
	LastBrowse string
	LastLogin  string
	client     *cloudscraper.CloudScrapper
}

func NewClient(dbPath, proxy string, cfg *Config) (*Client, error) {
	var err error
	c := &Client{cfg: cfg}
	c.db, err = leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}
	c.proxy, err = url.Parse(proxy)
	c.client, _ = cloudscraper.Init(false, false)
	return c, nil
}

// login 通过账号密码+otp秘钥登录来获取auth
func (c *Client) login(username, password, totpSecret string, isopt bool) error {
	if c.ua == "" {
		c.ua = c.cfg.Ua
	}
	ck, _ := c.db.Get([]byte(dbKey), nil)
	did, _ := c.db.Get([]byte(didKey), nil)
	visitorid, _ := c.db.Get([]byte(visitoridKey), nil)
	if visitorid == nil {
		c.visitorid, _ = SecureRandomString(32)
		_ = c.db.Put([]byte(visitoridKey), []byte(c.visitorid), nil)
	} else {
		c.visitorid = string(visitorid)
	}
	var needLogin bool
	if ck != nil && string(ck) != "" && string(did) != "" {
		c.token = string(ck)
		c.did = string(did)
	} else {
		needLogin = true
	}
	if needLogin {
		u := fmt.Sprintf("https://%s/api/login", c.cfg.ApiHost)
		t := time.Now().UnixMilli()
		_sgin := sgin("POST", "/api/login", t)
		body := url.Values{}
		if isopt {
			// 二次验证
			tk, err := dgoogauth.GetTOTPToken(totpSecret)
			if err != nil {
				return err
			}
			log.Debugf("token: %s", tk)
			body.Add("otpCode", tk)
		}
		body.Add("username", username)
		body.Add("password", password)
		body.Add("turnstile", "")
		body.Add("_timestamp", strconv.FormatInt(t, 10))
		body.Add("_sgin", _sgin)
		// options, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(body.Encode()))
		// client := c.newClient()
		// options.Header.Add("User-Agent", c.ua)
		// options.Header.Add("referer", c.cfg.Referer)
		// options.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		// options.Header.Add("Accept", "application/json;charset=UTF-8")
		// options.Header.Add("Ts", strconv.FormatInt(time.Now().Unix(), 10))
		options := cycletls.Options{
			Headers:         make(map[string]string),
			Body:            body.Encode(),
			Timeout:         c.cfg.TimeOut,
			DisableRedirect: true,
			UserAgent:       c.ua,
		}
		if c.proxy != nil {
			options.Proxy = c.proxy.String()
		}
		didstr, err := SecureRandomString(32)
		options.Headers["User-Agent"] = c.ua
		options.Headers["referer"] = c.cfg.Referer
		// options.Headers["Content-Type"] = writer.FormDataContentType()
		options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
		options.Headers["Accept"] = "application/json;charset=UTF-8"
		options.Headers["Ts"] = strconv.FormatInt(time.Now().Unix(), 10)
		options.Headers["version"] = c.cfg.Version
		options.Headers["webversion"] = c.cfg.WebVersion
		options.Headers["Did"] = didstr
		options.Headers["visitorid"] = c.visitorid
		fmt.Println("==================login start======================== ")
		defer fmt.Println("==================login end========================")
		// res, err := client.Do(options)
		// if err != nil {
		// 	return err
		// }
		// defer res.Body.Close()
		// res, err := client.Do(options)
		res, err := c.client.Do(u, options, http.MethodPost)
		// bodyBytes, err := io.ReadAll(res.Body)
		fmt.Printf("body %s \r\n", res.Body)
		fmt.Printf("headers %+v \r\n", res.Headers)
		fmt.Printf("Cookies %+v \r\n", res.Cookies)
		if err != nil || res.Status != http.StatusOK {
			return errors.New(fmt.Sprintf("登录失败 status=%d;body=%s", res.Status, res.Body))
		}
		resp := gjson.Parse(res.Body)
		if resp.Get("message").String() == "SUCCESS" {
			c.token = res.Headers["Authorization"]
			c.updateDid(res.Headers)
			// c.did = res.Headers["Did"]
			_ = c.db.Put([]byte(dbKey), []byte(c.token), nil)
			// _ = c.db.Put([]byte(didKey), []byte(c.did), nil)
			return nil
		} else if resp.Get("code").Int() == 1001 {
			// 需要二次认证
			return c.login(username, password, totpSecret, true)
		} else {
			return errors.New(resp.Get("message").String())
		}
	}
	return nil
}

// updateDid 更新did
func (c *Client) updateDid(headers map[string]string) {
	if headers["Did"] != "" {
		c.did = headers["Did"]
		fmt.Printf("updateDid did=%s \r\n", c.did)
		_ = c.db.Put([]byte(didKey), []byte(c.did), nil)
	}
	if headers["did"] != "" {
		c.did = headers["did"]
		fmt.Printf("updateDid did=%s \r\n", c.did)
		_ = c.db.Put([]byte(didKey), []byte(c.did), nil)
	}
	// fmt.Printf("visitorid =%s \r\n", headers["visitorid"])
	// fmt.Printf("Visitorid =%s \r\n", headers["Visitorid"])
}

// check 校验auth是否有效，有效的话再进行签到更新
func (c *Client) check() error {
	if c.ua == "" {
		c.ua = c.cfg.Ua
	}
	// 使用外部给的token
	if c.MTeamAuth != "" {
		c.token = c.MTeamAuth
	}
	u := fmt.Sprintf("https://%s/api/member/profile", c.cfg.ApiHost)
	// client := c.newClient()
	// options, _ := http.NewRequest(http.MethodPost, u, strings.NewReader(""))
	// options.Header.Add("User-Agent", c.ua)
	// options.Header.Add("referer", c.cfg.Referer)
	// options.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	// options.Header.Add("Accept", "application/json;charset=UTF-8")
	// options.Header.Add("Authorization", fmt.Sprintf("%s", c.token))
	// options.Header.Add("Ts", strconv.FormatInt(time.Now().Unix(), 10))
	// res, err := client.Do(options)
	body := url.Values{}
	t := time.Now().UnixMilli()
	body.Add("_timestamp", strconv.FormatInt(t, 10))
	_sgin := sgin("POST", "/api/member/profile", t)
	body.Add("_sgin", _sgin)
	options := cycletls.Options{
		Headers:         make(map[string]string),
		Body:            body.Encode(),
		Timeout:         c.cfg.TimeOut,
		DisableRedirect: true,
		UserAgent:       c.ua,
	}
	if c.proxy != nil {
		options.Proxy = c.proxy.String()
	}
	options.Headers["User-Agent"] = c.ua
	options.Headers["referer"] = c.cfg.Referer
	options.Headers["origin"] = c.cfg.Referer
	options.Headers["Content-Type"] = "application/x-www-form-urlencoded; charset=UTF-8"
	options.Headers["Accept"] = "application/json;charset=UTF-8"
	options.Headers["Authorization"] = fmt.Sprintf("%s", c.token)
	options.Headers["Ts"] = strconv.FormatInt(time.Now().Unix(), 10)
	options.Headers["version"] = c.cfg.Version
	options.Headers["webversion"] = c.cfg.WebVersion
	options.Headers["Did"] = c.did
	options.Headers["visitorid"] = c.visitorid
	// 调用之前请求一下funcState
	c.funcState(&options)
	options.Headers["Ts"] = strconv.FormatInt(time.Now().Unix(), 10)
	// 更新签名
	body = url.Values{}
	t = time.Now().UnixMilli()
	body.Add("_timestamp", strconv.FormatInt(t, 10))
	_sgin = sgin("POST", "/api/member/profile", t)
	body.Add("_sgin", _sgin)
	options.Body = body.Encode()

	res, err := c.client.Do(u, options, http.MethodPost)
	fmt.Println("==================check start======================== ")
	if err != nil {
		fmt.Println("==================check end======================== ")
		return err
	}
	if res.Status != http.StatusOK {
		fmt.Println("==================check end======================== ")
		return errors.New(fmt.Sprintf("cookie已过期 status=%d;body=%s", res.Status, res.Body))
	}
	// defer res.Body.Close()
	// body, _ := io.ReadAll(res.Body)
	fmt.Printf("body %s \r\n", res.Body)
	fmt.Printf("headers %+v \r\n", res.Headers)
	fmt.Printf("Cookies %+v \r\n", res.Cookies)
	fmt.Println("==================check end======================== ")
	fmt.Println("token:", c.token)
	fmt.Println("Did:", c.did)
	// 使用 gjson 解析 body
	user_info := gjson.Parse(res.Body)
	if user_info.Get("message").String() == "SUCCESS" {
		fmt.Printf("用户信息获取成功\r\n")
		c.updateDid(res.Headers)
		options.Headers["Did"] = c.did
		// 提取 uploaded, downloaded, bonus
		uploadedBitStr := user_info.Get("data.memberCount.uploaded").String()
		downloadedBitStr := user_info.Get("data.memberCount.downloaded").String()
		bonusStr := user_info.Get("data.memberCount.bonus").String()
		c.LastLogin = user_info.Get("data.memberStatus.lastLogin").String()
		c.LastBrowse = user_info.Get("data.memberStatus.lastBrowse").String()

		// 字符串转换为整数
		uploadedBit, err := strconv.ParseInt(uploadedBitStr, 10, 64)
		if err != nil {
			return fmt.Errorf("解析上传量失败: %v", err)
		}
		downloadedBit, err := strconv.ParseInt(downloadedBitStr, 10, 64)
		if err != nil {
			return fmt.Errorf("解析下载量失败: %v", err)
		}

		// 转换为吉比特
		c.Uploaded = fmt.Sprintf("%.2f Gb", float64(uploadedBit)/1073741824)     // 处理上传量转换
		c.Downloaded = fmt.Sprintf("%.2f Gb", float64(downloadedBit)/1073741824) // 处理下载量转换
		c.Bonus = bonusStr                                                       // 假设奖金的单位不需要转换

		// 提取 username
		c.g_Username = user_info.Get("data.username").String() // 假设 username 在 data 下

		// // 请求ping操作
		// fmt.Println("==================ping start======================== ")
		// pu := fmt.Sprintf("https://%s/ping", c.cfg.ApiHost)
		// pong, err := client.Do(pu, options, http.MethodGet)
		// if err != nil {
		// 	fmt.Println("==================ping err1======================== ")
		// 	return err
		// }
		// if pong.Status != http.StatusOK {
		// 	fmt.Println("==================ping err2======================== ")
		// 	return errors.New(fmt.Sprintf("cookie已过期 status=%d;body=%s", pong.Status, pong.Body))
		// }
		// fmt.Println("==================ping end======================== ")
		c.funcState(&options)
		// 更新最后访问时间
		uu := fmt.Sprintf("https://%s/api/member/updateLastBrowse", c.cfg.ApiHost)
		// time.Sleep(time.Second * 10)
		options.Headers["Ts"] = strconv.FormatInt(time.Now().Unix(), 10)
		// 更新签名
		body = url.Values{}
		t = time.Now().UnixMilli()
		body.Add("_timestamp", strconv.FormatInt(t, 10))
		_sgin = sgin("POST", "/api/member/updateLastBrowse", t)
		body.Add("_sgin", _sgin)
		options.Body = body.Encode()

		res, err = c.client.Do(uu, options, http.MethodPost)
		// options, _ = http.NewRequest(http.MethodPost, uu, strings.NewReader(""))
		// options.Header.Add("User-Agent", c.ua)
		// options.Header.Add("referer", c.cfg.Referer)
		// options.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		// options.Header.Add("Accept", "application/json;charset=UTF-8")
		// options.Header.Add("Authorization", fmt.Sprintf("%s", c.token))
		// options.Header.Add("Ts", strconv.FormatInt(time.Now().Unix(), 10))
		// res, err = client.Do(options)
		// if err != nil {
		// 	return err
		// }
		// defer res.Body.Close()
		// body, _ = io.ReadAll(res.Body)
		fmt.Println("==================update start======================== ")
		defer fmt.Println("==================update end======================== ")
		fmt.Printf("body %s \r\n", res.Body)
		fmt.Printf("headers %+v \r\n", res.Headers)
		fmt.Printf("Cookies %+v \r\n", res.Cookies)

		resp := gjson.Parse(res.Body)
		if resp.Get("message").String() == "SUCCESS" {
			c.updateDid(res.Headers)
			options.Headers["Did"] = c.did
			fmt.Printf("更新最后访问时间成功\r\n")
			return nil
		}
		return errors.New("连接成功，但更新状态失败")
	}
	if user_info.Get("code").Int() == http.StatusUnauthorized {
		c.cleanToken()
		return authFaildErr
	}
	return errors.New("cookie已过期")
}

// funcState 调用 profile之前需要调用一次
func (c *Client) funcState(options *cycletls.Options) error {
	urls := map[string]string{
		fmt.Sprintf("https://%s/api/system/unix", c.cfg.ApiHost):          http.MethodGet,
		fmt.Sprintf("https://%s/ping", c.cfg.ApiHost):                     http.MethodGet,
		fmt.Sprintf("https://%s/api/laboratory/funcState", c.cfg.ApiHost): http.MethodPost,
		fmt.Sprintf("https://%s/api/fun/first", c.cfg.ApiHost):            http.MethodPost,
		fmt.Sprintf("https://%s/api/system/state", c.cfg.ApiHost):         http.MethodPost,
		fmt.Sprintf("https://%s/api/links/view", c.cfg.ApiHost):           http.MethodPost,
		fmt.Sprintf("https://%s/api/msg/statistic", c.cfg.ApiHost):        http.MethodPost,
	}
	for u, p := range urls {
		options.Headers["Ts"] = strconv.FormatInt(time.Now().Unix(), 10)
		body := url.Values{}
		t := time.Now().UnixMilli()
		body.Add("_timestamp", strconv.FormatInt(t, 10))
		uu, _ := url.Parse(u)
		_sgin := sgin(p, uu.Path, t)
		body.Add("_sgin", _sgin)
		options.Body = body.Encode()
		res, err := c.client.Do(u, *options, p)
		if err != nil {
			return err
		}
		g := gjson.Parse(res.Body)
		c.updateDid(res.Headers)
		options.Headers["Did"] = c.did
		fmt.Printf("url=%s ,rsp body %s \r\n", u, g.String())
	}
	return nil
}

// newClient 另类的实现模拟浏览器指纹。暂时不支持proxy。目前过cf暂时还不用这么极端的控制参数。先放着吧。
func (c *Client) newClient() *http.Client {
	tr1 := &http.Transport{}
	tr2 := &http2.Transport{
		MaxDecoderHeaderTableSize: 1 << 16,
	}

	cli := &http.Client{
		Transport: &utls.UTransport{
			Tr1:     tr1,
			Tr2:     tr2,
			Proxy:   c.proxy,
			Timeout: time.Duration(c.cfg.TimeOut) * time.Second,
		},
		Timeout: time.Duration(c.cfg.TimeOut) * time.Second,
	}
	return cli
}

// cleanCookie 清理token
func (c *Client) cleanToken() {
	_ = c.db.Delete([]byte(dbKey), nil)
}

// SecureRandomString 生成密码学安全的随机字符串（小写字母+数字）
// 参数：
//
//	length: 需要生成的字符串长度（必须 > 0）
//
// 返回值：
//
//	string: 生成的随机字符串
//	error: 错误信息
func SecureRandomString(length int) (string, error) {
	// 参数验证错误
	var (
		ErrInvalidLength = errors.New("length must be positive integer")
		ErrCharsetTooBig = errors.New("charset size exceeds 256")
	)
	// 参数验证
	if length <= 0 {
		return "", ErrInvalidLength
	}
	if len(charset) > 256 {
		return "", ErrCharsetTooBig
	}

	// 预计算常量
	const (
		bufferSize    = 512              // 随机数批量读取缓冲区大小
		maxByte       = 255 - (255 % 36) // 252 (当字符集为36时)
		charsetLength = 36               // len(charset)
	)

	// 初始化结果缓冲区
	result := make([]byte, length)

	// 创建随机数缓冲池
	pool := make([]byte, bufferSize)
	poolIndex := 0

	// 主生成循环
	for i := 0; i < length; {
		// 当缓冲池耗尽时重新填充
		if poolIndex >= bufferSize {
			if _, err := rand.Read(pool); err != nil {
				return "", err
			}
			poolIndex = 0
		}

		// 获取随机字节
		b := pool[poolIndex]
		poolIndex++

		// 筛选有效字节（避免模运算偏差）
		if b > maxByte {
			continue
		}

		// 写入结果
		result[i] = charset[b%charsetLength]
		i++ // 仅当获得有效字节时递增
	}

	return string(result), nil
}

// ComputeHmacSha1 hmacsha1 算法
func ComputeHmacSha1(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha1.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// 登录签名计算，使用了13为的时间戳
func sgin(method string, path string, t int64) string {
	// 查看main.xxxxxx.js 文件的_sgin生成得到。method+apiPath+时间戳进行hmacsha1算法。
	// return ComputeHmacSha1(fmt.Sprintf("POST&/api/login&%d", t), "HLkPcWmycL57mfJt")
	return ComputeHmacSha1(fmt.Sprintf("%s&%s&%d", method, path, t), "HLkPcWmycL57mfJt")
}
