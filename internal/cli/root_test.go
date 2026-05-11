package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"feedctl/internal/testutil"
)

func TestCLIConfigSourceAndJSONError(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()

	out, err := executeTestCommand(t, "--json", "config", "path")
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if !strings.Contains(out, "\"ok\": true") || !strings.Contains(out, "config_path") {
		t.Fatalf("bad config path output: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "config", "validate")
	if err != nil {
		t.Fatalf("config validate: %v", err)
	}
	if !strings.Contains(out, "\"valid\": true") {
		t.Fatalf("bad validate output: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "add", "rss", server.URL, "--id", "example", "--name", "Example", "--tags", "tech,blog", "--dry-run")
	if err != nil {
		t.Fatalf("add dry-run: %v", err)
	}
	if !strings.Contains(out, "\"dry_run\": true") || !strings.Contains(out, "example") {
		t.Fatalf("bad add dry-run: %s", out)
	}

	out, err = executeTestCommand(t, "add", "rss", server.URL, "--id", "example", "--name", "Example", "--tags", "tech,blog")
	if err != nil {
		t.Fatalf("add apply: %v", err)
	}
	if !strings.Contains(out, "Created RSS source example") {
		t.Fatalf("bad add apply: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "list")
	if err != nil {
		t.Fatalf("sources list: %v", err)
	}
	if !strings.Contains(out, "example") || !strings.Contains(out, "active") {
		t.Fatalf("bad list: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "show", "example")
	if err != nil {
		t.Fatalf("sources show: %v", err)
	}
	if !strings.Contains(out, "id: example") {
		t.Fatalf("bad show: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "test", "example")
	if err != nil {
		t.Fatalf("sources test: %v", err)
	}
	if !strings.Contains(out, "items=1") {
		t.Fatalf("bad test: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "disable", "example", "--yes")
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if !strings.Contains(out, "Disabled source example") {
		t.Fatalf("bad disable: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "enable", "example", "--yes")
	if err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !strings.Contains(out, "Enabled source example") {
		t.Fatalf("bad enable: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "sources", "remove", "example", "--dry-run")
	if err != nil {
		t.Fatalf("remove dry-run: %v", err)
	}
	if !strings.Contains(out, "remove_source") || !strings.Contains(out, "\"runtime_kept\": true") {
		t.Fatalf("bad remove dry-run: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "add", "html", server.URL)
	if err == nil {
		t.Fatalf("expected unsupported source error")
	}
	if !strings.Contains(out, "unsupported-source-type") {
		t.Fatalf("bad json error: %s", out)
	}
}

func TestCLIAddTelegram(t *testing.T) {
	testutil.IsolatedEnv(t)
	page := testutil.TelegramWebPage("LLM под капотом", "llm_under_hood", []testutil.TelegramPost{
		testutil.DefaultTelegramPost(831, `<b>Post 831</b><br/>Body`),
	}, "")
	server := testutil.TelegramServer(t, "llm_under_hood", map[string]string{"/s/llm_under_hood": page})

	out, err := executeTestCommand(t, "--json", "add", "telegram", server.URL+"/s/llm_under_hood", "--id", "tg-llm", "--name", "LLM под капотом", "--tags", "telegram,llm", "--max-items", "25", "--dry-run")
	if err != nil {
		t.Fatalf("add telegram dry-run: %v", err)
	}
	if !strings.Contains(out, `"source_type": "telegram"`) || !strings.Contains(out, `"canonical_url": "`+server.URL+`/s/llm_under_hood"`) || !strings.Contains(out, `"dry_run": true`) {
		t.Fatalf("bad add telegram dry-run output: %s", out)
	}

	out, err = executeTestCommand(t, "add", "telegram", server.URL+"/s/llm_under_hood", "--id", "tg-llm", "--tags", "telegram", "--max-items", "25")
	if err != nil {
		t.Fatalf("add telegram: %v", err)
	}
	if !strings.Contains(out, "Created Telegram source tg-llm") {
		t.Fatalf("bad add telegram output: %s", out)
	}

	out, err = executeTestCommand(t, "sources", "test", "tg-llm")
	if err != nil {
		t.Fatalf("sources test telegram: %v", err)
	}
	if !strings.Contains(out, "ok: tg-llm items=1") || !strings.Contains(out, "LLM под капотом") {
		t.Fatalf("bad telegram source test output: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "sources", "test", "tg-llm")
	if err != nil {
		t.Fatalf("sources test telegram json: %v", err)
	}
	if !strings.Contains(out, `"items_found": 1`) || !strings.Contains(out, `"title": "LLM под капотом"`) {
		t.Fatalf("bad telegram source test json output: %s", out)
	}
}

func executeTestCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	opts := &options{}
	cmd := newRootCommand(opts)
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.ExecuteContext(context.Background())
	if err != nil && opts.JSON {
		_ = writeError(&out, err)
	}
	return out.String(), err
}
