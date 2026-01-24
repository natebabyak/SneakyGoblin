package api

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

var (
	token string
)

func init() {
	godotenv.Load()
	token = os.Getenv("API_TOKEN")
}

const baseUrl = "https://api.clashofclans.com/v1/players/"

func get(endpoint string) any {
	url := baseUrl + endpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("%d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return body
}
