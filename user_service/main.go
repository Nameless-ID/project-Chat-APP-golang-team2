package main

import (
	"user-service/config"
	"user-service/grpc"
	"user-service/repository"
	"user-service/service"
)

func main() {

	db := config.SetupDatabase()

	userRepo := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepo)

	grpc.StartGRPCServer(userService)
}
