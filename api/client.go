package api

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.clashofclans.com/v1/"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	token      string
)

func Init(bearerToken string) {
	token = bearerToken
}

func formatTag(tag string) string {
	if tag == "" {
		return ""
	}
	if tag[0] == '#' {
		return url.PathEscape(tag)
	}
	return url.PathEscape("#") + tag
}

func get(rawURL string) []byte {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return data
}
