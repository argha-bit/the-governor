package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrepareRequest(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		url     string
		headers map[string]string
		body    interface{}
		wantErr bool
	}{
		{
			name:   "valid POST request",
			method: http.MethodPost,
			url:    "http://test.local/api",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			body: map[string]string{
				"hello": "world",
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := prepareRequest(tc.method, tc.url, tc.headers, tc.body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("prepareRequest() error = %v, wantErr %v", err, tc.wantErr)
			}
			if req.Method != tc.method {
				t.Fatalf("expected method %s, got %s", tc.method, req.Method)
			}
			if req.URL.String() != tc.url {
				t.Fatalf("expected url %s, got %s", tc.url, req.URL.String())
			}
			for k, v := range tc.headers {
				if got := req.Header.Get(k); got != v {
					t.Fatalf("expected header %s=%s, got %s", k, v, got)
				}
			}
		})
	}
}

func TestMakeAPICall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer server.Close()

	tests := []struct {
		name       string
		method     string
		url        string
		headers    map[string]string
		body       interface{}
		wantStatus int
		wantBody   string
		wantErr    bool
	}{
		{
			name:       "successful GET",
			method:     http.MethodGet,
			url:        server.URL,
			headers:    map[string]string{"Accept": "application/json"},
			body:       map[string]string{"a": "b"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"ok"}`,
			wantErr:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, body, err := MakeAPICall(tc.method, tc.url, tc.headers, tc.body)
			if (err != nil) != tc.wantErr {
				t.Fatalf("MakeAPICall() error = %v, wantErr %v", err, tc.wantErr)
			}
			if status != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, status)
			}
			if strings.TrimSpace(string(body)) != tc.wantBody {
				t.Fatalf("expected body %s, got %s", tc.wantBody, string(body))
			}
		})
	}

	// test invalid body marshal
	t.Run("invalid body marshal", func(t *testing.T) {
		status, _, err := MakeAPICall(http.MethodPost, server.URL, nil, make(chan int))
		if err == nil {
			t.Fatalf("expected error for invalid body marshal, got status %d", status)
		}
	})
}
