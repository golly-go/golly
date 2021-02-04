package golly

import (
	"context"
	"sync"

	"gorm.io/gorm"
)

type Context struct {
	context context.Context

	data *sync.Map
	db   *gorm.DB
}

// NewContext returns a new application context provided some basic information
func NewContext(ctx context.Context, db *gorm.DB) Context {
	return Context{
		context: ctx,
		db:      db,
		data:    &sync.Map{},
	}
}

// Context returns the context
func (c Context) Context() context.Context {
	return c.context
}

// Set set a value on the context
func (c *Context) Set(key string, value interface{}) {
	c.data.Store(key, value)
}

// Get get a value from the context
func (c *Context) Get(key string) (interface{}, bool) {
	return c.data.Load(key)
}

// DB returns a new DB session
func (c Context) DB() *gorm.DB {
	if c.db != nil {
		return c.db.Session(&gorm.Session{})
	}
	return nil
}
