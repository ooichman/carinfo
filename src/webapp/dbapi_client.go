package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type Car struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Year        int    `json:"year"`
	Condition   string `json:"condition"`
	Reason      string `json:"reason"`
	Module      string `json:"module"`
	Manufacture string `json:"manufacture"`
}

type DBAPIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewDBAPIClient(baseURL string) *DBAPIClient {
	return &DBAPIClient{
		BaseURL: stringsTrimRightSlash(baseURL),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func stringsTrimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func (c *DBAPIClient) doJSON(method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("dbapi %s %s: %s (%s)", method, path, resp.Status, string(data))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *DBAPIClient) ListCars() ([]Car, error) {
	var cars []Car
	err := c.doJSON(http.MethodGet, "/cars", nil, &cars)
	if cars == nil {
		cars = []Car{}
	}
	return cars, err
}

func (c *DBAPIClient) GetCar(id int) (Car, error) {
	var car Car
	err := c.doJSON(http.MethodGet, "/cars/"+strconv.Itoa(id), nil, &car)
	return car, err
}

func (c *DBAPIClient) CreateCar(car Car) (Car, error) {
	var out Car
	err := c.doJSON(http.MethodPost, "/cars", car, &out)
	return out, err
}

func (c *DBAPIClient) UpdateCar(id int, car Car) (Car, error) {
	var out Car
	err := c.doJSON(http.MethodPut, "/cars/"+strconv.Itoa(id), car, &out)
	return out, err
}

func (c *DBAPIClient) DeleteCar(id int) error {
	return c.doJSON(http.MethodDelete, "/cars/"+strconv.Itoa(id), nil, nil)
}
