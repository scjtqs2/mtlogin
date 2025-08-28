package cloudscraper

import (
	"github.com/Danny-Dasilva/CycleTLS/cycletls"
)

type CloudScrapper struct {
	client        cycletls.CycleTLS
	respChan      chan []byte
	defaultHeader map[string]string
	ja3           string
	ja4           string
	userAgent     string
}

// prepareOptions 填充 headers / userAgent / ja3 / ja4
func (cs *CloudScrapper) prepareOptions(options cycletls.Options) cycletls.Options {
	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}
	for k, v := range cs.defaultHeader {
		if _, exists := options.Headers[k]; !exists {
			options.Headers[k] = v
		}
	}
	if options.UserAgent == "" {
		options.UserAgent = cs.userAgent
	}
	// 优先使用 JA4
	if options.Ja4r == "" && cs.ja4 != "" {
		options.Ja4r = cs.ja4
	}
	if options.Ja3 == "" && cs.ja3 != "" {
		options.Ja3 = cs.ja3
	}
	return options
}

func (cs *CloudScrapper) Do(url string, options cycletls.Options, method string) (cycletls.Response, error) {
	opts := cs.prepareOptions(options)
	return cs.client.Do(url, opts, method)
}

func (cs *CloudScrapper) Queue(url string, options cycletls.Options, method string) {
	opts := cs.prepareOptions(options)
	cs.client.Queue(url, opts, method)
}

func (cs *CloudScrapper) Get(url string, headers map[string]string, body string) (cycletls.Response, error) {
	return cs.Do(url, cycletls.Options{
		Headers: headers,
		Body:    body,
	}, "GET")
}

func (cs *CloudScrapper) Post(url string, headers map[string]string, body string) (cycletls.Response, error) {
	return cs.Do(url, cycletls.Options{
		Headers: headers,
		Body:    body,
	}, "POST")
}

func (cs *CloudScrapper) RespChan() chan []byte {
	return cs.respChan
}

// Init 初始化随机 UA/JA3/JA4
func Init(mobile, workers bool) (*CloudScrapper, error) {
	browserConf, err := getUserAgents(mobile)
	if err != nil {
		return nil, err
	}

	client := cycletls.Init(workers)
	return &CloudScrapper{
		client:        client,
		defaultHeader: browserConf.Headers,
		ja3:           browserConf.Ja3,
		ja4:           browserConf.Ja4,
		userAgent:     browserConf.UserAgent,
		respChan:      client.RespChan,
	}, nil
}

// WithCustomConf 允许外部指定 UA/JA3/JA4
func WithCustomConf(userAgent, ja3, ja4 string, headers map[string]string, workers bool) *CloudScrapper {
	client := cycletls.Init(workers)
	return &CloudScrapper{
		client:        client,
		defaultHeader: headers,
		ja3:           ja3,
		ja4:           ja4,
		userAgent:     userAgent,
		respChan:      client.RespChan,
	}
}
