package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	authpb "api-gateway/auth-service"
	chatpb "api-gateway/chat-service/script"
	"api-gateway/middleware"
	userpb "api-gateway/user-service/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	authClient authpb.AuthServiceClient
	userClient userpb.UserServiceClient
	chatClient chatpb.ChatServiceClient
)

func main() {
	// Inisialisasi koneksi gRPC ke Auth Service
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	defer conn.Close()
	authClient = authpb.NewAuthServiceClient(conn)

	// Inisialisasi koneksi gRPC ke User Service
	userConn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	defer userConn.Close()
	userClient = userpb.NewUserServiceClient(userConn)

	// Inisialisasi koneksi gRPC ke Chat Service
	chatConn, err := grpc.Dial("localhost:50054", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Chat Service: %v", err)
	}
	defer chatConn.Close()
	chatClient = chatpb.NewChatServiceClient(chatConn)

	router := gin.Default()

	// Routing untuk Auth
	router.POST("/auth/login", loginHandler)
	router.POST("/auth/verify-otp", verifyOTPHandler)
	router.POST("/auth/verify-token", verifyTokenHandler)

	// Middleware untuk autentikasi
	router.Use(middleware.Authentication())

	// Routing untuk User Service
	router.GET("/users", getAllUsersHandler)
	router.PUT("/users/:id", updateUserHandler)

	// Routing untuk Chat Service
	router.POST("/chat/send", sendMessageHandler)
	router.GET("/chat/messages", listMessageHandler)
	router.GET("/chat/messages/:sender_id", listMessagesBySenderHandler)

	log.Println("API Gateway running on port 50051...")
	router.Run(":50051")
}

// Handler untuk Login
func loginHandler(c *gin.Context) {
	var req authpb.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	res, err := authClient.Login(context.Background(), &req)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": res.Message,
	})
}

// Handler untuk Verify OTP
func verifyOTPHandler(c *gin.Context) {
	var req authpb.OTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	res, err := authClient.VerifyOTP(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": res.Message,
		"token":   res.Token,
	})
}

func verifyTokenHandler(c *gin.Context) {
	var req authpb.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	res, err := authClient.VerifyToken(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_email": res.UserEmail,
	})
}

// Handler untuk GetAllUsers
func getAllUsersHandler(c *gin.Context) {
	name := c.Query("name")
	req := &userpb.GetAllUsersRequest{Name: name}

	res, err := userClient.GetAllUsers(context.Background(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get users"})
		return
	}

	users := []gin.H{}
	for _, user := range res.Users {
		users = append(users, gin.H{
			"id":         user.Id,
			"email":      user.Email,
			"first_name": user.FirstName,
			"last_name":  user.LastName,
			"is_online":  user.IsOnline,
		})
	}

	c.JSON(http.StatusOK, users)
}

// Handler untuk UpdateUser
func updateUserHandler(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req userpb.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	req.Id = int32(id)

	res, err := userClient.UpdateUser(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": res.Message})
}

func sendMessageHandler(c *gin.Context) {
	token := c.GetHeader("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
		return
	}

	var rawMsg map[string]interface{}

	if err := c.ShouldBindJSON(&rawMsg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var req chatpb.SendMessageRequest
	if content, ok := rawMsg["content"].(string); ok {
		req.Content = content
	}

	receiverIDRaw := rawMsg["receiver_id"]

	switch v := receiverIDRaw.(type) {
	case float64:
		req.ReceiverId = []int32{int32(v)}
	case []interface{}:
		for _, id := range v {
			if idFloat, ok := id.(float64); ok {
				req.ReceiverId = append(req.ReceiverId, int32(idFloat))
			}
		}
	default:
		log.Println("Invalid receiver_id format")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receiver_id format"})
		return
	}

	if len(req.ReceiverId) == 0 {
		log.Println("No receiver IDs provided")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No receiver IDs provided"})
		return
	}

	for _, receiverID := range req.ReceiverId {
		log.Printf("Forwarding message to receiver ID %d", receiverID)
		if err := ForwardToGrpc(req.Content, []int{int(receiverID)}, token); err != nil {
			log.Printf("Message content length: %d", len(req.Content))
			log.Printf("Receiver IDs: %v", req.ReceiverId)
			log.Printf("Error forwarding message to receiver %d: %v", receiverID, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to forward message to receiver"})
			// conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to forward message to receiver %d", receiverID)))
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "successfully send message"})
}

func listMessageHandler(c *gin.Context) {
	token := c.GetHeader("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unauthorized"})
		return
	}

	md := metadata.Pairs("token", token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	res, err := chatClient.ListMessage(ctx, &emptypb.Empty{})
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages"})
		return
	}

	messages := []gin.H{}
	for _, msg := range res.Messages {
		messages = append(messages, gin.H{
			"sender":  msg.Sender,
			"message": msg.Message,
		})
	}

	c.JSON(http.StatusOK, messages)
}

func listMessagesBySenderHandler(c *gin.Context) {
	senderID := c.Query("sender_id")

	senderIDInt, err := strconv.Atoi(senderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender ID"})
		return
	}

	// Konversi ke int32
	senderIDInt32 := int32(senderIDInt)

	// Panggil service dengan senderID bertipe int32
	res, err := chatClient.ListMessageBySender(context.Background(), &chatpb.ListMessageBySenderRequest{
		SenderId: senderIDInt32,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list messages by sender"})
		return
	}

	// Siapkan respon
	messages := []gin.H{}
	for _, msg := range res.Messages {
		messages = append(messages, gin.H{
			"message": msg.Message,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sender_name": res.SenderName,
		"messages":    messages,
	})
}

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
