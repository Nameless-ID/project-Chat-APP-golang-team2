package helper

import (
	"fmt"

	pb "api-gateway/chat-service/script"

	"google.golang.org/grpc"
)

var grpcClient pb.ChatServiceClient

func InitGrpcClient() error {
	conn, err := grpc.Dial("localhost:50054", grpc.WithInsecure()) // Ganti alamat sesuai server gRPC Anda
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	grpcClient = pb.NewChatServiceClient(conn)
	return nil
}
