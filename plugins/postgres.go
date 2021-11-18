package golly

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/slimloans/golly"
	"github.com/slimloans/golly/env"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Postgres struct {
	DB *gorm.DB
}

func (Postgres) Name() string { return "db" }

func PostgresPlugin(app golly.Application) (golly.Plugin, error) {
	db, err := NewDBConnection(app.Config, app.Name)
	if err != nil {
		return Postgres{}, err
	}
	return Postgres{db}, nil
}

// Model default model struct (Can add additional functionality here)
type Model struct {
	ID        uint           `json:"id" faker:"-"`
	CreatedAt time.Time      `json:"created_at,omitempty" faker:"-"`
	UpdatedAt time.Time      `json:"updated_at,omitempty" faker:"-"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" faker:"-"`
}

// ModelUUID is a UUID version of model
type ModelUUID struct {
	ID        uuid.UUID      `gorm:"type:uuid;" json:"id" fake:"-"`
	CreatedAt time.Time      `json:"created_at" faker:"-"`
	UpdatedAt time.Time      `json:"updated_at" faker:"-"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" faker:"-"`
}

func (base *ModelUUID) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		uuid, err := uuid.NewRandom()
		if err != nil {
			return err
		}
		base.ID = uuid
	}
	return nil
}

func TestModelUUID() ModelUUID {
	uuid1, _ := uuid.NewUUID()
	return ModelUUID{ID: uuid1}
}

// NewDBConnection new db connection
func NewDBConnection(v *viper.Viper, prefixKey string) (*gorm.DB, error) {
	config := logger.Config{
		SlowThreshold: time.Second,
		LogLevel:      logger.Info,
		Colorful:      true,
	}

	if !env.IsDevelopmentOrTest() {
		config.LogLevel = logger.Warn
	}

	logger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		config,
	)

	db, err := gorm.Open(postgres.Open(connectionString(v, prefixKey)), &gorm.Config{Logger: logger})
	return db, err
}

func connectionString(v *viper.Viper, prefixKey string) string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	return fmt.Sprintf("dbname=%s host=%s port=%d user=%s password=%s sslmode=disable",
		v.GetString(prefixKey+".db.name"),
		v.GetString(prefixKey+".db.host"),
		v.GetInt(prefixKey+".db.port"),
		v.GetString(prefixKey+".db.username"),
		v.GetString(prefixKey+".db.password"),
	)
}

// NewInMemoryConnection creates a new database connection and migrates any passed in model
// this is used for testing makes things easier.
func NewInMemoryConnection(modelToMigrate ...interface{}) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	if len(modelToMigrate) > 0 {
		db.AutoMigrate(modelToMigrate...)
	}

	return db
}
