package repository

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Repository struct {
	Auth AuthRepository
}

func NewRepository(db *gorm.DB, log *zap.Logger) *Repository {
	return &Repository{
		Auth: *NewAuthRepository(db, log),
	}
}
