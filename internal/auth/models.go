package auth

import "time"

// User представляет пользователя бота
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	IsAdmin   bool      `json:"is_admin" db:"is_admin"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CommandHistory представляет историю выполненных команд
type CommandHistory struct {
	ID         int64     `json:"id" db:"id"`
	UserID     int64     `json:"user_id" db:"user_id"`
	Command    string    `json:"command" db:"command"`
	Arguments  string    `json:"arguments" db:"arguments"`
	Success    bool      `json:"success" db:"success"`
	Response   string    `json:"response" db:"response"`
	ExecutedAt time.Time `json:"executed_at" db:"executed_at"`
}

// UserSession представляет активную сессию пользователя
type UserSession struct {
	UserID   int64     `json:"user_id" db:"user_id"`
	ChatID   int64     `json:"chat_id" db:"chat_id"`
	LastSeen time.Time `json:"last_seen" db:"last_seen"`
	IsActive bool      `json:"is_active" db:"is_active"`
}
