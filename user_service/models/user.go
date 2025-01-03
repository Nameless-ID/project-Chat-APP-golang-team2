package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        int            `json:"id"  gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"unique"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	IsOnline  bool           `json:"is_online"`
	CreatedAt time.Time      `json:"created_at,omitempty" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index:,unique,composite:emaildeletedat" json:"-"`
}
