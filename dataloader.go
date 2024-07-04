package golly

import (
	"fmt"
	"sync"
)

type FetchFunc[T any] func(Context) (T, error)

type fetchfunc func(Context) (interface{}, error)

type DataLoader struct {
	cache sync.Map
}

func NewDataLoader() *DataLoader {
	return &DataLoader{}
}

func (dl *DataLoader) Fetch(gctx Context, key string, fetchFn fetchfunc) (interface{}, error) {
	if result, ok := dl.cache.Load(key); ok {
		return result, nil
	}

	result, err := fetchFn(gctx)
	if err != nil {
		return nil, err
	}

	dl.cache.Store(key, result)
	return result, nil
}

func LoadData[T any](gctx Context, key string, fetchFn FetchFunc[T]) (T, error) {
	var zero T

	result, err := gctx.loader.Fetch(gctx, key, func(gctx Context) (interface{}, error) {
		return fetchFn(gctx)
	})

	if err != nil {
		return zero, err
	}

	if castedResult, ok := result.(T); ok {
		return castedResult, nil
	}

	return zero, fmt.Errorf("type assertion to target type failed")
}
