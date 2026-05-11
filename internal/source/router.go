package source

import (
	"context"
	"fmt"

	"feedctl/internal/config"
)

type AdapterFactory func() Adapter

type AdapterRouter struct {
	factories map[string]AdapterFactory
}

func NewAdapterRouter(adapters map[string]Adapter) *AdapterRouter {
	factories := make(map[string]AdapterFactory, len(adapters))
	for sourceType, adapter := range adapters {
		adapter := adapter
		factories[sourceType] = func() Adapter { return adapter }
	}
	return NewAdapterRouterFactories(factories)
}

func NewAdapterRouterFactories(factories map[string]AdapterFactory) *AdapterRouter {
	copied := make(map[string]AdapterFactory, len(factories))
	for sourceType, factory := range factories {
		copied[sourceType] = factory
	}
	return &AdapterRouter{factories: copied}
}

func NewDefaultAdapter() *AdapterRouter {
	return NewAdapterRouterFactories(map[string]AdapterFactory{
		config.SourceTypeRSS:      func() Adapter { return NewRSSAdapter() },
		config.SourceTypeTelegram: func() Adapter { return NewTelegramAdapter() },
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
	factory := r.factories[sourceType]
	if factory == nil {
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
	adapter := factory()
	if adapter == nil {
		return nil, fmt.Errorf("unsupported source type %q", sourceType)
	}
	return adapter, nil
}
