package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusHandler(t *testing.T) {
	cases := []struct {
		name   string
		method string
		code   string // value of the code query parameter ("" = absent)
		want   int
		wantIn string // substring the body must contain; "" = body must be empty
	}{
		{"missing code defaults to 200", "GET", "", 200, "200 OK"},
		{"explicit 200", "GET", "200", 200, "200 OK"},
		{"201", "GET", "201", 201, "201 Created"},
		{"301", "GET", "301", 301, "301 Moved Permanently"},
		{"404", "GET", "404", 404, "404 Not Found"},
		{"418", "GET", "418", 418, "418 I'm a teapot"},
		{"503", "GET", "503", 503, "503 Service Unavailable"},
		{"204 has no body", "GET", "204", 204, ""},
		{"205 has no body", "GET", "205", 205, ""},
		{"304 has no body", "GET", "304", 304, ""},
		{"500", "GET", "500", 500, "500 Internal Server Error"},
		{"any method is accepted", "POST", "418", 418, "418 I'm a teapot"},
		{"HEAD is accepted", "HEAD", "404", 404, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/http", nil)
			req.URL.RawQuery = "code=" + url.QueryEscape(tc.code)
			rec := httptest.NewRecorder()
			statusHandler(rec, req)

			if got := rec.Code; got != tc.want {
				t.Errorf("status = %d, want %d", got, tc.want)
			}
			body := rec.Body.String()
			if tc.wantIn == "" && tc.method != "HEAD" {
				if body != "" {
					t.Errorf("body = %q, want empty", body)
				}
			} else if tc.wantIn != "" && !strings.Contains(body, tc.wantIn) {
				t.Errorf("body = %q, want it to contain %q", body, tc.wantIn)
			}
		})
	}
}

func TestStatusHandlerInvalidCode(t *testing.T) {
	for _, raw := range []string{"abc", "0", "199", "600", "1000", "-1", "200.0", " 200", "٢٠٠"} {
		t.Run(raw, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/http", nil)
			req.URL.RawQuery = "code=" + url.QueryEscape(raw)
			rec := httptest.NewRecorder()
			statusHandler(rec, req)

			if rec.Code != 400 {
				t.Errorf("code=%q: status = %d, want 400", raw, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "invalid code") {
				t.Errorf("code=%q: body = %q, want an invalid-code error", raw, rec.Body.String())
			}
		})
	}
}

func TestUsageHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	usageHandler(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/http?code=") {
		t.Errorf("body = %q, want it to mention /http?code=", rec.Body.String())
	}
}

func TestRequestLogger(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requests.log")
	rl, err := newRequestLogger(path)
	if err != nil {
		t.Fatal(err)
	}

	var sawBody string
	h := rl.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("handler body read: %v", err)
		}
		sawBody = string(b)
	}))

	req := httptest.NewRequest("POST", "/http?code=402", strings.NewReader("hello=world"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Probe", "abc123")
	req.RemoteAddr = "127.0.0.1:55555"
	h.ServeHTTP(httptest.NewRecorder(), req)

	if sawBody != "hello=world" {
		t.Errorf("handler saw body %q after the dump, want the original body", sawBody)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		"POST /http?code=402 HTTP/1.1",
		"X-Probe: abc123",
		"hello=world",
		"from 127.0.0.1:55555",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("request log missing %q;\ngot:\n%s", want, got)
		}
	}
}

func TestRequestLoggerDisabled(t *testing.T) {
	rl, err := newRequestLogger("")
	if err != nil {
		t.Fatal(err)
	}
	h := rl.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/http?code=200", nil))
}
