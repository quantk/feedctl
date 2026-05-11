package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	HabrProviderName = "habr"
	habrBaseURL      = "https://habr.com"
)

var habrArticlePathPattern = regexp.MustCompile(`(?:^|/)articles/(\d+)(?:/|$)`)

type HabrProvider struct {
	BaseURL string
	Client  *http.Client
}

func NewHabrProvider() HabrProvider {
	return HabrProvider{BaseURL: habrBaseURL, Client: http.DefaultClient}
}

func (p HabrProvider) Name() string { return HabrProviderName }

func (p HabrProvider) Match(candidate Candidate) bool {
	_, ok := habrArticleID(candidate)
	return ok
}

func (p HabrProvider) Fetch(ctx context.Context, candidate Candidate) (ItemMetrics, error) {
	articleID, ok := habrArticleID(candidate)
	if !ok {
		return ItemMetrics{}, fmt.Errorf("habr article id not found")
	}
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	base := strings.TrimRight(p.BaseURL, "/")
	if base == "" {
		base = habrBaseURL
	}
	endpoint := fmt.Sprintf("%s/kek/v2/articles/%s/", base, articleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ItemMetrics{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "feedctl")
	resp, err := client.Do(req)
	if err != nil {
		return ItemMetrics{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ItemMetrics{}, fmt.Errorf("habr metrics: %s", resp.Status)
	}
	var payload struct {
		Statistics struct {
			CommentsCount  *int `json:"commentsCount"`
			FavoritesCount *int `json:"favoritesCount"`
			ReadingCount   *int `json:"readingCount"`
			Score          *int `json:"score"`
			VotesCount     *int `json:"votesCount"`
		} `json:"statistics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ItemMetrics{}, err
	}
	return ItemMetrics{
		Provider:       HabrProviderName,
		Score:          payload.Statistics.Score,
		CommentsCount:  payload.Statistics.CommentsCount,
		VotesCount:     payload.Statistics.VotesCount,
		FavoritesCount: payload.Statistics.FavoritesCount,
		ReadingCount:   payload.Statistics.ReadingCount,
		FetchedAt:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func ExtractHabrArticleID(rawURL string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host != "habr.com" && !strings.HasSuffix(host, ".habr.com") {
		return "", false
	}
	matches := habrArticlePathPattern.FindStringSubmatch(u.EscapedPath())
	if len(matches) != 2 {
		return "", false
	}
	return matches[1], true
}

func habrArticleID(candidate Candidate) (string, bool) {
	if id, ok := ExtractHabrArticleID(candidate.CanonicalURL); ok {
		return id, true
	}
	return ExtractHabrArticleID(candidate.URL)
}
