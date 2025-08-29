package bot

import (
	"os"
	"testing"
	"time"

	"github.com/cupbot/cupbot/internal/auth"
	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	"github.com/cupbot/cupbot/internal/system"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Mock bot for testing without real Telegram API
type MockBot struct {
	*Bot
	sentMessages []tgbotapi.MessageConfig
}

func TestBotSetup(t *testing.T) {
	// Test that the bot can be created and basic setup works
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	if bot == nil {
		t.Error("Expected bot to be created")
	}

	if bot.config == nil {
		t.Error("Expected bot config to be set")
	}

	if bot.db == nil {
		t.Error("Expected bot database to be set")
	}

	if bot.authMw == nil {
		t.Error("Expected bot auth middleware to be set")
	}

	if bot.systemService == nil {
		t.Error("Expected bot system service to be set")
	}
}

func TestHandleHelp(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Test regular user
	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleHelp(message, user)

	if !success {
		t.Error("Expected handleHelp to succeed")
	}

	if response == "" {
		t.Error("Expected non-empty help response")
	}

	if !contains(response, "Основные команды") {
		t.Error("Expected help response to contain basic commands section")
	}

	// Test admin user
	adminUser := &database.User{
		ID:        987654321,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
	}

	adminResponse, adminSuccess := bot.handleHelp(message, adminUser)

	if !adminSuccess {
		t.Error("Expected handleHelp to succeed for admin")
	}

	if !contains(adminResponse, "Команды администратора") {
		t.Error("Expected admin help response to contain admin commands section")
	}
}

func TestHandleUptime(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleUptime(message, user)

	if !success {
		t.Error("Expected handleUptime to succeed")
	}

	if response == "" {
		t.Error("Expected non-empty uptime response")
	}

	if !contains(response, "Время работы системы") {
		t.Error("Expected uptime response to contain uptime information")
	}
}

func TestHandleHistory(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create user and add some command history
	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(user)

	// Add some test commands to history
	for i := 0; i < 5; i++ {
		history := &database.CommandHistory{
			UserID:     user.ID,
			Command:    "test_command",
			Arguments:  "",
			Success:    true,
			Response:   "Test response",
			ExecutedAt: time.Now().Add(time.Duration(-i) * time.Minute),
		}
		bot.db.AddCommandHistory(history)
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	// Test with default limit
	response, success := bot.handleHistory(message, user, "")

	if !success {
		t.Error("Expected handleHistory to succeed")
	}

	if !contains(response, "История команд") {
		t.Error("Expected history response to contain history header")
	}

	// Test with custom limit
	response2, success2 := bot.handleHistory(message, user, "3")

	if !success2 {
		t.Error("Expected handleHistory with limit to succeed")
	}

	if !contains(response2, "История команд") {
		t.Error("Expected limited history response to contain history header")
	}

	// Test with invalid limit
	_, success3 := bot.handleHistory(message, user, "invalid")

	if !success3 {
		t.Error("Expected handleHistory with invalid limit to still succeed with default")
	}
}

func TestHandleUsers_Admin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleUsers(message, adminUser)

	if !success {
		t.Error("Expected handleUsers to succeed for admin")
	}

	if !contains(response, "Список пользователей") {
		t.Error("Expected users response to contain users list header")
	}
}

func TestHandleUsers_NonAdmin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create regular user
	user := &database.User{
		ID:        123456789,
		Username:  "user",
		FirstName: "Regular",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleUsers(message, user)

	if success {
		t.Error("Expected handleUsers to fail for non-admin")
	}

	if !contains(response, "Доступ запрещен") {
		t.Error("Expected access denied message for non-admin")
	}
}

func TestHandleStats_Admin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleStats(message, adminUser)

	if !success {
		t.Error("Expected handleStats to succeed for admin")
	}

	if !contains(response, "Статистика использования") {
		t.Error("Expected stats response to contain statistics header")
	}
}

func TestHandleStats_NonAdmin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create regular user
	user := &database.User{
		ID:        123456789,
		Username:  "user",
		FirstName: "Regular",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleStats(message, user)

	if success {
		t.Error("Expected handleStats to fail for non-admin")
	}

	if !contains(response, "Доступ запрещен") {
		t.Error("Expected access denied message for non-admin")
	}
}

