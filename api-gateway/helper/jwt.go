package helper

import (
	"api-gateway/config"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/metadata"
)

func GetIdFromJWT1(tokenString string) (interface{}, error) {

	cfg, _ := config.SetConfig()
	publicKeyPEM := cfg.PublicKey

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("PEM decoding failed: block is nil")
	}

	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return rsaPubKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		id, exists := claims["id"]
		if !exists {
			return nil, fmt.Errorf("id not found in token")
		}
		return id, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func ParsingJWT(ctx context.Context) (*int, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("missing metadata in context")
	}

	authHeaders := md["token"]
	if len(authHeaders) == 0 {
		return nil, fmt.Errorf("authorization token not found")
	}

	tokenString := authHeaders[0]

	senderID, err := GetIdFromJWT1(tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to extract sender ID from token: %v", err)
	}

	senderId, _ := strconv.Atoi(senderID.(string))

	return &senderId, nil
}
