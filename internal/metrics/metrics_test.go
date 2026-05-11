package metrics

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	name    string
	match   bool
	metrics ItemMetrics
	err     error
	calls   int
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) Match(Candidate) bool { return p.match }

func (p *fakeProvider) Fetch(context.Context, Candidate) (ItemMetrics, error) {
	p.calls++
	return p.metrics, p.err
}

func TestHabrProviderDefaultsNameAndMatch(t *testing.T) {
	provider := NewHabrProvider()
	if provider.Name() != HabrProviderName || provider.BaseURL == "" || provider.Client == nil {
		t.Fatalf("provider=%#v", provider)
	}
	if !provider.Match(Candidate{URL: "https://habr.com/ru/articles/1033808/"}) {
		t.Fatal("provider should match Habr article URL")
	}
	if !provider.Match(Candidate{CanonicalURL: "https://habr.com/ru/articles/1033808/", URL: "https://example.com/copy"}) {
		t.Fatal("provider should prefer canonical Habr article URL")
	}
	if provider.Match(Candidate{URL: "https://example.com/articles/1033808/"}) {
		t.Fatal("provider should not match non-Habr URL")
	}
}

func TestEnricherFetchSelectsFirstMatchingProvider(t *testing.T) {
	first := &fakeProvider{name: "first", match: false, metrics: ItemMetrics{Provider: "first"}}
	second := &fakeProvider{name: "second", match: true, metrics: ItemMetrics{Provider: "second", FetchedAt: "now"}}
	third := &fakeProvider{name: "third", match: true, metrics: ItemMetrics{Provider: "third"}}
	enricher := &Enricher{Providers: []Provider{nil, first, second, third}}

	got, ok, err := enricher.Fetch(context.Background(), Candidate{URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || got.Provider != "second" || got.FetchedAt != "now" {
		t.Fatalf("got=%#v ok=%v", got, ok)
	}
	if first.calls != 0 || second.calls != 1 || third.calls != 0 {
		t.Fatalf("calls first=%d second=%d third=%d", first.calls, second.calls, third.calls)
	}
}

func TestEnricherPassesCallerContextToProvider(t *testing.T) {
	provider := &contextProvider{match: true}
	enricher := &Enricher{Providers: []Provider{provider}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, ok, err := enricher.Fetch(ctx, Candidate{URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("provider did not match")
	}
	if !errors.Is(provider.ctxErr, context.Canceled) {
		t.Fatalf("provider ctxErr=%v want context.Canceled", provider.ctxErr)
	}
}

type contextProvider struct {
	match  bool
	ctxErr error
}

func (p *contextProvider) Name() string { return "context" }
func (p *contextProvider) Match(candidate Candidate) bool {
	_ = candidate
	return p.match
}
func (p *contextProvider) Fetch(ctx context.Context, candidate Candidate) (ItemMetrics, error) {
	_ = candidate
	p.ctxErr = ctx.Err()
	return ItemMetrics{Provider: p.Name(), FetchedAt: "now"}, nil
}

func TestEnricherFetchNoMatchNilAndError(t *testing.T) {
	if got, ok, err := (*Enricher)(nil).Fetch(context.Background(), Candidate{}); err != nil || ok || got.Provider != "" {
		t.Fatalf("nil enricher got=%#v ok=%v err=%v", got, ok, err)
	}

	enricher := &Enricher{Providers: []Provider{&fakeProvider{name: "miss", match: false}}}
	if got, ok, err := enricher.Fetch(context.Background(), Candidate{}); err != nil || ok || got.Provider != "" {
		t.Fatalf("no match got=%#v ok=%v err=%v", got, ok, err)
	}

	boom := errors.New("boom")
	enricher = &Enricher{Providers: []Provider{&fakeProvider{name: "bad", match: true, err: boom}}}
	_, ok, err := enricher.Fetch(context.Background(), Candidate{})
	if !ok || !errors.Is(err, boom) {
		t.Fatalf("error ok=%v err=%v", ok, err)
	}
}

func TestDefaultEnricherIncludesHabrProvider(t *testing.T) {
	enricher := DefaultEnricher()
	if enricher == nil || len(enricher.Providers) != 1 || enricher.Providers[0].Name() != HabrProviderName {
		t.Fatalf("default enricher=%#v", enricher)
	}
}
