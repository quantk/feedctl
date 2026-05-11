package domain

import "fmt"

type SourceType string

const (
	SourceTypeRSS      SourceType = "rss"
	SourceTypeTelegram SourceType = "telegram"
)

func ParseSourceType(value string) (SourceType, error) {
	switch SourceType(value) {
	case SourceTypeRSS, SourceTypeTelegram:
		return SourceType(value), nil
	default:
		return "", fmt.Errorf("unknown source type %q", value)
	}
}

type SourceLifecycle string

const (
	SourceLifecycleActive   SourceLifecycle = "active"
	SourceLifecycleDisabled SourceLifecycle = "disabled"
	SourceLifecycleRemoved  SourceLifecycle = "removed"
)

func ParseSourceLifecycle(value string) (SourceLifecycle, error) {
	switch SourceLifecycle(value) {
	case SourceLifecycleActive, SourceLifecycleDisabled, SourceLifecycleRemoved:
		return SourceLifecycle(value), nil
	default:
		return "", fmt.Errorf("unknown source lifecycle %q", value)
	}
}

type SyncStatus string

const (
	SyncStatusOK     SyncStatus = "ok"
	SyncStatusFailed SyncStatus = "failed"
)

func ParseSyncStatus(value string) (SyncStatus, error) {
	if value == "" {
		return "", nil
	}
	switch SyncStatus(value) {
	case SyncStatusOK, SyncStatusFailed:
		return SyncStatus(value), nil
	default:
		return "", fmt.Errorf("unknown sync status %q", value)
	}
}

type ItemState struct {
	Read     bool
	Starred  bool
	Archived bool
}

type MetricsSnapshot struct {
	Provider  string
	FetchedAt string
	Values    map[string]int
}
