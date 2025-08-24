package auth

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents the JWT claims for DayBoard users
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token creation and validation
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager creates a new JWT manager with secret key from environment
func NewJWTManager() *JWTManager {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dayboard_default_secret_change_in_production"
	}

	// Get expiry hours from env, default to 7 days (168 hours)
	expiryHours := 168
	if envHours := os.Getenv("JWT_EXPIRY_HOURS"); envHours != "" {
		if hours, err := strconv.Atoi(envHours); err == nil {
			expiryHours = hours
		}
	}

	return &JWTManager{
		secretKey:     []byte(secret),
		tokenDuration: time.Duration(expiryHours) * time.Hour,
	}
}

// GenerateToken creates a new JWT token for a user
func (manager *JWTManager) GenerateToken(userID uuid.UUID, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(manager.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "dayboard",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(manager.secretKey)
}

// ValidateToken parses and validates a JWT token
func (manager *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return manager.secretKey, nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiry if the current token is valid
func (manager *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := manager.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	// Check if token is close to expiry (within 24 hours)
	if time.Until(claims.ExpiresAt.Time) > 24*time.Hour {
		return "", errors.New("token doesn't need refresh yet")
	}

	// Generate new token with same user info
	return manager.GenerateToken(claims.UserID, claims.Email)
}
