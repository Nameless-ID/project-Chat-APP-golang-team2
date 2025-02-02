package helper

import (
	"context"

	chatpb "api-gateway/chat-service/script"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ForwardToGrpc(message string, receiverIDs []int, token string) error {
	conn, err := grpc.Dial("localhost:50054", grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := chatpb.NewChatServiceClient(conn)

	md := metadata.Pairs("token", token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	request := &chatpb.SendMessageRequest{
		ReceiverId: make([]int32, len(receiverIDs)),
		Content:    message,
	}

	for i, receiverID := range receiverIDs {
		request.ReceiverId[i] = int32(receiverID)
	}

	_, err = client.SendMessage(ctx, request)
	return err
}
