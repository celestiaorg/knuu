package models

import (
	"time"
)

const (
	TestFinishedField  = "finished"
	TestCreatedAtField = "created_at"
)

type Test struct {
	Scope        string    `json:"scope" gorm:"primaryKey; varchar(255)"`
	UserID       uint      `json:"-" gorm:"index"` // the owner of the test
	Title        string    `json:"title" gorm:""`
	MinioEnabled bool      `json:"minio_enabled" gorm:""`
	ProxyEnabled bool      `json:"proxy_enabled" gorm:""`
	Deadline     time.Time `json:"deadline" gorm:"index"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	UpdatedAt    time.Time `json:"updated_at"`
	Finished     bool      `json:"finished" gorm:"index"`
	LogLevel     string    `json:"log_level" gorm:""` // logrus level as string (e.g. "debug", "info", "warn", "error", "fatal", "panic")
}
