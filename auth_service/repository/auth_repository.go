package repository

import (
	"auth-service/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthRepository struct {
	DB  *gorm.DB
	log *zap.Logger
}

func NewAuthRepository(db *gorm.DB, log *zap.Logger) *AuthRepository {
	return &AuthRepository{DB: db, log: log}
}

func (repo *AuthRepository) Create(user *model.User) error {
	return repo.DB.Transaction(func(tx *gorm.DB) error {
		// Create user
		err := tx.Create(&user).Error
		if err != nil {
			repo.log.Error("Failed to create user", zap.Error(err))
			return err
		}
		return nil
	})
}

func (repo *AuthRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := repo.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		repo.log.Error("Failed to find user by email", zap.Error(err))
		return nil, err
	}
	return &user, nil
}
func (repo *AuthRepository) Update(user *model.User) error {
	return repo.DB.Transaction(func(tx *gorm.DB) error {
		// Update user
		err := tx.Save(&user).Error
		if err != nil {
			repo.log.Error("Failed to update user", zap.Error(err))
			return err
		}
		return nil
	})
}
