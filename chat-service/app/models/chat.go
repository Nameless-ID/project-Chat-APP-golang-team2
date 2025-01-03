package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	SenderID   int    `binding:"required"`
	RecieverID int    `json:"reciever_id"`
	GroupId    int    `json:"group_id"`
	Content    string `gorm:"type:text" json:"content" binding:"required"`
	CreatedAt  time.Time
	DeletedAt  *gorm.DeletedAt
}

type MessageResponse struct {
	Success bool
}

type ListMessage struct {
	Name          string
	Message       string
	UnreadMessage int
}
