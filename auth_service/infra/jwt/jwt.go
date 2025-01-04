package jwt

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"go.uber.org/zap"
)

// constant
const (
	REQUEST = "invalid request"
	TOKEN   = "invalid token"
)

type JWT struct {
	PrivateKey string
	PublicKey  string
	Log        *zap.Logger
	UserID     string
}

type customClaims struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	jwt.StandardClaims
}

var (
	ErrTokenEmpty       = errors.New("token is empty")
	ErrTokenExpired     = errors.New("token is expired")
	ErrInvalidToken     = errors.New("invalid token")
	ErrUnexpectedMethod = errors.New("unexpected signing method")
)

func NewJWT(privateKey, publicKey string, log *zap.Logger) JWT {
	return JWT{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Log:        log,
	}
}

func (j *JWT) CreateToken(email, ID string) (string, error) {
	//prepare private key parsing
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(j.PrivateKey))
	if err != nil {
		return "", err
	}

	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &customClaims{
		ID:             ID,
		Email:          email,
		StandardClaims: jwt.StandardClaims{ExpiresAt: expirationTime.Unix()},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)
	if err != nil {
		return "", err
	}
	return token, nil
}

func (j *JWT) VerifyToken(tokenValue string) (*customClaims, error) {
	// Parse public key
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(j.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	claims := &customClaims{}

	// Check if token is empty
	tokenValue = strings.TrimSpace(tokenValue)
	if tokenValue == "" {
		return nil, ErrTokenEmpty
	}

	token := strings.TrimPrefix(tokenValue, "Bearer ")

	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrUnexpectedMethod
		}
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %v", err)
	}

	if !parsedToken.Valid {
		return nil, ErrInvalidToken
	}

	// Check if the token is expired
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return claims, nil
}
