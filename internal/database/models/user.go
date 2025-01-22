package models

import "time"

type UserRole int

const (
	RoleUser UserRole = iota
	RoleAdmin
)

type User struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"password" gorm:"not null"`
	Role      UserRole  `json:"role" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
}

type Token struct {
	ID        uint      `json:"-" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"index;not null"`
	Token     string    `json:"token" gorm:"unique;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}

type AccessLevel int

const (
	AccessLevelRead AccessLevel = iota + 1
	AccessLevelWrite
	AccessLevelAdmin
)

type Permission struct {
	ID          uint        `json:"-" gorm:"primaryKey"`
	UserID      uint        `json:"-" gorm:"index;not null"`
	Resource    string      `json:"resource" gorm:"not null"`
	AccessLevel AccessLevel `json:"access_level" gorm:"not null"`
}
