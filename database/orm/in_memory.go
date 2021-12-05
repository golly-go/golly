package orm

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// this is used for testing makes things easier.
// NewInMemoryConnection creates a new database connection and migrates any passed in model
func NewInMemoryConnection(modelToMigrate ...interface{}) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	if len(modelToMigrate) > 0 {
		db.AutoMigrate(modelToMigrate...)
	}

	return db
}
