package main

import (
	"log"
	"os"
	"strconv"
)

var (
	dbPath  = "/data/cookie.db"
	apiHost = "api.m-team.io"
	timeOut = 60
)

const (
	dbKey = "m-team-auth"
	ua    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0"
)

func defaultCfg() *Config {
	return &Config{
		Crontab:     "2 */2 * * *",
		Referer:     "https://kp.m-team.cc/index",
		MTeamAuth:   "",
		Ua:          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
		CorpID:      "",
		AgentSecret: "",
		AgentID:     1000002,
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
	if os.Getenv("UA") != "" {
		cfg.Ua = os.Getenv("UA")
	} else {
		cfg.Ua = ua
	}
	if os.Getenv("DB_PATH") != "" {
		dbPath = os.Getenv("DB_PATH")
	}
	if os.Getenv("API_HOST") != "" {
		apiHost = os.Getenv("API_HOST")
	}
	if os.Getenv("API_REFERER") != "" {
		cfg.Referer = os.Getenv("API_REFERER")
	}
	if os.Getenv("TIME_OUT") != "" {
		timeOut, _ = strconv.Atoi(os.Getenv("TIME_OUT"))
	}
	if os.Getenv("CorpID") != "" {
		cfg.CorpID = os.Getenv("CorpID")
	}
	if os.Getenv("AgentSecret") != "" {
		cfg.AgentSecret = os.Getenv("AgentSecret")
	}
	if os.Getenv("AgentID") != "" {
		// 从环境变量读取 AgentID 字符串，并转换为 int
		agentID, err := strconv.Atoi(os.Getenv("AgentID"))
		if err != nil {
			log.Fatalf("无法转换 AgentID 环境变量为整数: %v", err)
		}
		cfg.AgentID = agentID
	}
	job, err := NewJobserver(cfg)
	if err != nil {
		panic(err)
	}
	job.Loop()

}
