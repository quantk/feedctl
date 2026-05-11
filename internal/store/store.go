package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"feedctl/internal/config"
	"feedctl/internal/metrics"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("not found")

type DB struct {
	db *sql.DB
}

type Source struct {
	ID             string   `json:"id"`
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Lifecycle      string   `json:"lifecycle"`
	Enabled        bool     `json:"enabled"`
	Interval       string   `json:"interval,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	LastSyncAt     string   `json:"last_sync_at,omitempty"`
	LastSyncStatus string   `json:"last_sync_status,omitempty"`
	LastError      string   `json:"last_error,omitempty"`
	RemovedAt      string   `json:"removed_at,omitempty"`
}

type Item struct {
	ID              string               `json:"id"`
	SourceID        string               `json:"source_id"`
	SourceItemID    string               `json:"source_item_id"`
	IdentityKind    string               `json:"identity_kind"`
	Title           string               `json:"title"`
	URL             string               `json:"url"`
	CanonicalURL    string               `json:"canonical_url,omitempty"`
	PublishedAt     string               `json:"published_at,omitempty"`
	FetchedAt       string               `json:"fetched_at"`
	LastSeenAt      string               `json:"last_seen_at"`
	ContentPath     string               `json:"content_path"`
	ContentHash     string               `json:"content_hash"`
	Version         int                  `json:"version"`
	ReadAt          string               `json:"read_at,omitempty"`
	Starred         bool                 `json:"starred"`
	ArchivedAt      string               `json:"archived_at,omitempty"`
	UpdatedAt       string               `json:"updated_at,omitempty"`
	SourceLifecycle string               `json:"source_lifecycle,omitempty"`
	Tags            []string             `json:"tags,omitempty"`
	Metrics         *metrics.ItemMetrics `json:"metrics,omitempty"`
}

type ItemVersion struct {
	ID          string `json:"id"`
	ItemID      string `json:"item_id"`
	Version     int    `json:"version"`
	ContentPath string `json:"content_path"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	SizeBytes   int64  `json:"size_bytes"`
}

type ItemFilter struct {
	Unread         bool
	RemovedSources bool
	AllItems       bool
	Starred        bool
	SourceID       string
}

type StorageStats struct {
	ItemsCount           int    `json:"items_count"`
	CurrentMarkdownBytes int64  `json:"current_markdown_bytes"`
	VersionsBytes        int64  `json:"versions_bytes"`
	DatabaseBytes        int64  `json:"database_bytes"`
	TotalBytes           int64  `json:"total_bytes"`
	UpdatedAt            string `json:"updated_at,omitempty"`
}

type StatusSummary struct {
	UnreadCount        int          `json:"unread_count"`
	SourceCount        int          `json:"source_count"`
	RemovedSourceCount int          `json:"removed_source_count"`
	Storage            StorageStats `json:"storage"`
	LatestSyncStatus   string       `json:"latest_sync_status"`
	LatestSyncAt       string       `json:"latest_sync_at,omitempty"`
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	wrapped := &DB{db: db}
	if err := wrapped.Migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return wrapped, nil
}

func (d *DB) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

type migration struct {
	version int
	up      func(*sql.Tx) error
}

var migrations = []migration{
	{version: 1, up: func(tx *sql.Tx) error {
		for _, stmt := range initialSchemaStatements {
			if _, err := tx.Exec(stmt); err != nil {
				return err
			}
		}
		return nil
	}},
}

var initialSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS runtime_sources (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		lifecycle TEXT NOT NULL,
		enabled INTEGER NOT NULL,
		interval TEXT NOT NULL DEFAULT '',
		tags_json TEXT NOT NULL DEFAULT '[]',
		last_sync_at TEXT NOT NULL DEFAULT '',
		last_sync_status TEXT NOT NULL DEFAULT '',
		last_error TEXT NOT NULL DEFAULT '',
		etag TEXT NOT NULL DEFAULT '',
		last_modified TEXT NOT NULL DEFAULT '',
		removed_at TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS items (
		id TEXT PRIMARY KEY,
		source_id TEXT NOT NULL,
		source_item_id TEXT NOT NULL,
		identity_kind TEXT NOT NULL,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		canonical_url TEXT NOT NULL DEFAULT '',
		published_at TEXT NOT NULL DEFAULT '',
		fetched_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		content_path TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		version INTEGER NOT NULL,
		read_at TEXT NOT NULL DEFAULT '',
		starred INTEGER NOT NULL DEFAULT 0,
		archived_at TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL DEFAULT '',
		tags_json TEXT NOT NULL DEFAULT '[]',
		UNIQUE(source_id, source_item_id),
		FOREIGN KEY(source_id) REFERENCES runtime_sources(id)
	)`,
	`CREATE TABLE IF NOT EXISTS item_metrics (
		item_id TEXT PRIMARY KEY,
		provider TEXT NOT NULL,
		fetched_at TEXT NOT NULL,
		metrics_json TEXT NOT NULL,
		FOREIGN KEY(item_id) REFERENCES items(id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_items_source ON items(source_id)`,
	`CREATE INDEX IF NOT EXISTS idx_items_content_path ON items(content_path)`,
	`CREATE INDEX IF NOT EXISTS idx_items_read ON items(read_at)`,
	`CREATE TABLE IF NOT EXISTS item_versions (
		id TEXT PRIMARY KEY,
		item_id TEXT NOT NULL,
		version INTEGER NOT NULL,
		content_path TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		created_at TEXT NOT NULL,
		size_bytes INTEGER NOT NULL,
		UNIQUE(item_id, version),
		FOREIGN KEY(item_id) REFERENCES items(id)
	)`,
	`CREATE TABLE IF NOT EXISTS storage_stats (
		scope TEXT PRIMARY KEY,
		bytes INTEGER NOT NULL,
		item_count INTEGER NOT NULL,
		updated_at TEXT NOT NULL
	)`,
}

func (d *DB) Migrate() error {
	if _, err := d.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	return d.applyMigrations(migrations)
}

func (d *DB) applyMigrations(steps []migration) error {
	if _, err := d.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	applied, err := d.appliedMigrations()
	if err != nil {
		return err
	}
	ordered := append([]migration(nil), steps...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].version < ordered[j].version })
	for _, step := range ordered {
		if _, ok := applied[step.version]; ok {
			continue
		}
		if err := d.applyMigration(step); err != nil {
			return err
		}
		applied[step.version] = struct{}{}
	}
	return nil
}

func (d *DB) appliedMigrations() (map[int]struct{}, error) {
	rows, err := d.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[int]struct{}{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = struct{}{}
	}
	return applied, rows.Err()
}

func (d *DB) applyMigration(step migration) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if step.up != nil {
		if err := step.up(tx); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES (?, datetime('now'))`, step.version); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (d *DB) UpsertConfiguredSource(src config.Source) error {
	tags, _ := json.Marshal(src.Tags)
	lifecycle := "disabled"
	if src.Enabled {
		lifecycle = "active"
	}
	now := nowString()
	_, err := d.db.Exec(`INSERT INTO runtime_sources
		(id, type, name, url, lifecycle, enabled, interval, tags_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type=excluded.type,
			name=excluded.name,
			url=excluded.url,
			lifecycle=excluded.lifecycle,
			enabled=excluded.enabled,
			interval=excluded.interval,
			tags_json=excluded.tags_json,
			removed_at='',
			updated_at=excluded.updated_at`,
		src.ID, src.Type, src.Name, src.URL, lifecycle, boolInt(src.Enabled), src.Interval, string(tags), now, now)
	return err
}

func (d *DB) MarkRemovedExcept(activeIDs map[string]struct{}) error {
	rows, err := d.db.Query(`SELECT id FROM runtime_sources`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if _, ok := activeIDs[id]; !ok {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := nowString()
	for _, id := range ids {
		if _, err := d.db.Exec(`UPDATE runtime_sources SET lifecycle='removed', enabled=0, removed_at=CASE WHEN removed_at='' THEN ? ELSE removed_at END, updated_at=? WHERE id=?`, now, now, id); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ListSources(includeRemoved bool) ([]Source, error) {
	query := `SELECT id, type, name, url, lifecycle, enabled, interval, tags_json, last_sync_at, last_sync_status, last_error, removed_at FROM runtime_sources`
	if !includeRemoved {
		query += ` WHERE lifecycle <> 'removed'`
	}
	query += ` ORDER BY id`
	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (d *DB) GetSource(id string) (Source, error) {
	row := d.db.QueryRow(`SELECT id, type, name, url, lifecycle, enabled, interval, tags_json, last_sync_at, last_sync_status, last_error, removed_at FROM runtime_sources WHERE id=?`, id)
	s, err := scanSource(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	return s, err
}

func (d *DB) UpdateSourceSyncStatus(id, status, message string, newItems, updatedItems, unchangedItems int) error {
	_ = newItems
	_ = updatedItems
	_ = unchangedItems
	_, err := d.db.Exec(`UPDATE runtime_sources SET last_sync_at=?, last_sync_status=?, last_error=?, updated_at=? WHERE id=?`, nowString(), status, message, nowString(), id)
	return err
}

func (d *DB) CreateItem(item Item) error {
	if item.FetchedAt == "" {
		item.FetchedAt = nowString()
	}
	if item.LastSeenAt == "" {
		item.LastSeenAt = item.FetchedAt
	}
	if item.Version == 0 {
		item.Version = 1
	}
	tags, _ := json.Marshal(item.Tags)
	_, err := d.db.Exec(`INSERT INTO items
		(id, source_id, source_item_id, identity_kind, title, url, canonical_url, published_at, fetched_at, last_seen_at, content_path, content_hash, version, read_at, starred, archived_at, updated_at, tags_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.SourceID, item.SourceItemID, item.IdentityKind, item.Title, item.URL, item.CanonicalURL, item.PublishedAt, item.FetchedAt, item.LastSeenAt, item.ContentPath, item.ContentHash, item.Version, item.ReadAt, boolInt(item.Starred), item.ArchivedAt, item.UpdatedAt, string(tags))
	return err
}

func (d *DB) FindItemBySourceIdentity(sourceID, sourceItemID string) (Item, error) {
	row := d.db.QueryRow(itemSelect()+` WHERE i.source_id=? AND i.source_item_id=?`, sourceID, sourceItemID)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	return item, err
}

func (d *DB) GetItem(id string) (Item, error) {
	row := d.db.QueryRow(itemSelect()+` WHERE i.id=?`, id)
	item, err := scanItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	return item, err
}

func (d *DB) IsContentPathAssigned(path, exceptItemID string) (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM items WHERE content_path=? AND id<>?`, path, exceptItemID).Scan(&count)
	return count > 0, err
}

func (d *DB) UpdateItemSeen(id string) error {
	_, err := d.db.Exec(`UPDATE items SET last_seen_at=? WHERE id=?`, nowString(), id)
	return err
}

func (d *DB) UpdateItemChanged(item Item) error {
	_, err := d.db.Exec(`UPDATE items SET title=?, url=?, canonical_url=?, published_at=?, fetched_at=?, last_seen_at=?, content_path=?, content_hash=?, version=?, updated_at=?, tags_json=? WHERE id=?`,
		item.Title, item.URL, item.CanonicalURL, item.PublishedAt, item.FetchedAt, item.LastSeenAt, item.ContentPath, item.ContentHash, item.Version, item.UpdatedAt, tagsJSON(item.Tags), item.ID)
	return err
}

func (d *DB) UpsertItemMetrics(itemID string, itemMetrics metrics.ItemMetrics) error {
	if itemMetrics.FetchedAt == "" {
		itemMetrics.FetchedAt = nowString()
	}
	data, err := json.Marshal(itemMetrics)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO item_metrics(item_id, provider, fetched_at, metrics_json) VALUES (?, ?, ?, ?)
		ON CONFLICT(item_id) DO UPDATE SET provider=excluded.provider, fetched_at=excluded.fetched_at, metrics_json=excluded.metrics_json`,
		itemID, itemMetrics.Provider, itemMetrics.FetchedAt, string(data))
	return err
}

func (d *DB) AddItemVersion(v ItemVersion) error {
	_, err := d.db.Exec(`INSERT OR REPLACE INTO item_versions(id, item_id, version, content_path, content_hash, created_at, size_bytes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.ItemID, v.Version, v.ContentPath, v.ContentHash, v.CreatedAt, v.SizeBytes)
	return err
}

func (d *DB) AddItemVersionAndUpdateItemChanged(v ItemVersion, item Item) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.Exec(`INSERT OR REPLACE INTO item_versions(id, item_id, version, content_path, content_hash, created_at, size_bytes) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.ItemID, v.Version, v.ContentPath, v.ContentHash, v.CreatedAt, v.SizeBytes); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE items SET title=?, url=?, canonical_url=?, published_at=?, fetched_at=?, last_seen_at=?, content_path=?, content_hash=?, version=?, updated_at=?, tags_json=? WHERE id=?`,
		item.Title, item.URL, item.CanonicalURL, item.PublishedAt, item.FetchedAt, item.LastSeenAt, item.ContentPath, item.ContentHash, item.Version, item.UpdatedAt, tagsJSON(item.Tags), item.ID); err != nil {
		return err
	}
	committed = true
	return tx.Commit()
}

func (d *DB) ListItems(filter ItemFilter) ([]Item, error) {
	query := itemSelect()
	var clauses []string
	var args []any
	if filter.SourceID != "" {
		clauses = append(clauses, `i.source_id=?`)
		args = append(args, filter.SourceID)
	}
	if filter.Unread {
		clauses = append(clauses, `i.read_at=''`)
	}
	if filter.Starred {
		clauses = append(clauses, `i.starred=1`)
	}
	if filter.RemovedSources {
		clauses = append(clauses, `s.lifecycle='removed'`)
	} else if !filter.AllItems {
		clauses = append(clauses, `s.lifecycle<>'removed'`, `i.archived_at=''`)
	}
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, ` AND `)
	}
	query += ` ORDER BY COALESCE(NULLIF(i.published_at,''), i.fetched_at) DESC, i.id DESC`
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) SetRead(id string, read bool) error {
	value := ""
	if read {
		value = nowString()
	}
	res, err := d.db.Exec(`UPDATE items SET read_at=? WHERE id=?`, value, id)
	if err != nil {
		return err
	}
	return ensureAffected(res)
}

func (d *DB) SetStarred(id string, starred bool) error {
	res, err := d.db.Exec(`UPDATE items SET starred=? WHERE id=?`, boolInt(starred), id)
	if err != nil {
		return err
	}
	return ensureAffected(res)
}

func (d *DB) SetArchived(id string, archived bool) error {
	value := ""
	if archived {
		value = nowString()
	}
	res, err := d.db.Exec(`UPDATE items SET archived_at=? WHERE id=?`, value, id)
	if err != nil {
		return err
	}
	return ensureAffected(res)
}

func (d *DB) UpsertStorageStats(scope string, bytes int64, itemCount int) error {
	_, err := d.db.Exec(`INSERT INTO storage_stats(scope, bytes, item_count, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(scope) DO UPDATE SET bytes=excluded.bytes, item_count=excluded.item_count, updated_at=excluded.updated_at`, scope, bytes, itemCount, nowString())
	return err
}

func (d *DB) GetStorageStats(dbPath string) (StorageStats, error) {
	stats := StorageStats{}
	rows, err := d.db.Query(`SELECT scope, bytes, item_count, updated_at FROM storage_stats`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var scope, updated string
		var bytes int64
		var count int
		if err := rows.Scan(&scope, &bytes, &count, &updated); err != nil {
			return stats, err
		}
		stats.UpdatedAt = updated
		switch scope {
		case "current_markdown":
			stats.CurrentMarkdownBytes = bytes
			stats.ItemsCount = count
		case "versions":
			stats.VersionsBytes = bytes
		case "database":
			stats.DatabaseBytes = bytes
		}
	}
	if info, err := os.Stat(dbPath); err == nil {
		stats.DatabaseBytes = info.Size()
	}
	stats.TotalBytes = stats.CurrentMarkdownBytes + stats.VersionsBytes + stats.DatabaseBytes
	return stats, rows.Err()
}

func (d *DB) CountItems() (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM items`).Scan(&count)
	return count, err
}

func (d *DB) Status(dbPath string) (StatusSummary, error) {
	var s StatusSummary
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM items i JOIN runtime_sources rs ON rs.id=i.source_id WHERE i.read_at='' AND i.archived_at='' AND rs.lifecycle<>'removed'`).Scan(&s.UnreadCount); err != nil {
		return s, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM runtime_sources WHERE lifecycle<>'removed'`).Scan(&s.SourceCount); err != nil {
		return s, err
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM runtime_sources WHERE lifecycle='removed'`).Scan(&s.RemovedSourceCount); err != nil {
		return s, err
	}
	storage, err := d.GetStorageStats(dbPath)
	if err != nil {
		return s, err
	}
	s.Storage = storage
	row := d.db.QueryRow(`SELECT last_sync_status, last_sync_at FROM runtime_sources WHERE last_sync_at<>'' ORDER BY last_sync_at DESC LIMIT 1`)
	if err := row.Scan(&s.LatestSyncStatus, &s.LatestSyncAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return s, err
	}
	if s.LatestSyncStatus == "" {
		s.LatestSyncStatus = "never"
	}
	return s, nil
}

func (d *DB) Raw() *sql.DB { return d.db }

type scanner interface{ Scan(dest ...any) error }

func scanSource(s scanner) (Source, error) {
	var out Source
	var enabled int
	var tags string
	if err := s.Scan(&out.ID, &out.Type, &out.Name, &out.URL, &out.Lifecycle, &enabled, &out.Interval, &tags, &out.LastSyncAt, &out.LastSyncStatus, &out.LastError, &out.RemovedAt); err != nil {
		return out, err
	}
	out.Enabled = enabled == 1
	_ = json.Unmarshal([]byte(tags), &out.Tags)
	return out, nil
}

func itemSelect() string {
	return `SELECT i.id, i.source_id, i.source_item_id, i.identity_kind, i.title, i.url, i.canonical_url, i.published_at, i.fetched_at, i.last_seen_at, i.content_path, i.content_hash, i.version, i.read_at, i.starred, i.archived_at, i.updated_at, i.tags_json, s.lifecycle, im.provider, im.fetched_at, im.metrics_json FROM items i JOIN runtime_sources s ON s.id=i.source_id LEFT JOIN item_metrics im ON im.item_id=i.id`
}

func scanItem(s scanner) (Item, error) {
	var out Item
	var starred int
	var tags string
	var metricProvider sql.NullString
	var metricFetchedAt sql.NullString
	var metricJSON sql.NullString
	if err := s.Scan(&out.ID, &out.SourceID, &out.SourceItemID, &out.IdentityKind, &out.Title, &out.URL, &out.CanonicalURL, &out.PublishedAt, &out.FetchedAt, &out.LastSeenAt, &out.ContentPath, &out.ContentHash, &out.Version, &out.ReadAt, &starred, &out.ArchivedAt, &out.UpdatedAt, &tags, &out.SourceLifecycle, &metricProvider, &metricFetchedAt, &metricJSON); err != nil {
		return out, err
	}
	out.Starred = starred == 1
	_ = json.Unmarshal([]byte(tags), &out.Tags)
	if metricProvider.Valid || metricFetchedAt.Valid || metricJSON.Valid {
		var itemMetrics metrics.ItemMetrics
		if metricJSON.Valid && metricJSON.String != "" {
			_ = json.Unmarshal([]byte(metricJSON.String), &itemMetrics)
		}
		if metricProvider.Valid {
			itemMetrics.Provider = metricProvider.String
		}
		if metricFetchedAt.Valid {
			itemMetrics.FetchedAt = metricFetchedAt.String
		}
		out.Metrics = &itemMetrics
	}
	return out, nil
}

func ensureAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}

func tagsJSON(tags []string) string {
	b, _ := json.Marshal(tags)
	return string(b)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nowString() string { return time.Now().UTC().Format(time.RFC3339) }

func (s StorageStats) Total() int64 {
	return s.CurrentMarkdownBytes + s.VersionsBytes + s.DatabaseBytes
}

func HumanBytes(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(n)
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d%s", n, units[idx])
	}
	return fmt.Sprintf("%.1f%s", v, units[idx])
}
