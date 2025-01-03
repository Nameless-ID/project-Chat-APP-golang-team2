package service

import (
	"chat-service/app/models"
	"chat-service/helper"
	pb "chat-service/proto/script"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

var rdb = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

type ChatServiceServer struct {
	pb.UnimplementedChatServiceServer
	db           *gorm.DB
	Clients      map[int]chan *pb.StreamMessagesResponse // Menyimpan channel untuk streaming pesan
	ClientsMutex *sync.Mutex
}

func NewChatServer(db *gorm.DB) *ChatServiceServer {
	return &ChatServiceServer{
		db:           db,
		Clients:      make(map[int]chan *pb.StreamMessagesResponse),
		ClientsMutex: &sync.Mutex{},
	}
}

func (cs *ChatServiceServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {

	senderId, err := helper.ParsingJWT(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed parsing id %s", err)
	}

	var receiverIDs []int32

	if len(req.ReceiverId) == 0 {
		return nil, fmt.Errorf("receiver ID is required")
	} else if len(req.ReceiverId) == 1 {
		receiverIDs = []int32{req.ReceiverId[0]}
	} else {
		receiverIDs = req.ReceiverId
	}

	var messages []models.Message

	for _, receiverId := range receiverIDs {

		message := models.Message{
			SenderID:   *senderId,
			RecieverID: int(receiverId),
			Content:    req.Content,
			CreatedAt:  time.Now(),
		}
		messages = append(messages, message)

		cs.ClientsMutex.Lock()
		if ch, ok := cs.Clients[int(receiverId)]; ok {
			cs.ClientsMutex.Unlock()
			ch <- &pb.StreamMessagesResponse{
				SenderId:  int32(*senderId),
				Content:   req.Content,
				Timestamp: time.Now().Format(time.RFC3339),
			}
		} else {
			cs.ClientsMutex.Unlock()

			key := fmt.Sprintf("user:%d:offline_messages", receiverId)
			messageJSON, _ := json.Marshal(&pb.StreamMessagesResponse{
				SenderId:  int32(*senderId),
				Content:   req.Content,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			err := rdb.RPush(ctx, key, messageJSON).Err()
			if err != nil {
				log.Printf("Error storing message in Redis list: %v", err)
			}
		}
	}

	err = cs.db.Create(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to save messages to database: %v", err)
	}

	return &pb.SendMessageResponse{Status: "Sent Messages Successfully"}, nil
}

func (s *ChatServiceServer) StreamMessages(req *pb.StreamMessagesRequest, stream pb.ChatService_StreamMessagesServer) error {

	ctx := stream.Context()

	senderId, err := helper.ParsingJWT(ctx)
	if err != nil {
		return fmt.Errorf("failed parsing id %s", err)
	}

	clientChan := make(chan *pb.StreamMessagesResponse, 10)
	s.ClientsMutex.Lock()
	s.Clients[*senderId] = clientChan
	s.ClientsMutex.Unlock()

	go func() {
		key := fmt.Sprintf("user:%d:offline_messages:", *senderId)
		messages, err := rdb.LRange(context.Background(), key, 0, -1).Result()
		if err != nil {
			log.Printf("Error reading offline messages for user %d: %v", *senderId, err)
			return
		}

		for _, msg := range messages {
			var message pb.StreamMessagesResponse
			if err := json.Unmarshal([]byte(msg), &message); err == nil {
				clientChan <- &message
			}
		}

		rdb.Del(context.Background(), key)
	}()

	for msg := range clientChan {
		if err := stream.Send(msg); err != nil {
			log.Printf("Error sending message to client: %v", err)
			break
		}
	}

	s.ClientsMutex.Lock()
	delete(s.Clients, *senderId)
	s.ClientsMutex.Unlock()

	return nil
}

func (s *ChatServiceServer) ListMessage(ctx context.Context, req *emptypb.Empty) (*pb.ListMessageResponse, error) {

	userID, err := helper.ParsingJWT(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed parsing id %s", err)
	}

	var list []*pb.Message

	err = s.db.Table("messages AS m").
		Select("DISTINCT ON (m.sender_id) u.first_name as sender, m.content as message").
		Joins("JOIN users AS u ON m.sender_id = u.id").
		Where("m.reciever_id = ?", userID).
		Order("m.sender_id, m.created_at DESC").
		Scan(&list).Error

	if err != nil {
		return nil, fmt.Errorf("failed get list message")
	}

	return &pb.ListMessageResponse{Messages: list}, nil
}

func (s *ChatServiceServer) ListMessageBySender(ctx context.Context, req *pb.ListMessageBySenderRequest) (*pb.ListMessageBySenderResponse, error) {

	userID, err := helper.ParsingJWT(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed parsing id: %v", err)
	}

	if req.SenderId == 0 {
		return nil, fmt.Errorf("sender_id is required")
	}

	var senderName string
	err = s.db.Table("users").Select("first_name").Where("id = ?", req.SenderId).Scan(&senderName).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get sender name: %v", err)
	}

	var messages []string
	err = s.db.Table("messages").
		Select("content").
		Where("sender_id = ? AND reciever_id = ?", req.SenderId, userID).
		Order("created_at ASC").
		Pluck("content", &messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %v", err)
	}

	responseMessages := make([]*pb.Messages, len(messages))
	for i, msg := range messages {
		responseMessages[i] = &pb.Messages{Message: msg}
	}

	return &pb.ListMessageBySenderResponse{
		SenderName: senderName,
		Messages:   responseMessages,
	}, nil
}
