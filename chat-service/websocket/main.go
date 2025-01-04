package main

import (
	"chat-service/helper"
	pb "chat-service/proto/script"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var redisClient = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379", // Sesuaikan dengan konfigurasi Redis Anda
	Password: "",               // Jika tidak ada password, kosongkan
	DB:       0,                // Gunakan database default
})

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	db           *gorm.DB
	ClientsMutex sync.Mutex
	grpcClient   pb.ChatServiceClient
)

func initGrpcClient() error {
	conn, err := grpc.Dial("localhost:50054", grpc.WithInsecure()) // Ganti alamat sesuai server gRPC Anda
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	grpcClient = pb.NewChatServiceClient(conn)
	return nil
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	token := r.Header.Get("token")
	if token == "" {
		log.Println("Missing token header")
		conn.WriteMessage(websocket.TextMessage, []byte("Unauthorized"))
		return
	}

	senderID, err := helper.GetIdFromJWT1(token)
	if err != nil {
		fmt.Printf("Failed to extract sender ID from token: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte("Invalid token"))
		return
	}

	userID, _ := strconv.Atoi(senderID.(string))

	if err := updateUserIsOnlineStatus(userID, true); err != nil {
		log.Printf("Failed to update user online status for user_id %d: %v", userID, err)
		conn.WriteMessage(websocket.TextMessage, []byte("Failed to update online status"))
		return
	}

	defer func() {
		if err := updateUserIsOnlineStatus(userID, false); err != nil {
			log.Printf("Failed to update user offline status for user_id %d: %v", userID, err)
		}
	}()

	redisKey := fmt.Sprintf("user:%d:offline_messages", userID)

	offlineMessages, err := redisClient.LRange(context.Background(), redisKey, 0, -1).Result()
	if err != nil {
		log.Printf("Error retrieving offline messages for user_id %d: %v", userID, err)
	} else {
		for _, msg := range offlineMessages {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				log.Printf("Error sending offline message to user_id %d: %v", userID, err)
			}
		}

		if err := redisClient.Del(context.Background(), redisKey).Err(); err != nil {
			log.Printf("Error deleting offline messages for user_id %d: %v", userID, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		md := metadata.Pairs("token", token)
		ctx = metadata.NewOutgoingContext(ctx, md)

		stream, err := grpcClient.StreamMessages(ctx, &pb.StreamMessagesRequest{
			Token: token,
		})

		if err != nil {
			log.Printf("Error starting gRPC stream for user_id %d: %v", userID, err)
			return
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("Error receiving gRPC message for user_id %d: %v", userID, err)
				cancel()
				break
			}

			if err := conn.WriteJSON(resp); err != nil {
				log.Printf("Error writing to WebSocket: %v", err)
				cancel()
				break
			}
		}
	}()

	for {
		var rawMsg map[string]interface{}
		if err := conn.ReadJSON(&rawMsg); err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			cancel()
			break
		}

		var msg pb.SendMessageRequest
		if content, ok := rawMsg["content"].(string); ok {
			msg.Content = content
		}

		receiverIDRaw := rawMsg["receiver_id"]

		switch v := receiverIDRaw.(type) {
		case float64:
			msg.ReceiverId = []int32{int32(v)}
		case []interface{}:
			for _, id := range v {
				if idFloat, ok := id.(float64); ok {
					msg.ReceiverId = append(msg.ReceiverId, int32(idFloat))
				}
			}
		default:
			log.Println("Invalid receiver_id format")
			conn.WriteMessage(websocket.TextMessage, []byte("Invalid receiver_id format"))
			continue
		}

		log.Printf("Message from WebSocket: %s", msg.Content)

		if len(msg.ReceiverId) == 0 {
			log.Println("No receiver IDs provided")
			conn.WriteMessage(websocket.TextMessage, []byte("No receiver IDs provided"))
			continue
		}

		for _, receiverID := range msg.ReceiverId {
			log.Printf("Forwarding message to receiver ID %d", receiverID)
			if err := helper.ForwardToGrpc(msg.Content, []int{int(receiverID)}, token); err != nil {
				log.Printf("Message content length: %d", len(msg.Content))
				log.Printf("Receiver IDs: %v", msg.ReceiverId)
				log.Printf("Error forwarding message to receiver %d: %v", receiverID, err)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to forward message to receiver %d", receiverID)))
				continue
			}
		}
	}
}

func updateUserIsOnlineStatus(userID int, isOnline bool) error {
	err := db.Table("users").Where("id = ?", userID).Update("is_online", isOnline).Error
	if err != nil {
		return err
	}
	return nil
}

func initDB() error {
	var err error
	dsn := "host=localhost user=postgres password=admin dbname=db_jwt port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to the database: %v", err)
	}
	return nil
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	if err := initGrpcClient(); err != nil {
		log.Fatalf("Could not initialize gRPC client: %v", err)
	}

	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
