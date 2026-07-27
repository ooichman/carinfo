package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type CarInfoRequest struct {
	Module       string `json:"module"`
	Manufacture  string `json:"manufacture"`
}

type CarInfoAnswer struct {
	Name         string `json:"name"`
	Year         int    `json:"year"`
	Condition    string `json:"condition"`
	Reason       string `json:"reason"`
	Module       string `json:"module"`
	Manufacture  string `json:"manufacture"`
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}
	return value
}

func dbapiBase() string {
	base := strings.TrimRight(getEnv("DBAPI_URL", "http://dbapi:8080"), "/")
	// Allow legacy value that included /query
	return strings.TrimSuffix(base, "/query")
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

func proxyToDBAPI(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	var body io.Reader
	if r.Body != nil && r.Method != http.MethodGet && r.Method != http.MethodDelete {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "unable to read request body", http.StatusBadRequest)
			return
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(r.Method, dbapiBase()+upstreamPath, body)
	if err != nil {
		http.Error(w, "unable to create upstream request", http.StatusInternalServerError)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("dbapi request failed: %v", err)
		http.Error(w, "unable to reach dbapi", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "unable to read dbapi response", http.StatusBadGateway)
		return
	}

	for _, h := range []string{"Content-Type"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Content-Type") == "" && len(resBody) > 0 {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.StatusCode)
	if len(resBody) > 0 {
		_, _ = w.Write(resBody)
	}
}

// Legacy manufacturer query: POST /v1 → dbapi POST /query
func QueryRes(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1" {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method must be POST", http.StatusMethodNotAllowed)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read body", http.StatusBadRequest)
		return
	}

	var requestdata CarInfoRequest
	if err := json.Unmarshal(data, &requestdata); err != nil {
		http.Error(w, "unable to unmarshal body", http.StatusBadRequest)
		return
	}

	sendBody, _ := json.Marshal(requestdata)
	req, err := http.NewRequest(http.MethodPost, dbapiBase()+"/query", bytes.NewReader(sendBody))
	if err != nil {
		http.Error(w, "unable to create upstream request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("dbapi /query failed: %v", err)
		http.Error(w, "unable to retrieve data from dbapi", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "unable to copy response body", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(resBody)
}

// CRUD gateway: /v1/cars → dbapi /cars
func CarsRes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/cars")
	path = strings.Trim(path, "/")

	upstream := "/cars"
	if path != "" {
		upstream = "/cars/" + path
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		proxyToDBAPI(w, r, upstream)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func StaticRes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method must be GET", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/static" {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && r.Header.Get("Accept") != "application/json" {
		// Allow clients that set Accept instead of Content-Type on GET
		if !strings.Contains(r.Header.Get("Accept"), "application/json") {
			http.Error(w, "content type not supported", http.StatusBadRequest)
			return
		}
	}

	answer := CarInfoAnswer{
		Name:         "Example",
		Year:         1999,
		Condition:    "New",
		Reason:       "Static test",
		Module:       "Spider",
		Manufacture:  "Alfa Romeo",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(answer)
}

func main() {
	portnum := getEnv("PORT", "8080")
	http.HandleFunc("/static", StaticRes)
	http.HandleFunc("/v1", QueryRes)
	http.HandleFunc("/v1/cars", CarsRes)
	http.HandleFunc("/v1/cars/", CarsRes)
	log.Printf("Starting app-api on port %s (dbapi=%s)", portnum, dbapiBase())
	log.Fatal(http.ListenAndServe(":"+portnum, nil))
}
