package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"alert-center/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	Status   int    `json:"status"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Email  *string `json:"email"`
	Phone  *string `json:"phone"`
	Role   *string `json:"role"`
	Status *int    `json:"status"`
}

// UserManagementService handles user CRUD (admin).
type UserManagementService struct {
	db *pgxpool.Pool
}

// NewUserManagementService returns a new UserManagementService.
func NewUserManagementService(db *pgxpool.Pool) *UserManagementService {
	return &UserManagementService{db: db}
}

// Create creates a user with hashed password.
func (s *UserManagementService) Create(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "user"
	}
	status := req.Status
	if status != 0 && status != 1 {
		status = 1
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		ID:       uuid.New(),
		Username: req.Username,
		Password: string(hashed),
		Email:    req.Email,
		Phone:    req.Phone,
		Role:     role,
		Status:   status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO users (id, username, password, email, phone, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.Username, user.Password, user.Email, user.Phone, user.Role, user.Status, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByID returns a user by ID.
func (s *UserManagementService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(ctx, `
		SELECT id, username, password, email, phone, role, status, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.Phone, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// List returns users with pagination and optional role/status filter.
func (s *UserManagementService) List(ctx context.Context, page, pageSize int, role, status string) ([]models.User, int, error) {
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	args := []interface{}{}
	argNum := 0
	where := []string{}
	if role != "" {
		argNum++
		where = append(where, fmt.Sprintf("role = $%d", argNum))
		args = append(args, role)
	}
	if status != "" {
		argNum++
		where = append(where, fmt.Sprintf("status = $%d", argNum))
		args = append(args, status)
	}
	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	args = append(args, pageSize, offset)
	limitIdx := len(args) - 1
	offsetIdx := len(args)
	query := `SELECT id, username, password, email, phone, role, status, created_at, updated_at, last_login_at FROM users` + whereClause + ` ORDER BY created_at DESC LIMIT $` + fmt.Sprint(limitIdx) + ` OFFSET $` + fmt.Sprint(offsetIdx)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.Email, &u.Phone, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt); err != nil {
			return nil, 0, err
		}
		list = append(list, u)
	}
	return list, total, nil
}

// Update updates a user by ID (partial update).
func (s *UserManagementService) Update(ctx context.Context, id uuid.UUID, req *UpdateUserRequest) (*models.User, error) {
	if _, err := s.GetByID(ctx, id); err != nil {
		return nil, err
	}
	sets := []string{"updated_at = $2"}
	args := []interface{}{id, time.Now()}
	n := 3
	if req.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", n))
		args = append(args, *req.Email)
		n++
	}
	if req.Phone != nil {
		sets = append(sets, fmt.Sprintf("phone = $%d", n))
		args = append(args, *req.Phone)
		n++
	}
	if req.Role != nil {
		sets = append(sets, fmt.Sprintf("role = $%d", n))
		args = append(args, *req.Role)
		n++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", n))
		args = append(args, *req.Status)
		n++
	}
	_, err := s.db.Exec(ctx, `UPDATE users SET `+strings.Join(sets, ", ")+` WHERE id = $1`, args...)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete deletes a user by ID.
func (s *UserManagementService) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

// ChangePassword changes a user's password (old password required).
func (s *UserManagementService) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `UPDATE users SET password = $1, updated_at = $2 WHERE id = $3`, string(hashed), time.Now(), id)
	return err
}
