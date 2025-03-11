package main

import (
	"log"
	"os"
	"strconv"
)

const (
	dbKey        = "m-team-auth"
	didKey       = "m-team-did"
	visitoridKey = "m-team-visitorid"
	charset      = "1234567890abcdefghijklmnopqrstuvwxyz"
)

// defaultCfg 默认配置
func defaultCfg() *Config {
	return &Config{
		Crontab:       "2 */2 * * *",
		ApiHost:       "api.m-team.io",
		TimeOut:       60,
		Referer:       "https://kp.m-team.cc/",
		MTeamAuth:     "",
		Ua:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",
		WxCorpID:      "",
		WxAgentSecret: "",
		WxAgentID:     0,
		MinDelay:      0, // 默认最小延迟为0分钟
		MaxDelay:      0, // 默认最大延迟为30分钟
		DbPath:        "/data/cookie.db",
		Version:       "1.1.2",
		WebVersion:    "1120",
		Did:           "",
	}
}

func main() {
	cfg := defaultCfg()
	if os.Getenv("USERNAME") != "" {
		cfg.UserName = os.Getenv("USERNAME")
	}
	if os.Getenv("PASSWORD") != "" {
		cfg.Password = os.Getenv("PASSWORD")
	}
	if os.Getenv("TOTPSECRET") != "" {
		cfg.TotpSecret = os.Getenv("TOTPSECRET")
	}
	if os.Getenv("VERSION") != "" {
		cfg.Version = os.Getenv("VERSION")
	}
	if os.Getenv("WEB_VERSION") != "" {
		cfg.WebVersion = os.Getenv("WEB_VERSION")
	}
	if os.Getenv("PROXY") != "" {
		cfg.Proxy = os.Getenv("PROXY")
	}
	if os.Getenv("CRONTAB") != "" {
		cfg.Crontab = os.Getenv("CRONTAB")
	}
	if os.Getenv("QQPUSH") != "" {
		cfg.Qqpush = os.Getenv("QQPUSH")
	}
	if os.Getenv("QQPUSH_TOKEN") != "" {
		cfg.QqpushToken = os.Getenv("QQPUSH_TOKEN")
	}
	if os.Getenv("M_TEAM_AUTH") != "" {
		cfg.MTeamAuth = os.Getenv("M_TEAM_AUTH")
	}
	if os.Getenv("M_TEAM_DID") != "" {
		cfg.Did = os.Getenv("M_TEAM_DID")
	}
	if os.Getenv("UA") != "" {
		cfg.Ua = os.Getenv("UA")
	}
	if os.Getenv("DB_PATH") != "" {
		cfg.DbPath = os.Getenv("DB_PATH")
	}
	if os.Getenv("API_HOST") != "" {
		cfg.ApiHost = os.Getenv("API_HOST")
	}
	if os.Getenv("API_REFERER") != "" {
		cfg.Referer = os.Getenv("API_REFERER")
	}
	if os.Getenv("TIME_OUT") != "" {
		cfg.TimeOut, _ = strconv.Atoi(os.Getenv("TIME_OUT"))
	}
	if os.Getenv("WXCORPID") != "" {
		cfg.WxCorpID = os.Getenv("WXCORPID")
	}
	if os.Getenv("WXAGENTSECRET") != "" {
		cfg.WxAgentSecret = os.Getenv("WXAGENTSECRET")
	}
	if os.Getenv("WXAGENTID") != "" {
		// 从环境变量读取 AgentID 字符串，并转换为 int
		WxAgentID, err := strconv.Atoi(os.Getenv("WXAGENTID"))
		if err != nil {
			log.Fatalf("无法转换 AgentID 环境变量为整数: %v", err)
		}
		cfg.WxAgentID = WxAgentID
	}
	if os.Getenv("MINDELAY") != "" {
		// 从环境变量读取 AgentID 字符串，并转换为 int
		MinDelay, err := strconv.Atoi(os.Getenv("MINDELAY"))
		if err != nil {
			log.Fatalf("无法转换 MinDelay 环境变量为整数: %v", err)
		}
		cfg.MinDelay = MinDelay
	}
	if os.Getenv("MAXDELAY") != "" {
		// 从环境变量读取 AgentID 字符串，并转换为 int
		MaxDelay, err := strconv.Atoi(os.Getenv("MAXDELAY"))
		if err != nil {
			log.Fatalf("无法转换 MaxDelay 环境变量为整数: %v", err)
		}
		cfg.MaxDelay = MaxDelay
	}
	job, err := NewJobserver(cfg)
	if err != nil {
		panic(err)
	}
	// 本地调试直接run
	if os.Getenv("LOCAL_TEST_RUN") == "true" {
		job.checkToken()
		return
	}
	job.Loop()

}
