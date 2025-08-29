package bot

import (
	"testing"
	"time"

	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	"github.com/cupbot/cupbot/internal/power"
)

// TestAdminMenuAccess tests admin menu access control
func TestAdminMenuAccess(t *testing.T) {
	bot := createTestBot(t)
	
	// Test admin user access
	adminUser := &database.User{
		ID:       123,
		Username: "admin",
		IsAdmin:  true,
		IsActive: true,
	}
	
	response, success := bot.handleAdminMenuCallback(adminUser)
	if !success {
		t.Error("Admin user should have access to admin menu")
	}
	if response != "🔑 *Admin Menu*\n\nSelect an action:" {
		t.Errorf("Unexpected admin menu response: %s", response)
	}
	
	// Test regular user access
	regularUser := &database.User{
		ID:       456,
		Username: "user",
		IsAdmin:  false,
		IsActive: true,
	}
	
	response, success = bot.handleAdminMenuCallback(regularUser)
	if success {
		t.Error("Regular user should not have access to admin menu")
	}
	if response != "❌ Access denied" {
		t.Errorf("Unexpected access denied response: %s", response)
	}
}

// TestPowerManagementCallbacks tests power management callback handlers
func TestPowerManagementCallbacks(t *testing.T) {
	bot := createTestBot(t)
	adminUser := &database.User{
		ID:       123,
		Username: "admin",
		IsAdmin:  true,
		IsActive: true,
	}
	
	tests := []struct {
		name     string
		handler  func(*database.User) (string, bool)
		expected string
		success  bool
	}{
		{
			name:     "Power Menu",
			handler:  bot.handlePowerMenuCallback,
			expected: "🔌 *Power Management*",
			success:  true,
		},
		{
			name:     "Power Status",
			handler:  bot.handlePowerStatusCallback,
			expected: "ℹ️ *Power Management Status*",
			success:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, success := tt.handler(adminUser)
			
			if success != tt.success {
				t.Errorf("Expected success %v, got %v", tt.success, success)
			}
			
			if !containsString(response, tt.expected) {
				t.Errorf("Response should contain '%s', got: %s", tt.expected, response)
			}
		})
	}
}

// TestPowerOperations tests power operation handlers with delay
func TestPowerOperations(t *testing.T) {
	bot := createTestBot(t)
	adminUser := &database.User{
		ID:       123,
		Username: "admin",
		IsAdmin:  true,
		IsActive: true,
	}
	
	// Test shutdown with delay
	response, success := bot.handleShutdownDelayCallback(adminUser, 5*time.Minute, false)
	if !success {
		t.Error("Shutdown delay should succeed for admin")
	}
	
	if !containsString(response, "scheduled") {
		t.Errorf("Response should indicate scheduling, got: %s", response)
	}
	
	// Test reboot with delay
	response, success = bot.handleRebootDelayCallback(adminUser, 10*time.Minute, false)
	// This might fail because there's already a scheduled operation
	// That's acceptable behavior
	
	// Test cancel operation
	response, success = bot.handleCancelPowerCallback(adminUser)
	if !success {
		t.Error("Cancel operation should succeed")
	}
	
	if !containsString(response, "canceled") {
		t.Errorf("Response should indicate cancellation, got: %s", response)
	}
}

