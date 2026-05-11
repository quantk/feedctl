package app_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/testutil"
)

func TestAddTelegramDryRun(t *testing.T) {
	testutil.IsolatedEnv(t)
	server := testTelegramServer(t, "llm_under_hood", 1)

	res, err := app.AddTelegram(context.Background(), server.URL+"/s/llm_under_hood", app.AddTelegramParams{ID: "tg-llm", Name: "LLM под капотом", Tags: []string{"telegram", "llm"}, MaxItems: 25, DryRun: true})
	if err != nil {
		t.Fatalf("add telegram dry-run: %v", err)
	}
	if !res.DryRun || res.SourceID != "tg-llm" || res.SourceType != config.SourceTypeTelegram || res.CanonicalURL != server.URL+"/s/llm_under_hood" || res.ItemsFound != 1 {
		t.Fatalf("bad dry-run result: %#v", res)
	}
	if _, err := os.Stat(res.ConfigPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run wrote config path err=%v", err)
	}
}

func TestAddTelegramWritesSourceAndGeneratedID(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	server := testTelegramServer(t, "llm_under_hood", 2)

	res, err := app.AddTelegram(context.Background(), server.URL+"/s/llm_under_hood", app.AddTelegramParams{Tags: []string{"telegram"}, MaxItems: 50})
	if err != nil {
		t.Fatalf("add telegram: %v", err)
	}
	if res.SourceID != "llm_under_hood" || res.ItemsFound != 2 || res.CanonicalURL != server.URL+"/s/llm_under_hood" {
		t.Fatalf("bad add result: %#v", res)
	}

	path := filepath.Join(configDir, "sources.d", res.SourceID+".toml")
	loaded, err := config.LoadSources(filepath.Join(configDir, "sources.d"))
	if err != nil {
		t.Fatalf("load sources: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("sources len=%d", len(loaded))
	}
	src := loaded[0]
	if src.FilePath != path || src.Type != config.SourceTypeTelegram || src.URL != server.URL+"/s/llm_under_hood" || src.MaxItems != 50 || len(src.Tags) != 1 || src.Tags[0] != "telegram" {
		t.Fatalf("bad written source: %#v", src)
	}
}

func TestAddTelegramRejectsExistingSourceID(t *testing.T) {
	testutil.IsolatedEnv(t)
	server := testTelegramServer(t, "llm_under_hood", 1)
	params := app.AddTelegramParams{ID: "tg-llm"}
	if _, err := app.AddTelegram(context.Background(), server.URL+"/s/llm_under_hood", params); err != nil {
		t.Fatalf("first add telegram: %v", err)
	}

	_, err := app.AddTelegram(context.Background(), server.URL+"/s/llm_under_hood", params)
	if err == nil {
		t.Fatal("expected source conflict")
	}
	var appErr app.Error
	if !errors.As(err, &appErr) || appErr.Code != "source-already-exists" {
		t.Fatalf("expected source-already-exists, got %T %v", err, err)
	}
}

func testTelegramServer(t *testing.T, channel string, posts int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/s/"+channel {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, telegramHTML(channel, posts))
	}))
	t.Cleanup(server.Close)
	return server
}

func telegramHTML(channel string, posts int) string {
	body := "<!doctype html><html><head><title>LLM под капотом – Telegram</title></head><body>"
	for i := 1; i <= posts; i++ {
		body += fmt.Sprintf(`<div class="tgme_widget_message_wrap js-widget_message_wrap"><div class="tgme_widget_message" data-post="%s/%d"><div class="tgme_widget_message_author"><span class="tgme_widget_message_owner_name">LLM под капотом</span></div><div class="tgme_widget_message_text js-message_text" dir="auto"><b>Post %d</b><br/>Body %d</div><a class="tgme_widget_message_date" href="https://t.me/%s/%d"><time datetime="2026-05-11T15:%02d:00+00:00"></time></a><span class="tgme_widget_message_views">1K</span></div></div>`, channel, i, i, i, channel, i, i)
	}
	return body + "</body></html>"
}
