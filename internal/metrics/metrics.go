package metrics

import "context"

type ItemMetrics struct {
	Provider       string `json:"provider"`
	Score          *int   `json:"score,omitempty"`
	CommentsCount  *int   `json:"comments_count,omitempty"`
	VotesCount     *int   `json:"votes_count,omitempty"`
	FavoritesCount *int   `json:"favorites_count,omitempty"`
	ReadingCount   *int   `json:"reading_count,omitempty"`
	FetchedAt      string `json:"fetched_at"`
}

type Candidate struct {
	SourceID     string
	SourceType   string
	Title        string
	URL          string
	CanonicalURL string
}

type Provider interface {
	Name() string
	Match(Candidate) bool
	Fetch(ctx context.Context, candidate Candidate) (ItemMetrics, error)
}

type Enricher struct {
	Providers []Provider
}

func DefaultEnricher() *Enricher {
	return &Enricher{Providers: []Provider{NewHabrProvider()}}
}

func (e *Enricher) Fetch(ctx context.Context, candidate Candidate) (ItemMetrics, bool, error) {
	if e == nil {
		return ItemMetrics{}, false, nil
	}
	for _, provider := range e.Providers {
		if provider == nil || !provider.Match(candidate) {
			continue
		}
		m, err := provider.Fetch(ctx, candidate)
		return m, true, err
	}
	return ItemMetrics{}, false, nil
}
