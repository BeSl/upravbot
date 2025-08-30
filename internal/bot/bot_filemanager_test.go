package bot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cupbot/cupbot/internal/database"
	"github.com/cupbot/cupbot/internal/filemanager"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestEnhancedFileManagerKeyboardGeneration(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Test enhanced drive selection keyboard
	drives := []string{"C:", "D:"}
	keyboard := bot.generateEnhancedDriveSelectionKeyboard(drives)

	if len(keyboard.InlineKeyboard) == 0 {
		t.Error("Enhanced drive selection keyboard should have buttons")
	}

	// Should have drive buttons + back button
	expectedMinRows := 2 // 1 row for drives, 1 for back button
	if len(keyboard.InlineKeyboard) < expectedMinRows {
		t.Errorf("Expected at least %d rows, got %d", expectedMinRows, len(keyboard.InlineKeyboard))
	}
}

func TestEnhancedFileManagerKeyboardGeneration_NoDrives(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Test with no drives
	drives := []string{}
	keyboard := bot.generateEnhancedDriveSelectionKeyboard(drives)

	if len(keyboard.InlineKeyboard) == 0 {
		t.Error("Enhanced drive selection keyboard should have at least back button")
	}

	// Should have only back button
	if len(keyboard.InlineKeyboard) != 1 {
		t.Errorf("Expected 1 row for no drives case, got %d", len(keyboard.InlineKeyboard))
	}
}

func TestEnhancedDirectoryKeyboard(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cupbot_keyboard_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	// Get navigation context and paginated result
	context := bot.fileManager.GetNavigationContext(tempDir)
	result, err := bot.fileManager.ListDirectoryPaginated(tempDir, 1, 15)
	if err != nil {
		t.Fatalf("ListDirectoryPaginated failed: %v", err)
	}

	keyboard := bot.generateEnhancedDirectoryKeyboard(context, result)

	if len(keyboard.InlineKeyboard) == 0 {
		t.Error("Enhanced directory keyboard should have buttons")
	}

	// Should have file rows + navigation controls + back button
	expectedMinRows := 3 // At least files + navigation + back
	if len(keyboard.InlineKeyboard) < expectedMinRows {
		t.Errorf("Expected at least %d rows, got %d", expectedMinRows, len(keyboard.InlineKeyboard))
	}
}

func TestFileRowGeneration(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	// Test directory row
	dirInfo := filemanager.FileInfo{
		Name:  "testdir",
		Path:  "/test/testdir",
		IsDir: true,
		Size:  0,
	}

	dirRow := bot.generateFileRow(dirInfo)
	if len(dirRow) != 1 {
		t.Errorf("Expected 1 button in directory row, got %d", len(dirRow))
	}

	if dirRow[0].Text == "" {
		t.Error("Directory button should have text")
	}

	if dirRow[0].CallbackData == nil || *dirRow[0].CallbackData == "" {
		t.Error("Directory button should have callback data")
	}

	// Test file row
	fileInfo := filemanager.FileInfo{
		Name:  "test.txt",
		Path:  "/test/test.txt",
		IsDir: false,
		Size:  1024,
	}

	fileRow := bot.generateFileRow(fileInfo)
	if len(fileRow) != 1 {
		t.Errorf("Expected 1 button in file row, got %d", len(fileRow))
	}

	if fileRow[0].Text == "" {
		t.Error("File button should have text")
	}

	if fileRow[0].CallbackData == nil || *fileRow[0].CallbackData == "" {
		t.Error("File button should have callback data")
	}
}

func TestBreadcrumbRowGeneration(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	// Create navigation context with breadcrumbs
	context := &filemanager.NavigationContext{
		CurrentPath: "/test/path/deep/location",
		Breadcrumbs: []filemanager.BreadcrumbItem{
			{Name: "test", Path: "/test"},
			{Name: "path", Path: "/test/path"},
			{Name: "deep", Path: "/test/path/deep"},
			{Name: "location", Path: "/test/path/deep/location"},
		},
	}

	breadcrumbRow := bot.generateBreadcrumbRow(context)

	// Should have breadcrumb buttons (limited to avoid telegram limits)
	if len(breadcrumbRow) == 0 {
		t.Error("Breadcrumb row should have buttons for deep path")
	}

	// Should be limited to reasonable number of buttons
	maxExpectedButtons := 4 // 3 breadcrumbs + ellipsis
	if len(breadcrumbRow) > maxExpectedButtons {
		t.Errorf("Expected at most %d breadcrumb buttons, got %d", maxExpectedButtons, len(breadcrumbRow))
	}
}

