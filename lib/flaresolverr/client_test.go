package flaresolverr

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
)

func ensureFlareSolverrRunning(t *testing.T) Client {
	t.Helper()
	resp, err := http.Get("http://127.0.0.1:8191/")
	if err != nil {
		t.Fatalf("FlareSolverr is not running: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("FlareSolverr is not running: %v", resp.Status)
	}

	return &client{
		baseURL:    "http://127.0.0.1:8191/v1",
		timeout:    60 * time.Second,
		httpClient: http.DefaultClient,
	}
}

func cleanSessions(t *testing.T, c Client) {
	t.Helper()
	resp, err := c.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("could not list sessions: %v", err)
	}

	for _, session := range resp.Sessions {
		if err := c.DestroySession(context.Background(), session); err != nil {
			t.Fatalf("could not destroy session: %v", err)
		}
	}
}

func TestNew(t *testing.T) {
	type args struct {
		baseURL    string
		timeout    time.Duration
		httpClient *http.Client
	}
	tests := []struct {
		name string
		args args
		want Client
	}{
		{
			name: "Expect a client with default timeout and http client",
			args: args{
				baseURL:    "foo.bar",
				timeout:    0,
				httpClient: nil,
			},
			want: &client{
				baseURL:    "foo.bar",
				timeout:    time.Millisecond * 60000,
				httpClient: http.DefaultClient,
			},
		},
		{
			name: "Expect a client",
			args: args{
				baseURL:    "foo.bar",
				timeout:    100,
				httpClient: http.DefaultClient,
			},
			want: &client{
				baseURL:    "foo.bar",
				timeout:    100,
				httpClient: http.DefaultClient,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.baseURL, tt.args.timeout, tt.args.httpClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_client_CreateSession(t *testing.T) {
	c := ensureFlareSolverrRunning(t)
	defer cleanSessions(t, c)

	u := uuid.New()

	type args struct {
		ctx     context.Context
		session uuid.UUID
		proxy   []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			name: "Create a session",
			args: args{
				ctx:     context.Background(),
				session: u,
				proxy:   nil,
			},
			want: &Response{
				Message: "Session created successfully.",
				Session: u.String(),
				Status:  "ok",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.CreateSession(tt.args.ctx, tt.args.session, tt.args.proxy...)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(Response{}, "StartTimestamp", "EndTimestamp", "Version")); diff != "" {
				t.Errorf("CreateSession() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_client_DestroySession(t *testing.T) {
	c := ensureFlareSolverrRunning(t)
	u := uuid.New()

	// create the session to destroy
	if _, err := c.CreateSession(context.Background(), u); err != nil {
		t.Fatalf("could not create session: %v", err)
	}

	type args struct {
		ctx     context.Context
		session uuid.UUID
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Expected destroy session",
			args: args{
				ctx:     context.Background(),
				session: u,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.DestroySession(tt.args.ctx, tt.args.session); (err != nil) != tt.wantErr {
				t.Errorf("DestroySession() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_Get(t *testing.T) {
	c := ensureFlareSolverrRunning(t)

	type args struct {
		ctx     context.Context
		u       string
		session uuid.UUID
		proxy   []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			name: "Expect a response",
			args: args{
				ctx:     context.Background(),
				u:       "https://httpbin.org/status/200",
				session: uuid.Nil,
				proxy:   nil,
			},
			want: &Response{
				Status:  "ok",
				Message: "Challenge not detected!",
				Solution: &ResponseSolution{
					URL:    "https://httpbin.org/status/200",
					Status: http.StatusOK,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Get(tt.args.ctx, tt.args.u, tt.args.session, tt.args.proxy...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(
				Response{},
				"StartTimestamp",
				"EndTimestamp",
				"Version",
				"Solution.Response",
				"Solution.UserAgent",
				"Solution.Cookies",
			)); diff != "" {
				t.Errorf("Get() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_client_ListSessions(t *testing.T) {
	c := ensureFlareSolverrRunning(t)
	defer cleanSessions(t, c)

	expectedUUIDs := []uuid.UUID{
		uuid.New(),
		uuid.New(),
	}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name           string
		args           args
		want           *Response
		wantErr        bool
		createSessions []uuid.UUID
	}{
		{
			name: "No session",
			args: args{
				ctx: context.Background(),
			},
			want: &Response{
				Status:   "ok",
				Sessions: []uuid.UUID{},
			},
			createSessions: nil,
			wantErr:        false,
		},
		{
			name: "Session created",
			args: args{
				ctx: context.Background(),
			},
			want: &Response{
				Status:   "ok",
				Sessions: expectedUUIDs,
			},
			wantErr:        false,
			createSessions: expectedUUIDs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createSessions != nil {
				for _, u := range tt.createSessions {
					if _, err := c.CreateSession(tt.args.ctx, u); err != nil {
						t.Fatalf("could not create session: %v", err)
					}
				}
			}

			got, err := c.ListSessions(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListSessions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(Response{}, "StartTimestamp", "EndTimestamp", "Version")); diff != "" {
				t.Errorf("ListSessions() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_client_Post(t *testing.T) {
	c := ensureFlareSolverrRunning(t)

	type args struct {
		ctx     context.Context
		u       string
		session uuid.UUID
		data    string
		proxy   []string
	}
	tests := []struct {
		name    string
		args    args
		want    *Response
		wantErr bool
	}{
		{
			name: "Expect a response",
			args: args{
				ctx:     context.Background(),
				u:       "https://httpbin.org/anything",
				session: uuid.Nil,
				data:    "foo=bar",
				proxy:   nil,
			},
			want: &Response{
				Status:  "ok",
				Message: "Challenge not detected!",
				Solution: &ResponseSolution{
					URL:    "https://httpbin.org/anything",
					Status: http.StatusOK,
					// TODO: how to check response body
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Post(tt.args.ctx, tt.args.u, tt.args.session, tt.args.data, tt.args.proxy...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Post() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(
				Response{},
				"StartTimestamp",
				"EndTimestamp",
				"Version",
				"Solution.UserAgent",
				"Solution.Cookies",
				"Solution.Response",
			)); diff != "" {
				t.Errorf("Post() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_handleError(t *testing.T) {
	type args struct {
		resp *Response
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantErrErr error
	}{
		{
			name: "Default error",
			args: args{
				resp: &Response{
					Message: "Oops, something went wrong!",
				},
			},
			wantErr:    true,
			wantErrErr: ErrUnexpectedError,
		},
		{
			name: "Request timeout error",
			args: args{
				resp: &Response{
					Message: "maximum timeout reached",
				},
			},
			wantErr:    true,
			wantErrErr: ErrRequestTimeout,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleError(tt.args.resp)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleError() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !errors.Is(err, tt.wantErrErr) {
				t.Errorf("handleError() error = %v, wantErr %v", err, tt.wantErrErr)
			}
		})
	}
}

func Test_handleSession(t *testing.T) {
	type args struct {
		session uuid.UUID
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid UUID",
			args: args{uuid.MustParse("47d0a203-a007-4a01-b8c1-0cf0156c3cc7")},
			want: "47d0a203-a007-4a01-b8c1-0cf0156c3cc7",
		},
		{
			name: "invalid valid UUID",
			args: args{uuid.Nil},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleSession(tt.args.session); got != tt.want {
				t.Errorf("handleSession() = %v, want %v", got, tt.want)
			}
		})
	}
}
