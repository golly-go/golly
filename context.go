package golly

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	// LoggerKey key to the data map for the logger
	LoggerKey = "logger"
)

type Context struct {
	context context.Context

	data *sync.Map

	db *gorm.DB
}

// NewContext returns a new application context provided some basic information
func NewContext(ctx context.Context) Context {
	return Context{
		context: ctx,
		data:    &sync.Map{},
	}
}

func (c *Context) WithContext(ctx context.Context) context.Context {
	c.context = ctx
	return c.context
}

func (c *Context) SetDB(db *gorm.DB) {
	c.db = db
}

func (c *Context) UpdateLogFields(fields log.Fields) {
	c.Set(LoggerKey, c.Logger().WithFields(fields))
}

func (c *Context) SetLogger(l *log.Entry) {
	c.Set(LoggerKey, l)
}

func (c Context) Logger() *log.Entry {
	if lgr, found := c.Get(LoggerKey); found {
		if l, ok := lgr.(*log.Entry); ok {
			return l
		}
	}
	// Always make sure we return a log
	// this may be required for some applications
	return log.NewEntry(log.New())
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
// not sure what todo here as we may be returning nil
// might not be safe to call in all cases
func (c Context) DB() *gorm.DB {
	return c.db
}

// NewDB returns a new session (Not sure if i like this)
func (c Context) NewDB() *gorm.DB {
	if db := c.DB(); db != nil {
		return db.Session(&gorm.Session{NewDB: true})
	}
	return nil
}
