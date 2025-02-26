package database

import (
	"errors"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/celestiaorg/knuu/internal/database/models"
)

const (
	DefaultHost       = "localhost"
	DefaultUser       = "postgres"
	DefaultPassword   = "postgres"
	DefaultDBName     = "postgres"
	DefaultPort       = 5432
	DefaultSSLEnabled = false
)

type Options struct {
	Host       string
	User       string
	Password   string
	DBName     string
	Port       int
	SSLEnabled *bool
	LogLevel   logger.LogLevel
}

func New(opts Options) (*gorm.DB, error) {
	opts = setDefaults(opts)
	sslMode := "disable"
	if opts.SSLEnabled != nil && *opts.SSLEnabled {
		sslMode = "enable"
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		opts.Host, opts.User, opts.Password, opts.DBName, opts.Port, sslMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}

	db.Logger = db.Logger.LogMode(opts.LogLevel)
	return db, nil
}

// Please note that this function works only with postgres.
// For other databases, you need to implement your own function.
func IsDuplicateKeyError(err error) bool {
	return errors.Is(postgres.Dialector{}.Translate(err), gorm.ErrDuplicatedKey)
}

func setDefaults(opts Options) Options {
	if opts.Host == "" {
		opts.Host = DefaultHost
	}
	if opts.User == "" {
		opts.User = DefaultUser
	}
	if opts.Password == "" {
		opts.Password = DefaultPassword
	}
	if opts.DBName == "" {
		opts.DBName = DefaultDBName
	}
	if opts.Port == 0 {
		opts.Port = DefaultPort
	}
	if opts.SSLEnabled == nil {
		sslMode := DefaultSSLEnabled
		opts.SSLEnabled = &sslMode
	}
	if opts.LogLevel == 0 {
		opts.LogLevel = logger.Warn
	}
	return opts
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Token{},
		&models.Permission{},
		&models.Test{},
	)
}
