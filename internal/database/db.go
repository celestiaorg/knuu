package database

import (
	"fmt"

	"github.com/celestiaorg/knuu/internal/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DefaultHost     = "localhost"
	DefaultUser     = "postgres"
	DefaultPassword = "postgres"
	DefaultDBName   = "postgres"
	DefaultPort     = 5432
)

type Options struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     int
}

func New(opts Options) (*gorm.DB, error) {
	opts = setDefaults(opts)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		opts.Host, opts.User, opts.Password, opts.DBName, opts.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
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
	return opts
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Token{},
		&models.Permission{},
	)
}
