package service

import (
	"auth-service/database"
	"auth-service/helper"
	"auth-service/infra/jwt"
	"auth-service/model"
	pb "auth-service/proto"
	"auth-service/repository"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthService struct {
	Repo   repository.AuthRepository
	Email  EmailService
	Log    *zap.Logger
	Cacher database.Cacher
	Jwt    jwt.JWT
	pb.UnimplementedAuthServiceServer
}

// Login function handles user login request
func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	// Find user by email
	user, err := s.Repo.FindByEmail(req.Email)
	if err != nil {
		s.Log.Error("Failed to find user by email", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "Failed to find user by email")
	}

	if user == nil {
		// Create a new user if not found
		user = &model.User{Email: req.Email}
		err := s.Repo.Create(user)
		if err != nil {
			s.Log.Error("Failed to create user", zap.Error(err))
			return nil, status.Errorf(codes.Internal, "Failed to create user")
		}
	}

	// Generate a random OTP
	otp := helper.GenerateOTP()

	// Prepare email datax
	emailData := map[string]interface{}{
		"Email":   user.Email,
		"OTP":     otp,
		"Timeout": 5, // Timeout in minutes
	}

	// Set OTP in cache with expiration time (e.g., 5 minutes)
	otpData := map[string]string{
		"otp":        otp,
		"expires_at": strconv.FormatInt(time.Now().Add(5*time.Minute).Unix(), 10),
	}
	data, err := json.Marshal(otpData)
	if err != nil {
		s.Log.Error("Error marshalling OTP data", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error marshalling OTP data")
	}

	err = s.Cacher.SetWithExpiration(user.Email+"_otp_data", string(data), 5*time.Minute)
	if err != nil {
		s.Log.Error("Error saving OTP in Redis", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error saving OTP in Redis")
	}

	// Send OTP via email
	subject := "Your Login OTP"
	_, err = s.Email.Send(user.Email, subject, "otp_template", emailData)
	if err != nil {
		s.Log.Error("Failed to send OTP email", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to send OTP email")
	}

	// Return success response
	return &pb.AuthResponse{Message: "Login is successful. OTP sent to your email."}, nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, req *pb.OTPRequest) (*pb.AuthResponse, error) {
	// Retrieve the OTP and expiration time from the cache
	data, err := s.Cacher.Get(req.Email + "_otp_data")
	if err != nil {
		s.Log.Error("Error getting OTP from Redis", zap.Error(err))
		return nil, status.Errorf(codes.Aborted, "Error getting OTP from Redis")
	}
	// Unmarshal the data into a map
	var otpData map[string]string
	err = json.Unmarshal([]byte(data), &otpData)
	if err != nil {
		s.Log.Error("Error unmarshalling OTP data", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error unmarshalling OTP data")
	}

	// Check if the OTP matches
	tempOtp := otpData["otp"]
	if tempOtp != req.Code {
		s.Log.Error("Invalid OTP")
		return nil, status.Errorf(codes.Internal, "Invalid OTP")
	}

	// Check if the OTP is still valid
	expiresAt, err := strconv.ParseInt(otpData["expires_at"], 10, 64)
	if err != nil {
		s.Log.Error("Error parsing expiration time", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error parsing expiration time")
	}
	if time.Now().Unix() > expiresAt {
		s.Log.Error("OTP has expired")
		return nil, status.Errorf(codes.Internal, "OTP has expired")
	}

	// Delete OTP from Redis after verification
	err = s.Cacher.Delete(req.Email)
	if err != nil {
		s.Log.Error("Error deleting OTP from Redis", zap.Error(err))
		return nil, status.Errorf(codes.Aborted, "Error deleting OTP from Redis")
	}

	user, err := s.Repo.FindByEmail(req.Email)
	if err != nil {
		s.Log.Error("Failed to find user by email", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "Failed to find user by email")
	} else if user == nil {
		s.Log.Error("User not found")
		return nil, status.Errorf(codes.NotFound, "User not found")
	}

	// Generate JWT token with expiration time
	userIDStr := strconv.Itoa(int(user.ID))
	token, err := s.Jwt.CreateToken(user.Email, userIDStr)
	if err != nil {
		s.Log.Error("Error creating JWT token", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error creating JWT token")
	}

	// Check if the key exists and delete it if it does
	err = s.Cacher.Delete(user.Email + "_token")
	if err != nil && err != redis.Nil {
		s.Log.Error("Error deleting existing token from Redis", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error deleting existing token from Redis")
	}

	// Save the token to Redis
	err = s.Cacher.SaveToken(user.Email+"_token", token) // Set token with 24-hour expiration
	if err != nil {
		s.Log.Error("Error saving token to Redis", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Error saving token to Redis")
	}

	// Return success response
	return &pb.AuthResponse{Message: "OTP verified successfully", Token: token}, nil
}

func (s *AuthService) VerifyToken(ctx context.Context, req *pb.TokenRequest) (*pb.TokenResponse, error) {
	// Extract the token from the request
	tokenString := req.Token
	token := strings.TrimPrefix(tokenString, "Bearer ")

	// Log the received token
	s.Log.Debug("Received token:", zap.String("token", tokenString))

	// Verify the token using the Jwt service
	claims, err := s.Jwt.VerifyToken(tokenString)
	if err != nil {
		s.Log.Error("Error verifying token", zap.Error(err))
		if err.Error() == "token is expired" {
			return nil, status.Errorf(codes.Unauthenticated, "Token is expired")
		}
		return nil, status.Errorf(codes.Internal, "Error verifying token")
	}

	// Retrieve the token from Redis using the email from the claims
	storedToken, err := s.Cacher.Get(claims.Email + "_token")
	if err != nil {
		s.Log.Error("Error getting token from Redis", zap.Error(err))
		return nil, status.Errorf(codes.Aborted, "Error getting token from Redis")
	}

	// Log stored token for comparison
	s.Log.Debug("Stored token in Redis:", zap.String("storedToken", storedToken))

	// Compare the tokens
	if storedToken != tokenString || storedToken != token {
		s.Log.Error("Invalid token", zap.String("receivedToken", tokenString), zap.String("storedToken", storedToken))
		return nil, status.Errorf(codes.Unauthenticated, "Tokens do not match")
	}

	// Return success response
	return &pb.TokenResponse{UserEmail: claims.Email}, nil
}
