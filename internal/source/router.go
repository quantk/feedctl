package source

import (
	"context"
	"fmt"

	"feedctl/internal/config"
)

type AdapterRouter struct {
	adapters map[string]Adapter
}

func NewAdapterRouter(adapters map[string]Adapter) *AdapterRouter {
	copied := make(map[string]Adapter, len(adapters))
	for sourceType, adapter := range adapters {
		copied[sourceType] = adapter
	}
	return &AdapterRouter{adapters: copied}
}

func NewDefaultAdapter() *AdapterRouter {
	return NewAdapterRouter(map[string]Adapter{
		config.SourceTypeRSS:      NewRSSAdapter(),
		config.SourceTypeTelegram: NewTelegramAdapter(),
	})
}

func (r *AdapterRouter) Fetch(ctx context.Context, src config.Source) (Feed, error) {
	adapter, err := r.adapter(src.Type)
	if err != nil {
		return Feed{}, err
	}
	return adapter.Fetch(ctx, src)
}

func (r *AdapterRouter) Test(ctx context.Context, src config.Source) (Metadata, error) {
	adapter, err := r.adapter(src.Type)
	if err != nil {
		return Metadata{}, err
	}
	return adapter.Test(ctx, src)
}

func (r *AdapterRouter) adapter(sourceType string) (Adapter, error) {
	if r == nil {
		return nil, fmt.Errorf("unsupported source type %q: no adapter router configured", sourceType)
	}
	adapter := r.adapters[sourceType]
	if adapter == nil {
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
	return adapter, nil
}
