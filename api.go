package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.clashofclans.com/v1/"

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
)

type APIResult[T any] struct {
	Data       T
	OK         bool
	StatusCode int
	Error      string
}

func normalizeTag(raw string) string {
	tag := strings.ToUpper(strings.TrimSpace(raw))
	if tag == "" {
		return ""
	}
	if strings.HasPrefix(tag, "#") {
		return tag
	}
	return "#" + tag
}

func formatTag(tag string) string {
	normalized := normalizeTag(tag)
	if normalized == "" {
		return ""
	}
	return url.PathEscape(normalized)
}

func get(rawURL string) ([]byte, int, string, bool) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		log.Println("failed to build request:", err)
		return nil, 0, "failed to build request", false
	}

	if cocToken == "" {
		log.Println("COC_TOKEN is empty; set it in .env for Clash API requests")
		return nil, 0, "COC token is not configured", false
	}
	req.Header.Set("Authorization", "Bearer "+cocToken)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Println("request failed:", err)
		return nil, 0, "failed to reach Clash API", false
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("failed to read response body:", err)
		return nil, resp.StatusCode, "failed to read Clash API response", false
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("non-2xx status %d from %s", resp.StatusCode, rawURL)
		return nil, resp.StatusCode, parseAPIError(data), false
	}

	return data, resp.StatusCode, "", true
}

func parseAPIError(data []byte) string {
	var e struct {
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &e); err == nil {
		if e.Message != "" {
			return e.Message
		}
		if e.Reason != "" {
			return e.Reason
		}
	}
	return "Clash API rejected the request"
}

func getClanByTag(tag string) APIResult[Clan] {
	data, statusCode, reason, ok := get(baseURL + "clans/" + formatTag(tag))
	if !ok {
		return APIResult[Clan]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var clan Clan
	if err := json.Unmarshal(data, &clan); err != nil {
		log.Println("failed to parse clan response:", err)
		return APIResult[Clan]{OK: false, StatusCode: statusCode, Error: "failed to parse clan response"}
	}
	return APIResult[Clan]{Data: clan, OK: true, StatusCode: statusCode}
}

func getPlayerByTag(tag string) APIResult[Player] {
	data, statusCode, reason, ok := get(baseURL + "players/" + formatTag(tag))
	if !ok {
		return APIResult[Player]{OK: false, StatusCode: statusCode, Error: reason}
	}

	var player Player
	if err := json.Unmarshal(data, &player); err != nil {
		log.Println("failed to parse player response:", err)
		return APIResult[Player]{OK: false, StatusCode: statusCode, Error: "failed to parse player response"}
	}
	return APIResult[Player]{Data: player, OK: true, StatusCode: statusCode}
}
