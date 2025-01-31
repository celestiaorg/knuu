package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

func (u *User) BeforeCreate(tx *gorm.DB) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) ValidatePassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}
