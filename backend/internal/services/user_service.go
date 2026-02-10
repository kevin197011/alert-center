package services

import (
	"context"
	"errors"
	"time"

	"alert-center/internal/models"
	"alert-center/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest is the request body for login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserService handles auth and user profile.
type UserService struct {
	repo *repository.UserRepository
}

// NewUserService returns a new UserService.
func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Login authenticates by username/password and returns user and JWT.
func (s *UserService) Login(ctx context.Context, username, password string) (*models.User, string, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	if user.Status != 1 {
		return nil, "", errors.New("user disabled")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", errors.New("invalid credentials")
	}
	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		// non-fatal
	}
	return user, token, nil
}

// GetByID returns a user by ID (password omitted in response is handled by model json:"-").
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) generateToken(user *models.User) (string, error) {
	exp := viper.GetInt64("jwt.expiration")
	if exp <= 0 {
		exp = 86400
	}
	claims := jwt.MapClaims{
		"user_id":  user.ID.String(),
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Duration(exp) * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := viper.GetString("jwt.secret")
	if secret == "" {
		secret = "change-this-secret-in-production"
	}
	return token.SignedString([]byte(secret))
}
