package orm

import (
	"fmt"
	"sync"

	"github.com/slimloans/golly"
	"github.com/slimloans/golly/errors"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var db *gorm.DB
var lock sync.RWMutex

const contextKey = "database"

func InitializerWithMigration(app golly.Application, modelsToMigrate ...interface{}) golly.InitializerFunc {
	return func(app golly.Application) error {
		if err := Initializer(app); err != nil {
			return err
		}
		return db.AutoMigrate(modelsToMigrate...)
	}
}

// Initializer golly initializer setting up the databse
// todo: mkae this more dynamic going forward with interfaces etc
// since right now we only support gorm
func Initializer(app golly.Application) error {
	lock.Lock()
	defer lock.Unlock()

	v := setConfigDefaults(app.Name, app.Config)

	driver := v.GetString(fmt.Sprintf("%s.db.driver", app.Name))

	switch driver {
	case "in-memory":
		db = NewInMemoryConnection()
	case "postgres":
		d, err := NewPostgresConnection(v, app.Name)
		if err != nil {
			return errors.WrapGeneric(err)
		}
		db = d
	default:
		return errors.WrapGeneric(fmt.Errorf("database drive %s not supported", driver))
	}

	app.Routes().Use(middleware)
	return nil
}

func Connection() *gorm.DB {
	return db
}

// Not sure i want to go back to having a global database
// but for now lets do this
func DB(c golly.Context) *gorm.DB {
	if db, found := c.Get(contextKey); found {
		return db.(*gorm.DB)
	}
	return Connection()
}

func middleware(next golly.HandlerFunc) golly.HandlerFunc {
	return func(c golly.WebContext) {
		c.Set(contextKey, db.Session(&gorm.Session{NewDB: true}))
		next(c)
	}
}

// Sane defaults TODO: Clean this up
func setConfigDefaults(appName string, v *viper.Viper) *viper.Viper {
	v.SetDefault(appName, map[string]interface{}{
		"db": map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     "5432",
			"username": "app",
			"password": "password",
			"name":     appName,
			"driver":   "postgres",
		},
	})
	return v
}
