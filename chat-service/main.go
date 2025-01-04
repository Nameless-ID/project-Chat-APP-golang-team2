package main

import (
	"chat-service/app/models"
	"chat-service/app/service"
	"chat-service/config"
	"fmt"
	"log"
	"net"

	pb "chat-service/proto/script"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.SetConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s",
		cfg.Database.DBHost, cfg.Database.DBPort, cfg.Database.DBUser, cfg.Database.DBName, cfg.Database.DBPassword)
	log.Printf("Database DSN: %s", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Database connected")

	chatservice := service.NewChatServer(db)

	db.AutoMigrate(&models.Message{})
	log.Println("Database migration complete")

	grpcServer := grpc.NewServer()
	pb.RegisterChatServiceServer(grpcServer, chatservice)

	listener, err := net.Listen("tcp", ":50054")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("gRPC server started on port 50054")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}

}
