package auth

import (
	"fmt"
	"log"
	"time"

	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Middleware предоставляет функции авторизации и аутентификации
type Middleware struct {
	config *config.Config
	db     *database.DB
}

// NewMiddleware создает новый экземпляр middleware
func NewMiddleware(cfg *config.Config, db *database.DB) *Middleware {
	return &Middleware{
		config: cfg,
		db:     db,
	}
}

// AuthorizeUser проверяет права пользователя на выполнение команд
func (m *Middleware) AuthorizeUser(update tgbotapi.Update) (bool, *database.User) {
	var user *tgbotapi.User
	var chatID int64

	// Получаем пользователя из разных типов обновлений
	if update.Message != nil {
		user = update.Message.From
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		user = update.CallbackQuery.From
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		return false, nil
	}

	if user == nil {
		return false, nil
	}

	// Проверяем, разрешен ли пользователь в конфигурации
	if !m.config.IsAllowed(user.ID) {
		log.Printf("Unauthorized access attempt from user %d (%s)", user.ID, user.UserName)
		return false, nil
	}

	// Создаем или обновляем пользователя в базе данных
	dbUser := &database.User{
		ID:        user.ID,
		Username:  user.UserName,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		IsAdmin:   m.config.IsAdmin(user.ID),
		IsActive:  true,
		UpdatedAt: time.Now(),
	}

	// Проверяем, существует ли пользователь в БД
	existingUser, err := m.db.GetUser(user.ID)
	if err != nil {
		// Пользователь не найден, создаем нового
		dbUser.CreatedAt = time.Now()
	} else {
		// Пользователь существует, сохраняем дату создания
		dbUser.CreatedAt = existingUser.CreatedAt
	}

	if err := m.db.CreateOrUpdateUser(dbUser); err != nil {
		log.Printf("Failed to create/update user %d: %v", user.ID, err)
	}

	// Обновляем сессию пользователя
	session := &database.UserSession{
		UserID:   user.ID,
		ChatID:   chatID,
		LastSeen: time.Now(),
		IsActive: true,
	}

	if err := m.db.UpdateUserSession(session); err != nil {
		log.Printf("Failed to update user session %d: %v", user.ID, err)
	}

	return true, dbUser
}

// RequireAdmin проверяет, является ли пользователь администратором
func (m *Middleware) RequireAdmin(userID int64) bool {
	return m.config.IsAdmin(userID)
}

// LogCommand записывает выполненную команду в историю
func (m *Middleware) LogCommand(userID int64, command string, args string, success bool, response string) {
	history := &database.CommandHistory{
		UserID:     userID,
		Command:    command,
		Arguments:  args,
		Success:    success,
		Response:   response,
		ExecutedAt: time.Now(),
	}

	if err := m.db.AddCommandHistory(history); err != nil {
		log.Printf("Failed to log command for user %d: %v", userID, err)
	}
}

// GetUserHistory возвращает историю команд пользователя
func (m *Middleware) GetUserHistory(userID int64, limit int) ([]*database.CommandHistory, error) {
	return m.db.GetCommandHistory(userID, limit)
}

// GetAllHistory возвращает всю историю команд (только для админов)
func (m *Middleware) GetAllHistory(userID int64, limit int) ([]*database.CommandHistory, error) {
	if !m.RequireAdmin(userID) {
		return nil, fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.GetAllCommandHistory(limit)
}

// GetStats возвращает статистику использования (только для админов)
func (m *Middleware) GetStats(userID int64) (map[string]interface{}, error) {
	if !m.RequireAdmin(userID) {
		return nil, fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.GetStats()
}

// CleanupOldData очищает старые данные (только для админов)
func (m *Middleware) CleanupOldData(userID int64, days int) error {
	if !m.RequireAdmin(userID) {
		return fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.CleanOldHistory(days)
}

// GetActiveUsers возвращает список активных пользователей (только для админов)
func (m *Middleware) GetActiveUsers(userID int64, minutes int) ([]*database.UserSession, error) {
	if !m.RequireAdmin(userID) {
		return nil, fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.GetActiveUsers(minutes)
}

// GetAllUsers возвращает всех пользователей (только для админов)
func (m *Middleware) GetAllUsers(userID int64) ([]*database.User, error) {
	if !m.RequireAdmin(userID) {
		return nil, fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.GetAllUsers()
}

// SetUserAdmin sets admin privileges for a user (admin only)
func (m *Middleware) SetUserAdmin(adminID int64, userID int64, isAdmin bool) error {
	if !m.RequireAdmin(adminID) {
		return fmt.Errorf("access denied: admin privileges required")
	}

	// Prevent removing admin from the last admin
	if !isAdmin {
		admins, err := m.getAdminUsers()
		if err != nil {
			return err
		}
		if len(admins) <= 1 {
			return fmt.Errorf("cannot remove admin privileges: at least one admin must remain")
		}
	}

	return m.db.SetUserAdmin(userID, isAdmin)
}

// SetUserActive activates/deactivates a user (admin only)
func (m *Middleware) SetUserActive(adminID int64, userID int64, isActive bool) error {
	if !m.RequireAdmin(adminID) {
		return fmt.Errorf("access denied: admin privileges required")
	}

	// Prevent deactivating the last admin
	if !isActive {
		user, err := m.db.GetUser(userID)
		if err != nil {
			return err
		}
		if user.IsAdmin {
			admins, err := m.getActiveAdminUsers()
			if err != nil {
				return err
			}
			if len(admins) <= 1 {
				return fmt.Errorf("cannot deactivate the last active admin")
			}
		}
	}

	return m.db.SetUserActive(userID, isActive)
}

// DeleteUser removes a user (admin only)
func (m *Middleware) DeleteUser(adminID int64, userID int64) error {
	if !m.RequireAdmin(adminID) {
		return fmt.Errorf("access denied: admin privileges required")
	}

	// Prevent deleting admin users
	user, err := m.db.GetUser(userID)
	if err != nil {
		return err
	}
	if user.IsAdmin {
		return fmt.Errorf("cannot delete admin user: remove admin privileges first")
	}

	return m.db.DeleteUser(userID)
}

// GetUsersByStatus gets users by status (admin only)
func (m *Middleware) GetUsersByStatus(adminID int64, isActive bool) ([]*database.User, error) {
	if !m.RequireAdmin(adminID) {
		return nil, fmt.Errorf("access denied: admin privileges required")
	}
	return m.db.GetUsersByStatus(isActive)
}

// Helper methods
func (m *Middleware) getAdminUsers() ([]*database.User, error) {
	users, err := m.db.GetAllUsers()
	if err != nil {
		return nil, err
	}

	var admins []*database.User
	for _, user := range users {
		if user.IsAdmin {
			admins = append(admins, user)
		}
	}
	return admins, nil
}

func (m *Middleware) getActiveAdminUsers() ([]*database.User, error) {
	users, err := m.db.GetAllUsers()
	if err != nil {
		return nil, err
	}

	var admins []*database.User
	for _, user := range users {
		if user.IsAdmin && user.IsActive {
			admins = append(admins, user)
		}
	}
	return admins, nil
}
