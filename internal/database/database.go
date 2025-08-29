package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

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

// DB представляет подключение к базе данных
type DB struct {
	conn *sql.DB
}

// New создает новое подключение к базе данных
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// Close закрывает подключение к базе данных
func (db *DB) Close() error {
	return db.conn.Close()
}

// migrate выполняет миграции базы данных
func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT,
			is_admin BOOLEAN DEFAULT FALSE,
			is_active BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			command TEXT NOT NULL,
			arguments TEXT,
			success BOOLEAN NOT NULL,
			response TEXT,
			executed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			user_id INTEGER PRIMARY KEY,
			chat_id INTEGER NOT NULL,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT TRUE,
			FOREIGN KEY (user_id) REFERENCES users (id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_command_history_user_id ON command_history (user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_command_history_executed_at ON command_history (executed_at)`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions (user_id)`,
	}

	for _, query := range queries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query: %w", err)
		}
	}

	return nil
}

// CreateOrUpdateUser создает или обновляет пользователя
func (db *DB) CreateOrUpdateUser(user *User) error {
	query := `
		INSERT OR REPLACE INTO users (id, username, first_name, last_name, is_admin, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM users WHERE id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)
	`

	_, err := db.conn.Exec(query, user.ID, user.Username, user.FirstName, user.LastName,
		user.IsAdmin, user.IsActive, user.ID)

	return err
}

// GetUser получает пользователя по ID
func (db *DB) GetUser(userID int64) (*User, error) {
	query := `
		SELECT id, username, first_name, last_name, is_admin, is_active, created_at, updated_at
		FROM users WHERE id = ?
	`

	user := &User{}
	err := db.conn.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.FirstName, &user.LastName,
		&user.IsAdmin, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetAllUsers получает всех пользователей
func (db *DB) GetAllUsers() ([]*User, error) {
	query := `
		SELECT id, username, first_name, last_name, is_admin, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.FirstName, &user.LastName,
			&user.IsAdmin, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// AddCommandHistory добавляет запись в историю команд
func (db *DB) AddCommandHistory(history *CommandHistory) error {
	query := `
		INSERT INTO command_history (user_id, command, arguments, success, response, executed_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, history.UserID, history.Command, history.Arguments,
		history.Success, history.Response, history.ExecutedAt)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	history.ID = id
	return nil
}

// GetCommandHistory получает историю команд пользователя
func (db *DB) GetCommandHistory(userID int64, limit int) ([]*CommandHistory, error) {
	query := `
		SELECT id, user_id, command, arguments, success, response, executed_at
		FROM command_history 
		WHERE user_id = ? 
		ORDER BY executed_at DESC 
		LIMIT ?
	`

	rows, err := db.conn.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*CommandHistory
	for rows.Next() {
		cmd := &CommandHistory{}
		err := rows.Scan(
			&cmd.ID, &cmd.UserID, &cmd.Command, &cmd.Arguments,
			&cmd.Success, &cmd.Response, &cmd.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, cmd)
	}

	return history, nil
}

// GetAllCommandHistory получает всю историю команд (для админов)
func (db *DB) GetAllCommandHistory(limit int) ([]*CommandHistory, error) {
	query := `
		SELECT ch.id, ch.user_id, ch.command, ch.arguments, ch.success, ch.response, ch.executed_at
		FROM command_history ch
		ORDER BY ch.executed_at DESC 
		LIMIT ?
	`

	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*CommandHistory
	for rows.Next() {
		cmd := &CommandHistory{}
		err := rows.Scan(
			&cmd.ID, &cmd.UserID, &cmd.Command, &cmd.Arguments,
			&cmd.Success, &cmd.Response, &cmd.ExecutedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, cmd)
	}

	return history, nil
}

// UpdateUserSession обновляет сессию пользователя
func (db *DB) UpdateUserSession(session *UserSession) error {
	query := `
		INSERT OR REPLACE INTO user_sessions (user_id, chat_id, last_seen, is_active)
		VALUES (?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query, session.UserID, session.ChatID, session.LastSeen, session.IsActive)
	return err
}

// GetUserSession получает сессию пользователя
func (db *DB) GetUserSession(userID int64) (*UserSession, error) {
	query := `
		SELECT user_id, chat_id, last_seen, is_active
		FROM user_sessions WHERE user_id = ?
	`

	session := &UserSession{}
	err := db.conn.QueryRow(query, userID).Scan(
		&session.UserID, &session.ChatID, &session.LastSeen, &session.IsActive,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetActiveUsers получает активных пользователей за последние N минут
func (db *DB) GetActiveUsers(minutes int) ([]*UserSession, error) {
	query := `
		SELECT user_id, chat_id, last_seen, is_active
		FROM user_sessions 
		WHERE is_active = TRUE AND last_seen > datetime('now', '-' || ? || ' minutes')
		ORDER BY last_seen DESC
	`

	rows, err := db.conn.Query(query, minutes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*UserSession
	for rows.Next() {
		session := &UserSession{}
		err := rows.Scan(
			&session.UserID, &session.ChatID, &session.LastSeen, &session.IsActive,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CleanOldHistory удаляет старую историю команд (старше N дней)
func (db *DB) CleanOldHistory(days int) error {
	query := `
		DELETE FROM command_history 
		WHERE executed_at < datetime('now', '-' || ? || ' days')
	`

	_, err := db.conn.Exec(query, days)
	return err
}

// GetStats получает статистику использования
func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Общее количество пользователей
	var totalUsers int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	if err != nil {
		return nil, err
	}
	stats["total_users"] = totalUsers

	// Активные пользователи
	var activeUsers int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = TRUE").Scan(&activeUsers)
	if err != nil {
		return nil, err
	}
	stats["active_users"] = activeUsers

	// Общее количество команд
	var totalCommands int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM command_history").Scan(&totalCommands)
	if err != nil {
		return nil, err
	}
	stats["total_commands"] = totalCommands

	// Успешные команды
	var successfulCommands int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM command_history WHERE success = TRUE").Scan(&successfulCommands)
	if err != nil {
		return nil, err
	}
	stats["successful_commands"] = successfulCommands

	// Команды за последние 24 часа
	var recentCommands int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM command_history 
		WHERE executed_at > datetime('now', '-1 day')
	`).Scan(&recentCommands)
	if err != nil {
		return nil, err
	}
	stats["recent_commands"] = recentCommands

	return stats, nil
}

// SetUserAdmin sets or removes admin privileges for a user
func (db *DB) SetUserAdmin(userID int64, isAdmin bool) error {
	query := `UPDATE users SET is_admin = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, isAdmin, userID)
	return err
}

// SetUserActive activates or deactivates a user
func (db *DB) SetUserActive(userID int64, isActive bool) error {
	query := `UPDATE users SET is_active = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, isActive, userID)
	return err
}

// DeleteUser removes a user from the database
func (db *DB) DeleteUser(userID int64) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete user sessions
	_, err = tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Delete command history
	_, err = tx.Exec(`DELETE FROM command_history WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	// Delete user
	_, err = tx.Exec(`DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetUsersByStatus gets users by their active status
func (db *DB) GetUsersByStatus(isActive bool) ([]*User, error) {
	query := `
		SELECT id, username, first_name, last_name, is_admin, is_active, created_at, updated_at
		FROM users WHERE is_active = ? ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, isActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.FirstName, &user.LastName,
			&user.IsAdmin, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
