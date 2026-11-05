// Command anystatus is a tiny HTTP service for testing deployments, probes,
// health checks and retry logic: it answers /http?code=NNN with the HTTP
// status NNN you ask for.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const usageText = "anystatus: GET /http?code=NNN -> responds with HTTP status NNN (default 200)\n"

// statusHandler answers with the status code given in the "code" query
// parameter. A missing or empty parameter means 200; anything that is not an
// integer in [200, 599] is rejected with 400. Statuses that must not carry a
// body (204, 205, 304) are sent without one, as RFC 9110 requires.
func statusHandler(w http.ResponseWriter, r *http.Request) {
	code := http.StatusOK
	if raw := r.URL.Query().Get("code"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 200 || n > 599 {
			http.Error(w, fmt.Sprintf("invalid code %q: expected an integer in [200, 599]", raw), http.StatusBadRequest)
			return
		}
		code = n
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if code != http.StatusNoContent && code != http.StatusResetContent && code != http.StatusNotModified {
		fmt.Fprintf(w, "%d %s\n", code, http.StatusText(code))
	}
}

func usageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, usageText)
}

// requestLogger appends a full dump (wire format, body included) of every
// incoming request to a log file. An empty path disables logging.
type requestLogger struct {
	mu sync.Mutex
	w  io.Writer
}

func newRequestLogger(path string) (*requestLogger, error) {
	if path == "" {
		return &requestLogger{w: io.Discard}, nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &requestLogger{w: f}, nil
}

func (l *requestLogger) close() {
	if f, ok := l.w.(*os.File); ok {
		f.Close()
	}
}

// record writes one entry. httputil.DumpRequest reads the body but restores
// r.Body afterwards, so the wrapped handler can still consume it.
func (l *requestLogger) record(r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		dump = []byte(fmt.Sprintf("(dump failed: %v)", err))
	}
	entry := fmt.Sprintf("=== %s %s %s from %s ===\n%s\n\n",
		time.Now().Format("2006-01-02 15:04:05.000 -0700"),
		r.Method, r.RequestURI, r.RemoteAddr,
		strings.ReplaceAll(string(dump), "\r\n", "\n"))

	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.w, "%s", entry)
}

func (l *requestLogger) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l.record(r)
		next.ServeHTTP(w, r)
	})
}

// logPathFromEnv returns LOG_FILE if set (empty value disables logging),
// else "requests.log" in the working directory.
func logPathFromEnv() string {
	if s, ok := os.LookupEnv("LOG_FILE"); ok {
		return s
	}
	return "requests.log"
}

// portFromEnv returns PORT if it holds a valid port number, else 8080.
func portFromEnv() int {
	if s := os.Getenv("PORT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= 65535 {
			return n
		}
		log.Printf("anystatus: ignoring invalid PORT %q", s)
	}
	return 8080
}

func main() {
	port := flag.Int("port", portFromEnv(), "port to listen on")
	logPath := flag.String("log", logPathFromEnv(), "request log file: every request is dumped there in full (\"\" disables)")
	flag.Parse()

	rl, err := newRequestLogger(*logPath)
	if err != nil {
		log.Fatalf("anystatus: cannot open request log %q: %v", *logPath, err)
	}
	defer rl.close()

	mux := http.NewServeMux()
	mux.HandleFunc("/http", statusHandler)
	mux.HandleFunc("/", usageHandler)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           rl.middleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if *logPath != "" {
		log.Printf("anystatus: request log: %s", *logPath)
	}
	log.Printf("anystatus: listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
