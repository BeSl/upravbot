package auth

import (
	"os"
	"testing"
	"time"

	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestNewMiddleware(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDB(t)
	defer teardownTestDB(t, db)

	middleware := NewMiddleware(cfg, db)

	if middleware == nil {
		t.Error("Expected middleware instance, got nil")
	}
	if middleware.config != cfg {
		t.Error("Expected config to be set")
	}
	if middleware.db != db {
		t.Error("Expected database to be set")
	}
}

func TestAuthorizeUser_MessageUpdate(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test with authorized user
	user := &tgbotapi.User{
		ID:        123456789,
		UserName:  "testuser",
		FirstName: "Test",
		LastName:  "User",
	}

	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: user,
			Chat: &tgbotapi.Chat{ID: 987654321},
		},
	}

	authorized, dbUser := middleware.AuthorizeUser(update)

	if !authorized {
		t.Error("Expected user to be authorized")
	}
	if dbUser == nil {
		t.Error("Expected database user to be returned")
	}
	if dbUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, dbUser.ID)
	}
	if !dbUser.IsAdmin {
		t.Error("Expected user to be admin")
	}
}

func TestAuthorizeUser_CallbackUpdate(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	user := &tgbotapi.User{
		ID:        987654321,
		UserName:  "regularuser",
		FirstName: "Regular",
		LastName:  "User",
	}

	update := tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			From: user,
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{ID: 123456789},
			},
		},
	}

	authorized, dbUser := middleware.AuthorizeUser(update)

	if !authorized {
		t.Error("Expected user to be authorized")
	}
	if dbUser == nil {
		t.Error("Expected database user to be returned")
	}
	if dbUser.IsAdmin {
		t.Error("Expected user to not be admin")
	}
}

func TestAuthorizeUser_UnauthorizedUser(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	user := &tgbotapi.User{
		ID:        999999999, // Not in allowed users
		UserName:  "unauthorized",
		FirstName: "Unauthorized",
		LastName:  "User",
	}

	update := tgbotapi.Update{
		Message: &tgbotapi.Message{
			From: user,
			Chat: &tgbotapi.Chat{ID: 987654321},
		},
	}

	authorized, dbUser := middleware.AuthorizeUser(update)

	if authorized {
		t.Error("Expected user to be unauthorized")
	}
	if dbUser != nil {
		t.Error("Expected no database user to be returned")
	}
}

func TestAuthorizeUser_EmptyUpdate(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	update := tgbotapi.Update{} // Empty update

	authorized, dbUser := middleware.AuthorizeUser(update)

	if authorized {
		t.Error("Expected empty update to be unauthorized")
	}
	if dbUser != nil {
		t.Error("Expected no database user to be returned")
	}
}

func TestRequireAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test admin user
	if !middleware.RequireAdmin(123456789) {
		t.Error("Expected admin user to pass admin check")
	}

	// Test regular user
	if middleware.RequireAdmin(987654321) {
		t.Error("Expected regular user to fail admin check")
	}

	// Test unknown user
	if middleware.RequireAdmin(111111111) {
		t.Error("Expected unknown user to fail admin check")
	}
}

func TestLogCommand(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Create user first
	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateOrUpdateUser(user)

	// Test logging command
	middleware.LogCommand(123456789, "status", "", true, "System status retrieved")

	// Verify command was logged
	history, err := db.GetCommandHistory(123456789, 1)
	if err != nil {
		t.Fatalf("Failed to get command history: %v", err)
	}

	if len(history) != 1 {
		t.Errorf("Expected 1 command in history, got %d", len(history))
	}

	if history[0].Command != "status" {
		t.Errorf("Expected command 'status', got '%s'", history[0].Command)
	}

	if !history[0].Success {
		t.Error("Expected command to be marked as successful")
	}
}

func TestGetUserHistory(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Create user and add command history
	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.CreateOrUpdateUser(user)

	// Add multiple commands
	for i := 0; i < 5; i++ {
		middleware.LogCommand(123456789, "test_command", "", true, "Test response")
	}

	// Test getting user history
	history, err := middleware.GetUserHistory(123456789, 3)
	if err != nil {
		t.Fatalf("Failed to get user history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 commands in history, got %d", len(history))
	}
}

func TestGetAllHistory_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Add command history
	middleware.LogCommand(123456789, "admin_command", "", true, "Admin response")

	// Test getting all history as admin
	history, err := middleware.GetAllHistory(123456789, 10)
	if err != nil {
		t.Fatalf("Failed to get all history: %v", err)
	}

	if len(history) == 0 {
		t.Error("Expected to get command history")
	}
}

func TestGetAllHistory_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test getting all history as non-admin
	_, err := middleware.GetAllHistory(987654321, 10)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}

	expectedMsg := "access denied: admin privileges required"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGetStats_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Test getting stats as admin
	stats, err := middleware.GetStats(123456789)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats == nil {
		t.Error("Expected stats to be returned")
	}

	// Check for expected keys
	expectedKeys := []string{"total_users", "active_users", "total_commands", "successful_commands", "recent_commands"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected key '%s' in stats", key)
		}
	}
}