func TestPaginationRowGeneration(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	context := &filemanager.NavigationContext{
		CurrentPath: "/test/path",
		CurrentPage: 2,
		TotalPages:  5,
	}

	result := &filemanager.PaginatedDirectoryResult{
		CurrentPage: 2,
		TotalPages:  5,
		HasNext:     true,
		HasPrev:     true,
	}

	paginationRow := bot.generatePaginationRow(context, result)

	// Should have prev, info, and next buttons
	expectedButtons := 3
	if len(paginationRow) != expectedButtons {
		t.Errorf("Expected %d pagination buttons, got %d", expectedButtons, len(paginationRow))
	}

	// Test buttons content
	if paginationRow[0].Text != "◀️ Prev" {
		t.Errorf("Expected first button to be 'Prev', got '%s'", paginationRow[0].Text)
	}

	if paginationRow[2].Text != "Next ▶️" {
		t.Errorf("Expected last button to be 'Next', got '%s'", paginationRow[2].Text)
	}
}

func TestNavigationControlsRowGeneration(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	// Test with navigation up available
	context := &filemanager.NavigationContext{
		CurrentPath:   "/test/path",
		ParentPath:    "/test",
		CanNavigateUp: true,
	}

	navRow := bot.generateNavigationControlsRow(context)

	// Should have Up, Drives, and Refresh buttons
	expectedMinButtons := 3
	if len(navRow) < expectedMinButtons {
		t.Errorf("Expected at least %d navigation buttons, got %d", expectedMinButtons, len(navRow))
	}

	// Test without navigation up available (at root)
	contextRoot := &filemanager.NavigationContext{
		CurrentPath:   "/",
		ParentPath:    "/",
		CanNavigateUp: false,
	}

	navRowRoot := bot.generateNavigationControlsRow(contextRoot)

	// Should have Drives and Refresh buttons (no Up button)
	expectedButtonsRoot := 2
	if len(navRowRoot) != expectedButtonsRoot {
		t.Errorf("Expected %d navigation buttons at root, got %d", expectedButtonsRoot, len(navRowRoot))
	}
}

func TestEnhancedFileDetailsKeyboard(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	testFilePath := "/test/file.txt"
	keyboard := bot.generateEnhancedFileDetailsKeyboard(testFilePath)

	if len(keyboard.InlineKeyboard) == 0 {
		t.Error("Enhanced file details keyboard should have buttons")
	}

	// Should have at least navigation buttons and back to menu
	expectedMinRows := 3 // Navigation + Drives + Back to menu
	if len(keyboard.InlineKeyboard) < expectedMinRows {
		t.Errorf("Expected at least %d rows, got %d", expectedMinRows, len(keyboard.InlineKeyboard))
	}
}

func TestEnhancedHandleFiles(t *testing.T) {
	bot := setupTestBot(t)
	defer teardownTestBot(t, bot)

	// Check if fileManager is available
	if bot.fileManager == nil {
		t.Skip("File manager not available in test setup")
	}

	user := &database.User{
		ID:        123456789,
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		IsAdmin:   true,
		IsActive:  true,
	}

	// Mock message
	message := &tgbotapi.Message{
		Chat: &tgbotapi.Chat{ID: 987654321},
	}

	// Test without arguments (should show drive selection)
	response, success := bot.handleFiles(message, user, "")

	if !success {
		t.Error("Enhanced handleFiles should succeed without arguments")
	}

	// Response should be empty since we send the message directly
	if response != "" {
		t.Errorf("Expected empty response when sending message directly, got: %s", response)
	}
}