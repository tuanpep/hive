package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token expired")
)

type AuthService struct {
	config        *Config
	users         map[string]*User
	usersMutex    sync.RWMutex
	refreshTokens map[string]TokenMetadata
	refreshMutex  sync.RWMutex
}

type TokenMetadata struct {
	UserID    string
	ExpiresAt time.Time
}

func NewAuthService(cfg *Config) *AuthService {
	return &AuthService{
		config:        cfg,
		users:         make(map[string]*User),
		refreshTokens: make(map[string]TokenMetadata),
	}
}

func (s *AuthService) Register(req RegisterRequest) (*User, error) {
	s.usersMutex.Lock()
	defer s.usersMutex.Unlock()

	if _, exists := s.users[req.Username]; exists {
		return nil, ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &User{
		ID:        generateID(),
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.users[user.ID] = user
	s.users[req.Username] = user

	return user, nil
}

func (s *AuthService) Login(req LoginRequest) (*TokenPair, *User, error) {
	s.usersMutex.RLock()
	user, exists := s.users[req.Username]
	s.usersMutex.RUnlock()

	if !exists {
		return nil, nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, *User, error) {
	s.refreshMutex.RLock()
	metadata, exists := s.refreshTokens[refreshToken]
	s.refreshMutex.RUnlock()

	if !exists {
		return nil, nil, ErrInvalidToken
	}

	if time.Now().After(metadata.ExpiresAt) {
		s.refreshMutex.Lock()
		delete(s.refreshTokens, refreshToken)
		s.refreshMutex.Unlock()
		return nil, nil, ErrExpiredToken
	}

	s.usersMutex.RLock()
	user, exists := s.users[metadata.UserID]
	s.usersMutex.RUnlock()

	if !exists {
		return nil, nil, ErrUserNotFound
	}

	s.refreshMutex.Lock()
	delete(s.refreshTokens, refreshToken)
	s.refreshMutex.Unlock()

	tokens, err := s.generateTokens(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func (s *AuthService) Logout(refreshToken string) error {
	s.refreshMutex.Lock()
	defer s.refreshMutex.Unlock()

	if _, exists := s.refreshTokens[refreshToken]; !exists {
		return ErrInvalidToken
	}

	delete(s.refreshTokens, refreshToken)
	return nil
}

func (s *AuthService) GetUserByID(userID string) (*User, error) {
	s.usersMutex.RLock()
	defer s.usersMutex.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *AuthService) generateTokens(user *User) (*TokenPair, error) {
	now := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Type:     "access",
	})

	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshTokenRaw := make([]byte, 32)
	if _, err := rand.Read(refreshTokenRaw); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshTokenRaw)

	s.refreshMutex.Lock()
	s.refreshTokens[refreshTokenString] = TokenMetadata{
		UserID:    user.ID,
		ExpiresAt: now.Add(s.config.RefreshTokenDuration),
	}
	s.refreshMutex.Unlock()

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(s.config.AccessTokenDuration),
	}, nil
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(hash[:])
}
