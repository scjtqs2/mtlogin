package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/martian/log"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/mtlogin/lib/qqpush"
	"github.com/scjtqs2/mtlogin/lib/weixin"
)

type Config struct {
	UserName      string `yaml:"username"`    // m-team账号
	Password      string `yaml:"password"`    // m-team密码
	TotpSecret    string `yaml:"totp_secret"` // Google 二次验证的秘钥
	Proxy         string `yaml:"proxy"`       // 代理服务 eg: http://192.168.50.21:7890
	Crontab       string `yaml:"crontab"`     // 定时规则
	Qqpush        string `yaml:"qqpush"`
	QqpushToken   string `yaml:"qqpush_token"`
	MTeamAuth     string `yaml:"m_team_auth"`   // 直接提供登录的认证
	Ua            string `yaml:"ua"`            // auth对应的user-agent
	ApiHost       string `yaml:"api_host"`      // api的host地址。eg:"api.m-team.io"
	Referer       string `yaml:"referer"`       // referer地址
	WxCorpID      string `yaml:"WxCorpID"`      // 企业 ID
	WxAgentSecret string `yaml:"WxAgentSecret"` // 应用密钥
	WxAgentID     int    `yaml:"WxAgentID"`     // 应用 ID
	MinDelay      int    `yaml:"min_delay"`     // 最小延迟（分钟）
	MaxDelay      int    `yaml:"max_delay"`     // 最大延迟（分钟）
	TimeOut       int    `yaml:"time_out"`      // api请求的超时时间(秒）
	DbPath        string `yaml:"db_path"`       // 数据库存储位置
	Version       string `yaml:"version"`       // 系统版本号
	WebVersion    string `yaml:"web_version"`   // web版本号
	Did           string `yaml:"did"`
}

const (
	CookieModeNormal = "normal" // 普通模式
	CookieModeStrict = "strict" // 严格模式
)

type Jobserver struct {
	Cron        *cron.Cron
	cfg         *Config
	client      *Client
	failedCount int // 失败次数
	cookieMode  string
}

func NewJobserver(cfg *Config) (*Jobserver, error) {
	s := &Jobserver{cfg: cfg}
	s.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	// 添加定时任务，调用 scheduleLogin
	_, err := s.Cron.AddFunc(s.cfg.Crontab, s.scheduleLogin)
	if err != nil {
		return nil, err
	}

	s.client, err = NewClient(cfg.DbPath, s.cfg.Proxy, cfg)
	if err != nil {
		panic(err)
	}
	s.client.ua = cfg.Ua
	s.client.MTeamAuth = cfg.MTeamAuth
	s.client.did = cfg.Did
	s.cookieMode = os.Getenv("COOKIE_MODE")

	return s, nil
}

func (j *Jobserver) scheduleLogin() {
	// 生成随机的延迟（单位：分钟）
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomMinutes := rand.Intn(j.cfg.MaxDelay-j.cfg.MinDelay+1) + j.cfg.MinDelay
	randomSeconds := rand.Intn(60) // 随机生成0-59秒
	randomDelay := time.Duration(randomMinutes)*time.Minute + time.Duration(randomSeconds)*time.Second
	fmt.Printf("Random minutes for delay: %d, Random seconds for delay: %d\n", randomMinutes, randomSeconds)
	// 使用 goroutine 来在随机时间后执行登录
	go func() {
		time.Sleep(randomDelay)
		j.checkToken() // 调用具体的登录或检查逻辑
	}()
}

func (j *Jobserver) Loop() error {
	j.Cron.Run()
	return nil
}

func (j *Jobserver) checkToken() {
	fmt.Printf("checkToken \r\n")
	defer func() {
		if j.failedCount > 5 {
			_ = j.client.db.Delete([]byte(dbKey), nil) // 连续失败6次清理cookie
		}
	}()
	for i := 1; i <= 3; i++ {
		// 如果 MTeamAuth 为空，则尝试登录
		if j.cfg.MTeamAuth == "" && j.cfg.Did == "" {
			err := j.client.login(j.cfg.UserName, j.cfg.Password, j.cfg.TotpSecret, false)
			if err != nil {
				log.Errorf("m-team login failed err=%v", err)
				j.sendErrorNotification(err)
				return
			}
		}

		// 检查 token
		err := j.client.check()
		if err != nil {
			j.failedCount++
			if j.cookieMode == CookieModeStrict {
				_ = j.client.db.Delete([]byte(dbKey), nil) // 直接清理cookie
			}
			log.Errorf("m-team check token failed err=%v", err)
			if errors.Is(err, authFaildErr) && i < 3 {
				log.Errorf("token 401了，需要重新登录，重试中 try=%d", i)
				continue
			}
			j.sendErrorNotification(err)
			return
		}
		break
	}

	j.failedCount = 0
	// 成功时发送通知
	j.sendSuccessNotification()
}

func (j *Jobserver) sendErrorNotification(err error) {
	if j.cfg.Qqpush != "" {
		qqpush.Qqpush(fmt.Sprintf("m-team login failed err=%v", err), j.cfg.Qqpush, j.cfg.QqpushToken)
	}
	if j.cfg.WxCorpID != "" {
		j.sendWeixinMessage(fmt.Sprintf("m-team login failed err=%v", err))
	}
}

func (j *Jobserver) sendSuccessNotification() {

	message := fmt.Sprintf("m-team 账号%s 刷新成功\n上传量: %s\n下载量: %s\n魔力值: %s\n上次登录时间: %s\n上次刷新时间: %s",
		j.client.g_Username,
		j.client.Uploaded,
		j.client.Downloaded,
		j.client.Bonus,
		j.client.LastLogin,
		j.client.LastBrowse,
	)

	if j.cfg.Qqpush != "" {
		qqpush.Qqpush(message, j.cfg.Qqpush, j.cfg.QqpushToken)
	}
	if j.cfg.WxCorpID != "" {
		j.sendWeixinMessage(message)
	}
}

// sendWeixinMessage method to push message via WeChat
func (j *Jobserver) sendWeixinMessage(message string) {
	if j.cfg.WxCorpID != "" && j.cfg.WxAgentSecret != "" {
		err := weixin.SendMessage(j.cfg.WxCorpID, j.cfg.WxAgentSecret, message, j.cfg.WxAgentID)
		if err != nil {
			log.Errorf("企业微信推送失败: %v", err)
		}
	} else {
		log.Errorf("缺少 CorpID 或 AgentSecret")
	}
}

func (j *Jobserver) GetAllDepartments() ([]string, error) {
	token, err := weixin.GetAccessToken(j.cfg.WxCorpID, j.cfg.WxAgentSecret)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/department/list?access_token=%s", token)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response struct {
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
		Departments []struct {
			ID int `json:"id"`
		} `json:"department"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.ErrCode != 0 {
		return nil, fmt.Errorf("获取部门失败: %s", response.ErrMsg)
	}

	var departmentIDs []string
	for _, dept := range response.Departments {
		departmentIDs = append(departmentIDs, fmt.Sprintf("%d", dept.ID))
	}

	return departmentIDs, nil
}
