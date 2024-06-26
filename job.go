package main

import (
	"fmt"
	"github.com/google/martian/log"
	"github.com/robfig/cron/v3"
	"github.com/scjtqs2/mtlogin/lib/qqpush"
)

type Config struct {
	UserName    string `yaml:"username"`    // m-team账号
	Password    string `yaml:"password"`    // m-team密码
	TotpSecret  string `yaml:"totp_secret"` // google 二次验证的秘钥
	Proxy       string `yaml:"proxy"`       // 代理服务 eg: http://192.168.50.21:7890
	Crontab     string `yaml:"crontab"`     // 定时规则
	Qqpush      string `yaml:"qqpush"`
	QqpushToken string `yaml:"qqpush_token"`
	MTeamAuth   string `yaml:"m_team_auth"` // 直接提供登录的认证
	Ua          string `yaml:"ua"`          // auth对应的user-agent
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
	// _, err := s.Cron.AddFunc("* * * * *", s.checkToken)
	if err != nil {
		return nil, err
	}
	s.client, err = NewClient(dbPath, s.cfg.Proxy)
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
	// 非直接给auth字段，需要手动登录
	if j.cfg.MTeamAuth == "" {
		err := j.client.login(j.cfg.UserName, j.cfg.Password, j.cfg.TotpSecret)
		if err != nil {
			log.Errorf("m-team login failed err=%v", err)
			if j.cfg.Qqpush != "" {
				qqpush.Qqpush(fmt.Sprintf("m-team login failed err=%v", err), j.cfg.Qqpush, j.cfg.QqpushToken)
			}
			return
		}
	}

	err := j.client.check()
	if err != nil {
		log.Errorf("m-team check token failed err=%v", err)
		if j.cfg.Qqpush != "" {
			qqpush.Qqpush(fmt.Sprintf("m-team login failed err=%v", err), j.cfg.Qqpush, j.cfg.QqpushToken)
		}
		return
	}
	if j.cfg.Qqpush != "" {
		qqpush.Qqpush(fmt.Sprintf("m-team 账号%s刷新成功", j.cfg.UserName), j.cfg.Qqpush, j.cfg.QqpushToken)
	}
}
