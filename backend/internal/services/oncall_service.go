package services

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// OnCallService manages on-call schedules and assignments via pool.
type OnCallService struct {
	db *pgxpool.Pool
}

// NewOnCallService returns a new OnCallService.
func NewOnCallService(db *pgxpool.Pool) *OnCallService {
	return &OnCallService{db: db}
}
