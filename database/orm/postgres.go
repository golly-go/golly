package orm

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/slimloans/golly/env"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDBConnection new db connection
func NewPostgresConnection(v *viper.Viper, prefixKey string) (*gorm.DB, error) {
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

	db, err := gorm.Open(postgres.Open(postgressConnectionString(v, prefixKey)), &gorm.Config{Logger: logger})
	return db, err
}

func postgressConnectionString(v *viper.Viper, prefixKey string) string {
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
