package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/config"
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

func TestCLISyncItemsStorageStatusAndConfigFormat(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()

	out, err := executeTestCommand(t, "add", "rss", server.URL, "--id", "example", "--name", "Example")
	if err != nil {
		t.Fatalf("add rss: %v", err)
	}
	if !strings.Contains(out, "Created RSS source example") {
		t.Fatalf("bad add output: %s", out)
	}

	out, err = executeTestCommand(t, "sync")
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if !strings.Contains(out, "example: ok new=1") {
		t.Fatalf("bad sync output: %s", out)
	}

	out, err = executeTestCommand(t, "--json", "sync", "--source", "example")
	if err != nil {
		t.Fatalf("sync json: %v", err)
	}
	if !strings.Contains(out, `"action": "sync"`) || !strings.Contains(out, `"unchanged_items": 1`) {
		t.Fatalf("bad sync json output: %s", out)
	}

	out, err = executeTestCommand(t, "items", "list", "--unread")
	if err != nil {
		t.Fatalf("items list unread: %v", err)
	}
	if !strings.Contains(out, "First") || !strings.Contains(out, "read=false") {
		t.Fatalf("bad items output: %s", out)
	}
	itemID := firstField(t, out)

	out, err = executeTestCommand(t, "items", "markdown", itemID)
	if err != nil {
		t.Fatalf("items markdown: %v", err)
	}
	if !strings.Contains(out, ".md") {
		t.Fatalf("bad markdown output: %s", out)
	}

	out, err = executeTestCommand(t, "items", "open", itemID)
	if err != nil {
		t.Fatalf("items open: %v", err)
	}
	if !strings.Contains(out, "Opened "+itemID) {
		t.Fatalf("bad open output: %s", out)
	}

	out, err = executeTestCommand(t, "items", "list", "--json")
	if err != nil {
		t.Fatalf("items list json: %v", err)
	}
	if !strings.Contains(out, `"action": "items_list"`) || !strings.Contains(out, itemID) {
		t.Fatalf("bad items json output: %s", out)
	}

	out, err = executeTestCommand(t, "storage")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	if !strings.Contains(out, "Items: 1") || !strings.Contains(out, "Total:") {
		t.Fatalf("bad storage output: %s", out)
	}

	out, err = executeTestCommand(t, "storage", "reconcile", "--json")
	if err != nil {
		t.Fatalf("storage reconcile json: %v", err)
	}
	if !strings.Contains(out, `"action": "storage_reconcile"`) {
		t.Fatalf("bad reconcile json output: %s", out)
	}

	out, err = executeTestCommand(t, "status")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !strings.Contains(out, "1 unread") || !strings.Contains(out, "src:1") {
		t.Fatalf("bad status output: %s", out)
	}

	out, err = executeTestCommand(t, "config", "format", "--yes")
	if err != nil {
		t.Fatalf("config format: %v", err)
	}
	if !strings.Contains(out, "Config formatted") {
		t.Fatalf("bad format output: %s", out)
	}
}

func TestCLIOutputHelpers(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil)=%d", got)
	}
	if got := exitCode(app.AppError("item-not-found", "missing", nil)); got != 4 {
		t.Fatalf("item-not-found exit=%d", got)
	}
	if got := exitCode(app.AppError("invalid-url", "bad", nil)); got != 2 {
		t.Fatalf("invalid-url exit=%d", got)
	}
	if got := exitCode(app.AppError("unsupported-source-type", "bad", nil)); got != 3 {
		t.Fatalf("unsupported-source-type exit=%d", got)
	}
	if got := exitCode(errors.New("boom")); got != 1 {
		t.Fatalf("generic exit=%d", got)
	}

	var out bytes.Buffer
	if err := printLines(&out, "one", "two"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "one\ntwo\n" {
		t.Fatalf("printLines=%q", got)
	}
	if got := plainBool(true); got != "true" {
		t.Fatalf("plainBool(true)=%q", got)
	}
	if got := plainBool(false); got != "false" {
		t.Fatalf("plainBool(false)=%q", got)
	}
	assertTags(t, splitTags(" tech, ,rss "), []string{"tech", "rss"})
	assertTags(t, splitTags("  "), nil)

	out.Reset()
	if err := writeError(&out, config.ValidationError{Code: "bad", Message: "bad field", Path: "file", Field: "id"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"code": "bad"`) || !strings.Contains(out.String(), `"field": "id"`) {
		t.Fatalf("bad validation error json: %s", out.String())
	}
	out.Reset()
	if err := writeError(&out, config.ValidationErrors{{Code: "first", Message: "one"}, {Code: "second", Message: "two"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"code": "first"`) || !strings.Contains(out.String(), `"code": "second"`) {
		t.Fatalf("bad validation errors json: %s", out.String())
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

func firstField(t *testing.T, output string) string {
	t.Helper()
	fields := strings.Fields(output)
	if len(fields) == 0 {
		t.Fatalf("no fields in output %q", output)
	}
	return fields[0]
}

func assertTags(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tags=%v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags=%v want %v", got, want)
		}
	}
}
