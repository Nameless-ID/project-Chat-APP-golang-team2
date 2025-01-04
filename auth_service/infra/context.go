package infra

import (
	"auth-service/config"
	"auth-service/database"
	"auth-service/infra/jwt"
	"auth-service/log"
	"auth-service/repository"
	"auth-service/service"
)

type ServiceContext struct {
	Service *service.Service
	Cacher  database.Cacher
	JWT     jwt.JWT
}

func NewServiceContext() (*ServiceContext, error) {
	handlerError := func(err error) (*ServiceContext, error) {
		return nil, err
	}

	appConfig, err := config.LoadConfig()
	if err != nil {
		return handlerError(err)
	}

	db, err := database.ConnectDB(appConfig)
	if err != nil {
		return handlerError(err)
	}

	logger, err := log.InitZapLogger(appConfig)
	if err != nil {
		return handlerError(err)
	}

	rdb := database.NewCacher(appConfig, 60*60)

	jwtLib := jwt.NewJWT(appConfig.PrivateKey, appConfig.PublicKey, logger)

	repo := repository.NewRepository(db, logger)
	return &ServiceContext{
		Service: service.NewService(*repo, appConfig, logger, rdb, jwtLib),
	}, nil
}
