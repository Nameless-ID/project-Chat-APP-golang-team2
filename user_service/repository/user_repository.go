package repository

import (
	"errors"
	"user-service/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	GetUserByID(id string) (*models.User, error)
	GetAllUsers(name string) ([]*models.User, error)
	UpdateUser(user *models.User) error
	CreateUser(user *models.User) error 
}

type userRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepo{db}
}

func (r *userRepo) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetAllUsers(name string) ([]*models.User, error) {
	var users []*models.User

	if name == "" {
		if err := r.db.Model(models.User{}).Find(&users).Error; err != nil {
			return nil, err
		}
	} else {
		if err := r.db.Model(models.User{}).
			Where("first_name ILIKE ? OR last_name ILIKE ?",
				"%"+name+"%", "%"+name+"%").
			Find(&users).Error; err != nil {
			return nil, err
		}
	}

	if len(users) == 0 {
		return nil, errors.New("users not found")
	}
	return users, nil
}
func (r *userRepo) UpdateUser(user *models.User) error {
	if err := r.db.Model(&user).Where("id = ?", user.ID).Updates(user).Error; err != nil {
		return err
	}
	return nil
}
func (r *userRepo) CreateUser(user *models.User) error {
	if err := r.db.Create(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errors.New("user with this email already exists")
		}
		return err
	}

	return nil
}
