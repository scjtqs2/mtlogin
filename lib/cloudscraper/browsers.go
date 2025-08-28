package cloudscraper

import (
	_ "embed"
	"encoding/json"
	"errors"
	"math/rand"
	"time"
)

type browserDescription struct {
	UserAgents userAgents                   `json:"user_agents"`
	Ja3        map[string]string            `json:"ja3"`
	Ja4        map[string]string            `json:"ja4"`
	Headers    map[string]map[string]string `json:"headers"`
}

type userAgents struct {
	Desktop map[string]map[string][]string `json:"desktop"`
	Mobile  map[string]map[string][]string `json:"mobile"`
}

type BrowserConf struct {
	UserAgent string
	Ja3       string
	Ja4       string
	Headers   map[string]string
}

//go:embed resources/browsers.json
var browsersJson string

func readJsonFile() (browserDescription, error) {
	var browsers browserDescription
	err := json.Unmarshal([]byte(browsersJson), &browsers)
	return browsers, err
}

func randomPick[T any](arr []T) (T, error) {
	if len(arr) == 0 {
		var zero T
		return zero, errors.New("empty slice")
	}
	return arr[rand.Intn(len(arr))], nil
}

func getUserAgents(mobile bool) (BrowserConf, error) {
	rand.Seed(time.Now().UnixNano())
	browsersDescription, err := readJsonFile()
	if err != nil {
		return BrowserConf{}, err
	}

	var userAgents map[string]map[string][]string
	if mobile {
		userAgents = browsersDescription.UserAgents.Mobile
	} else {
		userAgents = browsersDescription.UserAgents.Desktop
	}

	var osList []string
	for k := range userAgents {
		osList = append(osList, k)
	}
	pickedOS, err := randomPick(osList)
	if err != nil {
		return BrowserConf{}, err
	}

	var browserList []string
	for k := range userAgents[pickedOS] {
		browserList = append(browserList, k)
	}
	browserName, err := randomPick(browserList)
	if err != nil {
		return BrowserConf{}, err
	}

	pickedBrowser := userAgents[pickedOS][browserName]
	ua, err := randomPick(pickedBrowser)
	if err != nil {
		return BrowserConf{}, err
	}

	ja3 := browsersDescription.Ja3[browserName]
	ja4 := browsersDescription.Ja4[browserName]
	headers := browsersDescription.Headers[browserName]

	if ja3 == "" && ja4 == "" {
		ja3 = "771,4865-4866-4867-49195,0-23-65281-10-11-35-16-5-51,29-23-24,0"
	}
	if len(headers) == 0 {
		headers = map[string]string{"Accept": "*/*"}
	}

	return BrowserConf{
		UserAgent: ua,
		Ja3:       ja3,
		Ja4:       ja4,
		Headers:   headers,
	}, nil
}
