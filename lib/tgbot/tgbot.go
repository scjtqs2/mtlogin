package tgbot

import (
	"github.com/google/martian/log"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendTextMessage(botToken string, chatID int64, text string, tgProxy string) error {
	var client = http.DefaultClient
	if tgProxy != "" && strings.HasPrefix(tgProxy, "http") {
		proxyURL, err := url.Parse(tgProxy)
		if err != nil {
			log.Errorf("init http client faild with err tgproxy, proxy=%s, err=%v", tgProxy, err)
			return err
		}
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	} else if tgProxy != "" && strings.HasPrefix(tgProxy, "socks5") {
		dialer, err := proxy.SOCKS5("tcp", stripScheme(tgProxy), nil, proxy.Direct)
		if err != nil {
			log.Errorf("init http client faild with err tgproxy, proxy=%s, err=%v", tgProxy, err)
			return err
		}

		dialFunc := func(network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
		client = &http.Client{
			Transport: &http.Transport{
				Dial:              dialFunc,
				DisableKeepAlives: false,
				MaxIdleConns:      10,
				IdleConnTimeout:   30 * time.Second,
			},
		}
	}
	bot, err := tgbotapi.NewBotAPIWithClient(botToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		log.Errorf("init tgbotapi faild err=%v", err)
		return err
	}
	msg := tgbotapi.NewMessage(chatID, text)
	// msg.ParseMode = "Markdown"
	_, err = bot.Send(msg)
	if err != nil {
		log.Errorf("tgbotapi 发送失败: %v", err)
		return err
	}
	return nil
}

// stripScheme 去掉 URL 中的协议前缀
func stripScheme(proxyAddr string) string {
	if u, err := url.Parse(proxyAddr); err == nil && u.Host != "" {
		return u.Host
	}
	return proxyAddr
}
