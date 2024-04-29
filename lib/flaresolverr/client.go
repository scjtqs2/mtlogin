package flaresolverr

//go:generate go run github.com/vburenin/ifacemaker@v1.2.0 --file=client.go --struct=client --iface=Client --pkg=flaresolverr -y "Client interface describes wrapped Flaresolverr client." --doc=true --output=client.gen.go

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrRequestTimeout when timeout reached before flaresolverr can answer.
	ErrRequestTimeout = errors.New("maximum timeout reached")

	// ErrUnexpectedError .
	ErrUnexpectedError = errors.New("unexpected error from FlareSolverr server")
)

type client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// New creates a Flaresolverr client.
// Uses the default http client if not provided.
func New(baseURL string, timeout time.Duration, httpClient *http.Client) Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// default timeout to 60000ms
	if timeout == 0 {
		timeout = time.Millisecond * 60000
	}

	return &client{baseURL: baseURL, httpClient: httpClient, timeout: timeout}
}

type Response struct {
	Status         string            `json:"status"`
	Message        string            `json:"message"`
	StartTimestamp int64             `json:"startTimestamp"`
	EndTimestamp   int64             `json:"endTimestamp"`
	Version        string            `json:"version"`
	Session        string            `json:"session"`
	Sessions       []uuid.UUID       `json:"sessions"`
	Solution       *ResponseSolution `json:"solution"`
}

type ResponseSolution struct {
	URL     string `json:"url"`
	Status  int    `json:"status"`
	Headers struct {
		Status              string `json:"status"`
		Date                string `json:"date"`
		ContentType         string `json:"content-type"`
		Expires             string `json:"expires"`
		CacheControl        string `json:"cache-control"`
		Pragma              string `json:"pragma"`
		XFrameOptions       string `json:"x-frame-options"`
		XContentTypeOptions string `json:"x-content-type-options"`
		CfCacheStatus       string `json:"cf-cache-status"`
		ExpectCt            string `json:"expect-ct"`
		ReportTo            string `json:"report-to"`
		Nel                 string `json:"nel"`
		Server              string `json:"server"`
		CfRay               string `json:"cf-ray"`
		ContentEncoding     string `json:"content-encoding"`
		AltSvc              string `json:"alt-svc"`
		Authorization       string `json:"authorization"`
	} `json:"headers"`
	Response string `json:"response"`
	Cookies  []struct {
		Name     string  `json:"name"`
		Value    string  `json:"value"`
		Domain   string  `json:"domain"`
		Path     string  `json:"path"`
		Expires  float64 `json:"expires"`
		Size     int     `json:"size"`
		HTTPOnly bool    `json:"httpOnly"`
		Secure   bool    `json:"secure"`
		Session  bool    `json:"session"`
		SameSite string  `json:"sameSite,omitempty"`
	} `json:"cookies"`
	UserAgent string `json:"userAgent"`
}

type flaresolverrCommand struct {
	Cmd               command `json:"cmd"`
	URL               string  `json:"url"`
	Session           string  `json:"session,omitempty"`
	MaxTimeout        int     `json:"maxTimeout"`
	Cookies           []any   `json:"cookies,omitempty"`
	ReturnOnlyCookies bool    `json:"returnOnlyCookies,omitempty"`
	Proxy             string  `json:"proxy,omitempty"`
	PostData          string  `json:"postData,omitempty"`
}

// CreateSession launch a new browser instance
// which will retain cookies until you destroy it with sessions.destroy.
//
// This comes in handy, so you don't have to keep solving challenges over
// and over, and you won't need to keep sending cookies for the browser to use.
//
// This also speeds up the requests since it won't have to launch a new browser instance for every request.
func (c *client) CreateSession(ctx context.Context, session uuid.UUID, proxy ...string) (*Response, error) {
	cmd := &flaresolverrCommand{
		Cmd:     CommandSessionscreate,
		Session: handleSession(session),
	}

	if len(proxy) > 0 {
		cmd.Proxy = proxy[0]
	}

	return c.do(ctx, cmd)
}

// ListSessions Returns a list of all the active sessions.
// More for debugging if you are curious to see how many sessions are running.
// You should always make sure to properly close each session
// when you are done using them as too many may slow your computer down.
func (c *client) ListSessions(ctx context.Context) (*Response, error) {
	cmd := &flaresolverrCommand{Cmd: CommandSessionslist}
	return c.do(ctx, cmd)
}

// DestroySession will properly shut down a browser instance
// and remove all files associated with it to free up resources for a new session.
// When you no longer need to use a session you should make sure to close it.
func (c *client) DestroySession(ctx context.Context, session uuid.UUID) error {
	cmd := &flaresolverrCommand{
		Cmd:     CommandSessionsdestroy,
		Session: handleSession(session),
	}
	_, err := c.do(ctx, cmd)
	return err
}

// Get makes an HTTP GET request using flaresolverr proxy
// Session can be nil.
func (c *client) Get(ctx context.Context, u string, session uuid.UUID, proxy ...string) (*Response, error) {
	cmd := &flaresolverrCommand{
		Cmd:               CommandRequestget,
		URL:               u,
		Session:           handleSession(session),
		Cookies:           nil, // TODO: handle cookies
		ReturnOnlyCookies: false,
	}

	if len(proxy) > 0 {
		cmd.Proxy = proxy[0]
	}

	return c.do(ctx, cmd)
}

// Post makes an HTTP POST request using flaresolverr proxy
// data must be an application/x-www-form-urlencoded string.
func (c *client) Post(ctx context.Context, u string, session uuid.UUID, data string, proxy ...string) (*Response, error) {
	cmd := &flaresolverrCommand{
		Cmd:               CommandRequestpost,
		URL:               u,
		Session:           handleSession(session),
		Cookies:           nil, // TODO: handle cookies
		ReturnOnlyCookies: false,
		PostData:          data,
	}

	if len(proxy) > 0 {
		cmd.Proxy = proxy[0]
	}

	return c.do(ctx, cmd)
}

func (c *client) do(ctx context.Context, cmd *flaresolverrCommand) (*Response, error) {
	// set the flaresolverr default timeout
	cmd.MaxTimeout = int(c.timeout.Milliseconds())

	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(cmd); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}

	// set the timeout, add 10 seconds
	ctx, cancel := context.WithTimeout(ctx, c.timeout+10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, payload)
	if err != nil {
		return nil, fmt.Errorf("cannot make request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to flaresolverr: %w", err)
	}
	defer resp.Body.Close()

	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("cannot read flaresolverr response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, handleError(&response)
	}

	return &response, nil
}

func handleError(resp *Response) error {
	switch message := strings.ToLower(resp.Message); {
	case strings.Contains(message, "maximum timeout reached"):
		return ErrRequestTimeout
	default:
		return fmt.Errorf("%w: %s", ErrUnexpectedError, resp.Message)
	}
}

func handleSession(session uuid.UUID) string {
	if session == uuid.Nil {
		return ""
	}

	return session.String()
}
