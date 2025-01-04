package middleware

import (
	"context"
	"net/http"

	authpb "api-gateway/auth-service"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type AuthMiddleware struct {
	AuthClient authpb.AuthServiceClient
}

// NewAuthMiddleware initializes the middleware with a gRPC client
func NewAuthMiddleware(authServiceAddress string) (*AuthMiddleware, error) {
	conn, err := grpc.Dial(authServiceAddress, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &AuthMiddleware{AuthClient: authpb.NewAuthServiceClient(conn)}, nil
}

// Authentication validates the JWT token
func (am *AuthMiddleware) Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("token")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "Missing Authorization header"})
			c.Abort()
			return
		}

		// Validate token using Auth Service
		_, err := am.AuthClient.VerifyToken(context.Background(), &authpb.TokenRequest{Token: authHeader})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": http.StatusUnauthorized, "message": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Proceed to the next handler
		c.Next()
	}
}
