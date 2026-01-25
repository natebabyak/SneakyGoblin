package bot

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

var (
	client http.Client = http.Client{
		Timeout: 10 * time.Second,
	}
	token string
)

func init() {
	godotenv.Load()
	token = os.Getenv("API_TOKEN")

}

func formatTag(tag string) string {
	if tag[0] == '#' {
		return url.PathEscape(tag)
	} else {
		return url.PathEscape("#") + tag
	}
}

func Get(url string) any {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
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

const baseUrl = "https://api.clashofclans.com/v1/"

func GetClanByTag(tag string) any {
	return Get(baseUrl + formatTag(tag))
}

func GetPlayerByTag(tag string) Player {
	return Get(baseUrl + formatTag(tag)).(Player)
}
