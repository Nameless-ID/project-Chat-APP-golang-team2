package main

import (
	"chat-service/helper"
	pb "chat-service/proto/script"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	ClientsMutex sync.Mutex
	grpcClient   pb.ChatServiceClient
)

func initGrpcClient() error {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // Ganti alamat sesuai server gRPC Anda
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	grpcClient = pb.NewChatServiceClient(conn)
	return nil
}

// func wsHandler(w http.ResponseWriter, r *http.Request) {
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Println("Failed to upgrade connection:", err)
// 		return
// 	}
// 	defer conn.Close()
//
// 	// Ambil token dari header
// 	authHeader := r.Header.Get("Authorization")
// 	if authHeader == "" {
// 		log.Println("Missing Authorization header")
// 		conn.WriteMessage(websocket.TextMessage, []byte("Unauthorized"))
// 		return
// 	}
//
// 	tokenParts := strings.Split(authHeader, " ")
// 	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
// 		log.Println("Invalid Authorization header format")
// 		conn.WriteMessage(websocket.TextMessage, []byte("Invalid token format"))
// 		return
// 	}
//
// 	token := tokenParts[1]
//
// 	senderID, err := helper.GetIdFromJWT1(token)
// 	if err != nil {
// 		fmt.Printf("failed to extract sender ID from token: %v", err)
// 		return
// 	}
//
// 	userID, _ := strconv.Atoi(senderID.(string))
//
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
//
// 	var wg sync.WaitGroup
//
// 	// Goroutine untuk streaming pesan dari gRPC
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		md := metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token))
// 		ctx = metadata.NewOutgoingContext(ctx, md)
//
// 		stream, err := grpcClient.StreamMessages(ctx, &pb.StreamMessagesRequest{
// 			Token: token,
// 		})
//
// 		if err != nil {
// 			log.Printf("Error starting gRPC stream for user_id %d: %v", userID, err)
// 			return
// 		}
//
// 		for {
// 			resp, err := stream.Recv()
// 			if err != nil {
// 				log.Printf("Error receiving gRPC message for user_id %d: %v", userID, err)
// 				cancel()
// 				break
// 			}
//
// 			if err := conn.WriteJSON(resp); err != nil {
// 				log.Printf("Error writing to WebSocket: %v", err)
// 				cancel()
// 				break
// 			}
// 		}
// 	}()
//
// 	for {
//
// 		var msg pb.SendMessageRequest
// 		if err := conn.ReadJSON(&msg); err != nil {
// 			log.Printf("Error reading WebSocket message: %v", err)
// 			cancel()
// 			break
// 		}
//
// 		log.Printf("Message from WebSocket: %s", &msg)
//
// 		if err := helper.ForwardToGrpc(msg.Content, int(msg.ReceiverId), authHeader); err != nil {
// 			log.Println("Error forwarding message to gRPC:", err)
// 			conn.WriteMessage(websocket.TextMessage, []byte("Failed to forward message"))
// 			continue
// 		}
// 	}
// }

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}
	defer conn.Close()

	// Ambil token dari header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		log.Println("Missing Authorization header")
		conn.WriteMessage(websocket.TextMessage, []byte("Unauthorized"))
		return
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		log.Println("Invalid Authorization header format")
		conn.WriteMessage(websocket.TextMessage, []byte("Invalid token format"))
		return
	}

	token := tokenParts[1]

	senderID, err := helper.GetIdFromJWT1(token)
	if err != nil {
		fmt.Printf("failed to extract sender ID from token: %v", err)
		return
	}

	userID, _ := strconv.Atoi(senderID.(string))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Goroutine untuk streaming pesan dari gRPC
	wg.Add(1)
	go func() {
		defer wg.Done()
		md := metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", token))
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
			if err := helper.ForwardToGrpc(msg.Content, []int{int(receiverID)}, authHeader); err != nil {
				log.Printf("Error forwarding message to receiver %d: %v", receiverID, err)
				conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to forward message to receiver %d", receiverID)))
				continue
			}
		}
	}
}

func main() {

	if err := initGrpcClient(); err != nil {
		log.Fatalf("Could not initialize gRPC client: %v", err)
	}

	http.HandleFunc("/ws", wsHandler)
	log.Println("WebSocket server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