// TestUserManagementCallbacks tests user management callback handlers
func TestUserManagementCallbacks(t *testing.T) {
	bot := createTestBot(t)
	adminUser := &database.User{
		ID:       123,
		Username: "admin",
		IsAdmin:  true,
		IsActive: true,
	}
	
	regularUser := &database.User{
		ID:       456,
		Username: "user",
		IsAdmin:  false,
		IsActive: true,
	}
	
	tests := []struct {
		name      string
		handler   func(*database.User) (string, bool)
		user      *database.User
		expectErr bool
	}{
		{
			name:      "User Menu - Admin",
			handler:   bot.handleUserMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
		{
			name:      "User Menu - Regular User",
			handler:   bot.handleUserMenuCallback,
			user:      regularUser,
			expectErr: true,
		},
		{
			name:      "Add Admin Menu - Admin",
			handler:   bot.handleAddAdminMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
		{
			name:      "Remove Admin Menu - Admin",
			handler:   bot.handleRemoveAdminMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
		{
			name:      "Ban User Menu - Admin",
			handler:   bot.handleBanUserMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
		{
			name:      "Unban User Menu - Admin",
			handler:   bot.handleUnbanUserMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
		{
			name:      "Delete User Menu - Admin",
			handler:   bot.handleDeleteUserMenuCallback,
			user:      adminUser,
			expectErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, success := tt.handler(tt.user)
			
			if tt.expectErr && success {
				t.Error("Expected error but operation succeeded")
			}
			
			if !tt.expectErr && !success {
				t.Errorf("Expected success but got error: %s", response)
			}
			
			if tt.expectErr && !containsString(response, "Access denied") {
				t.Errorf("Expected access denied message, got: %s", response)
			}
		})
	}
}

// TestEnhancedServiceCallbacks tests enhanced service callbacks
func TestEnhancedServiceCallbacks(t *testing.T) {
	bot := createTestBot(t)
	adminUser := &database.User{
		ID:       123,
		Username: "admin",
		IsAdmin:  true,
		IsActive: true,
	}
	
	// Test file manager admin callback
	response, success := bot.handleFileManagerAdminCallback(adminUser)
	if !success {
		t.Error("File manager admin callback should succeed")
	}
	
	if !containsString(response, "Enhanced File Manager") {
		t.Errorf("Response should mention enhanced file manager, got: %s", response)
	}
	
	// Test screenshot admin callback
	response, success = bot.handleScreenshotAdminCallback(adminUser)
	// Success depends on platform and service context, so we just check response format
	if !containsString(response, "Screenshot Service") {
		t.Errorf("Response should mention screenshot service, got: %s", response)
	}
	
	// Test system tools callback
	response, success = bot.handleSystemToolsCallback(adminUser)
	if !success {
		t.Error("System tools callback should succeed")
	}
	
	if !containsString(response, "System Tools") {
		t.Errorf("Response should mention system tools, got: %s", response)
	}
}

// TestMenuNavigationHelpers tests the helper functions for menu navigation
func TestMenuNavigationHelpers(t *testing.T) {
	testCases := []struct {
		callback string
		isMenu   bool
		isPower  bool
		isUser   bool
	}{
		{"main_menu", true, false, false},
		{"admin_menu", true, false, false},
		{"power_menu", true, true, false},
		{"user_menu", true, false, true},
		{"shutdown_now", false, true, false},
		{"reboot_1min", false, true, false},
		{"add_admin_menu", false, false, true},
		{"status", false, false, false},
		{"unknown_callback", false, false, false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.callback, func(t *testing.T) {
			if isMenuCallback(tc.callback) != tc.isMenu {
				t.Errorf("isMenuCallback(%s) = %v, expected %v", tc.callback, isMenuCallback(tc.callback), tc.isMenu)
			}
			
			if isPowerCallback(tc.callback) != tc.isPower {
				t.Errorf("isPowerCallback(%s) = %v, expected %v", tc.callback, isPowerCallback(tc.callback), tc.isPower)
			}
			
			if isUserManagementCallback(tc.callback) != tc.isUser {
				t.Errorf("isUserManagementCallback(%s) = %v, expected %v", tc.callback, isUserManagementCallback(tc.callback), tc.isUser)
			}
		})
	}
}

// TestKeyboardGeneration tests that keyboards are generated correctly
func TestKeyboardGeneration(t *testing.T) {
	bot := createTestBot(t)
	
	// Test main keyboard for regular user
	keyboard := bot.getMainKeyboard(false)
	if len(keyboard.InlineKeyboard) == 0 {
		t.Error("Main keyboard should have buttons")
	}
	
	// Test main keyboard for admin (should have more buttons)
	adminKeyboard := bot.getMainKeyboard(true)
	if len(adminKeyboard.InlineKeyboard) <= len(keyboard.InlineKeyboard) {
		t.Error("Admin keyboard should have more buttons than regular keyboard")
	}
	
	// Test admin-specific keyboards
	adminOnlyKeyboard := bot.getAdminKeyboard()
	if len(adminOnlyKeyboard.InlineKeyboard) == 0 {
		t.Error("Admin keyboard should have buttons")
	}
	
	powerKeyboard := bot.getPowerMenuKeyboard()
	if len(powerKeyboard.InlineKeyboard) == 0 {
		t.Error("Power keyboard should have buttons")
	}
	
	userManagementKeyboard := bot.getUserManagementKeyboard()
	if len(userManagementKeyboard.InlineKeyboard) == 0 {
		t.Error("User management keyboard should have buttons")
	}
}

// Helper functions for testing

func createTestBot(t *testing.T) *Bot {
	cfg := &config.Config{
		Bot: config.BotConfig{
			Token: "test_token",
			Debug: false,
		},
		Screenshot: config.ScreenshotConfig{
			Enabled: true,
			Format:  "png",
		},
	}
	
	// Create a mock database (in real implementation, use a test database)
	db := &database.DB{} // This should be properly initialized in real tests
	
	bot := &Bot{
		config:            cfg,
		db:               db,
		powerService:     power.NewService(cfg),
		// Other services would be initialized here
	}
	
	return bot
}

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && 
		   (str == substr || 
		    findInString(str, substr))
}

func findInString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}