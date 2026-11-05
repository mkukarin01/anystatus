// Command anystatus is a tiny HTTP service for testing deployments, probes,
// health checks and retry logic: it answers /http?code=NNN with the HTTP
// status NNN you ask for.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/http", statusHandler)
	mux.HandleFunc("/", usageHandler)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("anystatus: listening on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
