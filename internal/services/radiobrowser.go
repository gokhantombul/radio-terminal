package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OnlineStation struct {
	UUID    string `json:"stationuuid"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Tags    string `json:"tags"`
	URL     string `json:"url_resolved"`
	Bitrate int    `json:"bitrate"`
	Votes   int    `json:"votes"`
	Codec   string `json:"codec"`
}

func (s OnlineStation) GenreDisplay() string {
	if strings.TrimSpace(s.Tags) == "" {
		return "Çeşitli"
	}
	parts := strings.Split(s.Tags, ",")
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return "Çeşitli"
	}
	return first
}

func (s OnlineStation) CountryDisplay() string {
	if strings.TrimSpace(s.Country) == "" {
		return "Bilinmiyor"
	}
	return s.Country
}

type RadioBrowserService struct {
	apiBase string
}

func NewRadioBrowserService() *RadioBrowserService {
	return &RadioBrowserService{
		apiBase: "https://de1.api.radio-browser.info/json",
	}
}

func (r *RadioBrowserService) Search(query, country, tag string, limit int) ([]OnlineStation, error) {
	u, _ := url.Parse(fmt.Sprintf("%s/stations/search", r.apiBase))
	q := u.Query()
	q.Set("hidebroken", "true")
	q.Set("order", "votes")
	q.Set("reverse", "true")

	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	if strings.TrimSpace(query) != "" {
		q.Set("name", query)
	}
	if strings.TrimSpace(country) != "" {
		q.Set("country", country)
	}
	if strings.TrimSpace(tag) != "" {
		q.Set("tag", tag)
	}
	u.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", u.String(), nil)
	req.Header.Set("User-Agent", "Radio Shell/2.0 (Go)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var stations []OnlineStation
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, err
	}

	var filtered []OnlineStation
	for _, s := range stations {
		if strings.HasPrefix(s.URL, "http") {
			filtered = append(filtered, s)
		}
	}

	return filtered, nil
}
