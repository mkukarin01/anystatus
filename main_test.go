package main

import (
	"net/http/httptest"
	"net/url"
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
