package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Fatalf("invalid upstream URL %q: %v", raw, err)
	}
	return u
}

func isAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func stripAPIPrefix(path string) string {
	if path == "/api" {
		return "/"
	}
	stripped := strings.TrimPrefix(path, "/api")
	if stripped == "" {
		return "/"
	}
	return stripped
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error for %s: %v", r.URL.Path, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	return proxy
}

func main() {
	port := getEnv("PORT", "8080")
	appAPIURL := mustParseURL(getEnv("APP_API_URL", "http://app-api:8080"))
	webURL := mustParseURL(getEnv("WEB_URL", "http://webapp:8080"))

	appAPIProxy := newProxy(appAPIURL)
	webProxy := newProxy(webURL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if isAPIPath(r.URL.Path) {
			r.URL.Path = stripAPIPrefix(r.URL.Path)
			if r.URL.RawPath != "" {
				r.URL.RawPath = stripAPIPrefix(r.URL.RawPath)
			}
			appAPIProxy.ServeHTTP(w, r)
			return
		}
		webProxy.ServeHTTP(w, r)
	})

	log.Printf("Starting frontend reverse proxy on port %s (api=%s web=%s)", port, appAPIURL, webURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