func TestHandleCleanup_Admin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	// Test with default days
	response, success := bot.handleCleanup(message, adminUser, "")

	if !success {
		t.Error("Expected handleCleanup to succeed for admin")
	}

	if !contains(response, "Очистка завершена") {
		t.Error("Expected cleanup success message")
	}

	// Test with custom days
	response2, success2 := bot.handleCleanup(message, adminUser, "60")

	if !success2 {
		t.Error("Expected handleCleanup with custom days to succeed")
	}

	if !contains(response2, "60 дней") {
		t.Error("Expected cleanup message to mention custom days")
	}
}

func TestHandleCleanup_NonAdmin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create regular user
	user := &database.User{
		ID:        123456789,
		Username:  "user",
		FirstName: "Regular",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
	}

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	response, success := bot.handleCleanup(message, user, "")

	if success {
		t.Error("Expected handleCleanup to fail for non-admin")
	}

	if !contains(response, "Доступ запрещен") {
		t.Error("Expected access denied message for non-admin")
	}
}

func TestHandleAddAdmin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	// Create target user
	targetUser := &database.User{
		ID:        987654321,
		Username:  "target",
		FirstName: "Target",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(targetUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 123456789},
	}

	// Test with valid user ID
	response, success := bot.handleAddAdmin(message, adminUser, "987654321")

	if !success {
		t.Error("Expected handleAddAdmin to succeed")
	}

	if !contains(response, "назначен администратором") {
		t.Error("Expected success message for admin assignment")
	}

	// Test with empty args
	response2, success2 := bot.handleAddAdmin(message, adminUser, "")

	if success2 {
		t.Error("Expected handleAddAdmin to fail with empty args")
	}

	if !contains(response2, "Необходимо указать ID") {
		t.Error("Expected error message for missing ID")
	}

	// Test with invalid user ID
	response3, success3 := bot.handleAddAdmin(message, adminUser, "invalid")

	if success3 {
		t.Error("Expected handleAddAdmin to fail with invalid ID")
	}

	if !contains(response3, "Неверный ID") {
		t.Error("Expected error message for invalid ID")
	}
}

func TestHandleRemoveAdmin(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create two admin users
	adminUser1 := &database.User{
		ID:        123456789,
		Username:  "admin1",
		FirstName: "Admin",
		LastName:  "One",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser1)

	adminUser2 := &database.User{
		ID:        987654321,
		Username:  "admin2",
		FirstName: "Admin",
		LastName:  "Two",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser2)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 123456789},
	}

	// Test removing admin from another user
	response, success := bot.handleRemoveAdmin(message, adminUser1, "987654321")

	if !success {
		t.Error("Expected handleRemoveAdmin to succeed")
	}

	if !contains(response, "убраны") {
		t.Error("Expected success message for admin removal")
	}

	// Test trying to remove admin from self
	response2, success2 := bot.handleRemoveAdmin(message, adminUser1, "123456789")

	if success2 {
		t.Error("Expected handleRemoveAdmin to fail when trying to remove self")
	}

	if !contains(response2, "у себя") {
		t.Error("Expected error message for self-removal attempt")
	}
}

func TestHandleBanUser(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin and regular user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	regularUser := &database.User{
		ID:        987654321,
		Username:  "regular",
		FirstName: "Regular",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(regularUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 123456789},
	}

	// Test banning regular user
	response, success := bot.handleBanUser(message, adminUser, "987654321")

	if !success {
		t.Error("Expected handleBanUser to succeed")
	}

	if !contains(response, "заблокирован") {
		t.Error("Expected success message for user ban")
	}

	// Test trying to ban self
	response2, success2 := bot.handleBanUser(message, adminUser, "123456789")

	if success2 {
		t.Error("Expected handleBanUser to fail when trying to ban self")
	}

	if !contains(response2, "себя") {
		t.Error("Expected error message for self-ban attempt")
	}
}

func TestHandleUnbanUser(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin and banned user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	bannedUser := &database.User{
		ID:        987654321,
		Username:  "banned",
		FirstName: "Banned",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(bannedUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 123456789},
	}

	// Test unbanning user
	response, success := bot.handleUnbanUser(message, adminUser, "987654321")

	if !success {
		t.Error("Expected handleUnbanUser to succeed")
	}

	if !contains(response, "разблокирован") {
		t.Error("Expected success message for user unban")
	}
}

