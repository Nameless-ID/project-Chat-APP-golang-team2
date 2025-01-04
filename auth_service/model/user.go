package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID         int            `json:"id" gorm:"primaryKey;autoIncrement"`
	Email      string         `json:"email" gorm:"not null" binding:"required,email"`
	IsVerified bool           `json:"is_verified" gorm:"default:false"`
	CreatedAt  time.Time      `json:"created_at,omitempty" gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}