func TestGetStats_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test getting stats as non-admin
	_, err := middleware.GetStats(987654321)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}
}

func TestCleanupOldData_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test cleanup as admin
	err := middleware.CleanupOldData(123456789, 30)
	if err != nil {
		t.Fatalf("Failed to cleanup old data: %v", err)
	}
}

func TestCleanupOldData_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test cleanup as non-admin
	err := middleware.CleanupOldData(987654321, 30)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}
}

func TestGetActiveUsers_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Create user session to have active users
	session := &database.UserSession{
		UserID:   123456789,
		ChatID:   987654321,
		LastSeen: time.Now(),
		IsActive: true,
	}
	db.UpdateUserSession(session)

	// Test getting active users as admin
	activeUsers, err := middleware.GetActiveUsers(123456789, 30)
	if err != nil {
		t.Fatalf("Failed to get active users: %v", err)
	}

	if activeUsers == nil {
		t.Error("Expected active users list to be returned")
	}

	if len(activeUsers) == 0 {
		t.Error("Expected at least one active user")
	}
}

func TestGetActiveUsers_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test getting active users as non-admin
	_, err := middleware.GetActiveUsers(987654321, 30)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}
}

func TestGetAllUsers_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Test getting all users as admin
	users, err := middleware.GetAllUsers(123456789)
	if err != nil {
		t.Fatalf("Failed to get all users: %v", err)
	}

	if len(users) == 0 {
		t.Error("Expected to get at least one user")
	}
}

func TestGetAllUsers_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test getting all users as non-admin
	_, err := middleware.GetAllUsers(987654321)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}
}

func TestSetUserAdmin_Success(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Create regular user
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
	db.CreateOrUpdateUser(regularUser)

	// Test setting user as admin
	err := middleware.SetUserAdmin(123456789, 987654321, true)
	if err != nil {
		t.Fatalf("Failed to set user admin: %v", err)
	}

	// Verify user is now admin
	updatedUser, err := db.GetUser(987654321)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if !updatedUser.IsAdmin {
		t.Error("Expected user to be admin")
	}
}

func TestSetUserAdmin_NonAdminCaller(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test setting admin as non-admin user
	err := middleware.SetUserAdmin(987654321, 123456789, true)
	if err == nil {
		t.Error("Expected error for non-admin caller")
	}
}

func TestSetUserAdmin_LastAdminProtection(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Create single admin user
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
	db.CreateOrUpdateUser(adminUser)

	// Test removing admin from last admin
	err := middleware.SetUserAdmin(123456789, 123456789, false)
	if err == nil {
		t.Error("Expected error when removing last admin")
	}

	expectedMsg := "cannot remove admin privileges: at least one admin must remain"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestSetUserActive_Success(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Create regular user
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
	db.CreateOrUpdateUser(regularUser)

	// Test deactivating regular user
	err := middleware.SetUserActive(123456789, 987654321, false)
	if err != nil {
		t.Fatalf("Failed to set user active: %v", err)
	}

	// Verify user is now inactive
	updatedUser, err := db.GetUser(987654321)
	if err != nil {
		t.Fatalf("Failed to get updated user: %v", err)
	}

	if updatedUser.IsActive {
		t.Error("Expected user to be inactive")
	}
}

func TestSetUserActive_LastAdminProtection(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Create single admin user
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
	db.CreateOrUpdateUser(adminUser)

	// Test deactivating last admin
	err := middleware.SetUserActive(123456789, 123456789, false)
	if err == nil {
		t.Error("Expected error when deactivating last admin")
	}

	expectedMsg := "cannot deactivate the last active admin"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestDeleteUser_Success(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Create regular user
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
	db.CreateOrUpdateUser(regularUser)

	// Test deleting regular user
	err := middleware.DeleteUser(123456789, 987654321)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = db.GetUser(987654321)
	if err == nil {
		t.Error("Expected error when getting deleted user")
	}
}

func TestDeleteUser_AdminProtection(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Test deleting admin user
	err := middleware.DeleteUser(123456789, 123456789)
	if err == nil {
		t.Error("Expected error when deleting admin user")
	}

	expectedMsg := "cannot delete admin user: remove admin privileges first"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGetUsersByStatus_Admin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

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
	db.CreateOrUpdateUser(adminUser)

	// Test getting users by status as admin
	users, err := middleware.GetUsersByStatus(123456789, true)
	if err != nil {
		t.Fatalf("Failed to get users by status: %v", err)
	}

	if len(users) == 0 {
		t.Error("Expected to get at least one active user")
	}
}

func TestGetUsersByStatus_NonAdmin(t *testing.T) {
	cfg := createTestConfig()
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	middleware := NewMiddleware(cfg, db)

	// Test getting users by status as non-admin
	_, err := middleware.GetUsersByStatus(987654321, true)
	if err == nil {
		t.Error("Expected error for non-admin user")
	}
}

// Helper functions
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
	tmpFile, err := os.CreateTemp("", "auth_test_*.db")
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

func teardownTestDB(t *testing.T, db *database.DB) {
	db.Close()
	// Note: The actual file cleanup is handled by the test framework
}