func TestHandleDeleteUser(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create admin and regular user
	adminUser := &database.User{
		ID:        123456789,
		Username:  "admin",
		FirstName: "Admin",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(adminUser)

	regularUser := &database.User{
		ID:        987654321,
		Username:  "regular",
		FirstName: "Regular",
		LastName:  "User",
		IsAdmin:   false,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	bot.db.CreateOrUpdateUser(regularUser)

	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 123456789},
	}

	// Test deleting regular user
	response, success := bot.handleDeleteUser(message, adminUser, "987654321")

	if !success {
		t.Error("Expected handleDeleteUser to succeed")
	}

	if !contains(response, "удален") {
		t.Error("Expected success message for user deletion")
	}

	// Test trying to delete self
	response2, success2 := bot.handleDeleteUser(message, adminUser, "123456789")

	if success2 {
		t.Error("Expected handleDeleteUser to fail when trying to delete self")
	}

	if !contains(response2, "себя") {
		t.Error("Expected error message for self-deletion attempt")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{time.Minute * 30, "30 мин."},
		{time.Hour*2 + time.Minute*15, "2 ч. 15 мин."},
		{time.Hour*24*3 + time.Hour*5 + time.Minute*30, "3 дн. 5 ч. 30 мин."},
		{time.Hour * 25, "1 дн. 1 ч. 0 мин."},
	}

	for _, test := range tests {
		result := formatDuration(test.input)
		if result != test.expected {
			t.Errorf("formatDuration(%v): expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestParseLimit(t *testing.T) {
	tests := []struct {
		input       string
		expected    int
		shouldError bool
	}{
		{"10", 10, false},
		{"0", 0, false},
		{" 25 ", 25, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"100", 100, false},
	}

	for _, test := range tests {
		result, err := parseLimit(test.input)
		if test.shouldError {
			if err == nil {
				t.Errorf("parseLimit(%s): expected error, got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseLimit(%s): expected no error, got %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("parseLimit(%s): expected %d, got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestParseUserID(t *testing.T) {
	tests := []struct {
		input       string
		expected    int64
		shouldError bool
	}{
		{"123456789", 123456789, false},
		{" 987654321 ", 987654321, false},
		{"0", 0, false},
		{"invalid", 0, true},
		{"", 0, true},
		{"12.34", 12, false}, // Sscanf parses the integer part
	}

	for _, test := range tests {
		result, err := parseUserID(test.input)
		if test.shouldError {
			if err == nil {
				t.Errorf("parseUserID(%s): expected error, got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseUserID(%s): expected no error, got %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("parseUserID(%s): expected %d, got %d", test.input, test.expected, result)
			}
		}
	}
}

func TestGetMainKeyboard(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Test regular user keyboard
	regularKeyboard := bot.getMainKeyboard(false)
	if len(regularKeyboard.InlineKeyboard) == 0 {
		t.Error("Expected keyboard to have buttons")
	}

	// Test admin keyboard
	adminKeyboard := bot.getMainKeyboard(true)
	if len(adminKeyboard.InlineKeyboard) == 0 {
		t.Error("Expected admin keyboard to have buttons")
	}

	// Admin keyboard should have more rows than regular
	if len(adminKeyboard.InlineKeyboard) <= len(regularKeyboard.InlineKeyboard) {
		t.Error("Expected admin keyboard to have more options than regular keyboard")
	}
}

// Helper functions
func setupTestBot(t *testing.T) *Bot {
	cfg := createTestConfig()
	db := setupTestDB(t)

	// Create bot struct without API initialization
	bot := &Bot{
		api:           nil, // No real API for testing
		config:        cfg,
		db:            db,
		authMw:        auth.NewMiddleware(cfg, db),
		systemService: system.NewService(),
	}

	return bot
}

func teardownTestBot(t *testing.T, bot *Bot) {
	if bot.db != nil {
		bot.db.Close()
	}
}

func createTestConfig() *config.Config {
	return &config.Config{
		Bot: config.BotConfig{
			Token: "test_token",
			Debug: false,
		},
		Database: config.DatabaseConfig{
			Path: "test.db",
		},
		Users: config.UsersConfig{
			AdminUserIDs: []int64{123456789},
			AllowedUsers: []int64{123456789, 987654321},
		},
	}
}

func setupTestDB(t *testing.T) *database.DB {
	tmpFile, err := os.CreateTemp("", "bot_test_*.db")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	db, err := database.New(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}
