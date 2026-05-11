package app

import "feedctl/internal/store"

type Source = store.Source
type Item = store.Item
type ItemFilter = store.ItemFilter
type StorageStats = store.StorageStats
type StatusSummary = store.StatusSummary

func HumanBytes(n int64) string { return store.HumanBytes(n) }
