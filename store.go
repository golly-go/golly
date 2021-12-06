package golly

import "sync"

type Store struct {
	data *sync.Map
}

func NewStore() *Store {
	return &Store{&sync.Map{}}
}

// Set set a value on the context
func (store *Store) Set(key string, value interface{}) {
	store.data.Store(key, value)
}

// Get get a value from the context
func (store *Store) Get(key string) (interface{}, bool) {
	return store.data.Load(key)
}