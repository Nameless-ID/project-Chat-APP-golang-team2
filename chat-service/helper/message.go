package helper

import (
	"context"

	pb "chat-service/proto/script"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ForwardToGrpc(message string, receiverIDs []int, token string) error {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewChatServiceClient(conn)

	md := metadata.Pairs("Authorization", token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	request := &pb.SendMessageRequest{
		ReceiverId: make([]int32, len(receiverIDs)),
		Content:    message,
	}

	for i, receiverID := range receiverIDs {
		request.ReceiverId[i] = int32(receiverID)
	}

	_, err = client.SendMessage(ctx, request)
	return err
}
