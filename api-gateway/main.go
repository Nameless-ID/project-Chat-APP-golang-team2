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
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	authClient authpb.AuthServiceClient
	userClient userpb.UserServiceClient
	chatClient chatpb.ChatServiceClient
)

func main() {
	// Inisialisasi koneksi gRPC ke Auth Service
	conn, err := grpc.Dial("103.127.132.149:50052", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Auth Service: %v", err)
	}
	defer conn.Close()
	authClient = authpb.NewAuthServiceClient(conn)

	// Inisialisasi koneksi gRPC ke User Service
	userConn, err := grpc.Dial("103.127.132.149:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to User Service: %v", err)
	}
	defer userConn.Close()
	userClient = userpb.NewUserServiceClient(userConn)

	// Inisialisasi koneksi gRPC ke Chat Service
	chatConn, err := grpc.Dial("103.127.132.149:50054", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to Chat Service: %v", err)
	}
	defer chatConn.Close()
	chatClient = chatpb.NewChatServiceClient(chatConn)

	router := gin.Default()

	// Routing untuk Auth
	router.POST("/auth/register", registerHandler)
	router.POST("/auth/login", loginHandler)
	router.POST("/auth/verify-otp", verifyOTPHandler)

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

// Handler untuk Register
func registerHandler(c *gin.Context) {
	var req authpb.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	res, err := authClient.Register(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": res.Status, "message": res.Message})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to login"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": res.Status, "message": res.Message})
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
		"status":     res.Status,
		"message":    res.Message,
		"user_email": res.UserEmail,
		"token":      res.Token,
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
	var req chatpb.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	res, err := chatClient.SendMessage(context.Background(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": res.Status})
}

func listMessageHandler(c *gin.Context) {
	res, err := chatClient.ListMessage(context.Background(), &emptypb.Empty{})
	if err != nil {
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
