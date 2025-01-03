package service

import (
	"auth-service/config"
	"auth-service/database"
	"auth-service/infra/jwt"
	"auth-service/repository"

	"go.uber.org/zap"
)

type Service struct {
	Auth AuthService
}

func NewService(repo repository.Repository, config config.Config, log *zap.Logger, rdb database.Cacher, jwt jwt.JWT) *Service {
	return &Service{
		Auth: AuthService{Repo: repo.Auth, Email: NewEmailService(config.Email, log), Log: log, Cacher: rdb, Jwt: jwt},
	}
}
