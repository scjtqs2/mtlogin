package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/martian/log"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/mtlogin/lib/qqpush"
	"github.com/scjtqs2/mtlogin/lib/weixin"
)

var failedCount int = 0 // 失败次数

type Config struct {
	UserName    string `yaml:"username"`    // m-team账号
	Password    string `yaml:"password"`    // m-team密码
	TotpSecret  string `yaml:"totp_secret"` // google 二次验证的秘钥
	Proxy       string `yaml:"proxy"`       // 代理服务 eg: http://192.168.50.21:7890
	Crontab     string `yaml:"crontab"`     // 定时规则
	Qqpush      string `yaml:"qqpush"`
	QqpushToken string `yaml:"qqpush_token"`
	MTeamAuth   string `yaml:"m_team_auth"`  // 直接提供登录的认证
	Ua          string `yaml:"ua"`           // auth对应的user-agent
	Referer     string `yaml:"referer"`      // referer地址
	CorpID      string `yaml:"corp_id"`      // 企业 ID
	AgentSecret string `yaml:"agent_secret"` // 应用密钥
	AgentID     int    `yaml:"agent_id"`     // 应用 ID
}

type Jobserver struct {
	Cron   *cron.Cron
	cfg    *Config
	client *Client
}

func NewJobserver(cfg *Config) (*Jobserver, error) {
	s := &Jobserver{cfg: cfg}
	s.Cron = cron.New(cron.WithParser(cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))
	_, err := s.Cron.AddFunc(s.cfg.Crontab, s.checkToken)
	if err != nil {
		return nil, err
	}
	s.client, err = NewClient(dbPath, s.cfg.Proxy, cfg)
	if err != nil {
		panic(err)
	}
	s.client.ua = cfg.Ua
	s.client.MTeamAuth = cfg.MTeamAuth
	return s, nil
}

func (j *Jobserver) Loop() error {
	j.Cron.Run()
	return nil
}

func (j *Jobserver) checkToken() {
	fmt.Printf("checkToken \r\n")

	// 如果 MTeamAuth 为空，则登录
	if j.cfg.MTeamAuth == "" {
		err := j.client.login(j.cfg.UserName, j.cfg.Password, j.cfg.TotpSecret)
		if err != nil {
			log.Errorf("m-team login failed err=%v", err)

			if j.cfg.Qqpush != "" {
				qqpush.Qqpush(fmt.Sprintf("m-team login failed err=%v", err), j.cfg.Qqpush, j.cfg.QqpushToken)
			}
			if j.cfg.CorpID != "" {
				j.sendWeixinMessage(fmt.Sprintf("m-team login failed err=%v", err))
			}

			return
		}
	}

	// 检查 token
	err := j.client.check()
	if err != nil {
		failedCount++
		log.Errorf("m-team check token failed err=%v", err)

		if j.cfg.Qqpush != "" {
			qqpush.Qqpush(fmt.Sprintf("m-team login failed err=%v", err), j.cfg.Qqpush, j.cfg.QqpushToken)
		}
		if j.cfg.CorpID != "" {
			j.sendWeixinMessage(fmt.Sprintf("m-team login failed err=%v", err))
		}

		return
	}

	// 成功时发送通知
	if j.cfg.Qqpush != "" {
		qqpush.Qqpush(fmt.Sprintf("m-team 账号%s刷新成功", j.cfg.UserName), j.cfg.Qqpush, j.cfg.QqpushToken)
	}
	if j.cfg.CorpID != "" {
		j.sendWeixinMessage(fmt.Sprintf("m-team 账号%s刷新成功", j.cfg.UserName))
	}

}

// sendWeixinMessage method to push message via WeChat
func (j *Jobserver) sendWeixinMessage(message string) {
	if j.cfg.CorpID != "" && j.cfg.AgentSecret != "" {

		err := weixin.SendMessage(j.cfg.CorpID, j.cfg.AgentSecret, message, j.cfg.AgentID)
		if err != nil {
			log.Errorf("企业微信推送失败: %v", err)
		}
	} else {
		log.Errorf("缺少 CorpID 或 AgentSecret")
	}
}

func (j *Jobserver) GetAllDepartments() ([]string, error) {
	token, err := weixin.GetAccessToken(j.cfg.CorpID, j.cfg.AgentSecret)
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
