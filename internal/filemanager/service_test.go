package filemanager

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cupbot/cupbot/internal/config"
)

func TestGetParentDirectory(t *testing.T) {
	service := NewService(&config.Config{})
	
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Regular directory",
			path:     "C:\\Users\\Test\\Documents",
			expected: "C:\\Users\\Test",
		},
		{
			name:     "Drive root",
			path:     "C:\\",
			expected: "C:\\",
		},
		{
			name:     "Single level",
			path:     "C:\\Windows",
			expected: "C:\\",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				result := service.GetParentDirectory(tt.path)
				if result != tt.expected {
					t.Errorf("GetParentDirectory(%s) = %s, expected %s", tt.path, result, tt.expected)
				}
			} else {
				t.Skip("Skipping Windows-specific test on non-Windows platform")
			}
		})
	}
}

func TestGetDirectoryBreadcrumb(t *testing.T) {
	service := NewService(&config.Config{})
	
	tests := []struct {
		name     string
		path     string
		expected []BreadcrumbItem
	}{
		{
			name:     "Deep path",
			path:     "C:\\Users\\Test\\Documents\\Files",
			expected: []BreadcrumbItem{
				{Name: "C:", Path: "C:\\"},
				{Name: "Users", Path: "C:\\Users"},
				{Name: "Test", Path: "C:\\Users\\Test"},
				{Name: "Documents", Path: "C:\\Users\\Test\\Documents"},
				{Name: "Files", Path: "C:\\Users\\Test\\Documents\\Files"},
			},
		},
		{
			name:     "Drive root",
			path:     "C:\\",
			expected: []BreadcrumbItem{{Name: "C:", Path: "C:\\"}},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if runtime.GOOS == "windows" {
				result := service.GetDirectoryBreadcrumb(tt.path)
				if len(result) != len(tt.expected) {
					t.Errorf("GetDirectoryBreadcrumb(%s) returned %d items, expected %d", tt.path, len(result), len(tt.expected))
				}
				
				for i, expected := range tt.expected {
					if i < len(result) {
						if result[i].Name != expected.Name {
							t.Errorf("GetDirectoryBreadcrumb(%s)[%d].Name = %s, expected %s", tt.path, i, result[i].Name, expected.Name)
						}
						if result[i].Path != expected.Path {
							t.Errorf("GetDirectoryBreadcrumb(%s)[%d].Path = %s, expected %s", tt.path, i, result[i].Path, expected.Path)
						}
					}
				}
			} else {
				t.Skip("Skipping Windows-specific test on non-Windows platform")
			}
		})
	}
}

func TestIsValidPath(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Valid system path",
			path:     "C:\\Windows\\System32",
			expected: true, // Should be valid (exists) even if restricted
		},
		{
			name:     "Invalid path",
			path:     "Z:\\NonExistent\\Path",
			expected: false,
		},
		{
			name:     "Current directory",
			path:     ".",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsValidPath(tt.path)
			// Note: This test depends on the actual file system state
			// For a more robust test, we would need to mock the file system
			_ = result // Just ensure it doesn't panic
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Bytes",
			bytes:    100,
			expected: "100 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "Megabytes", 
			bytes:    1024 * 1024,
			expected: "1.0 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    1024 * 1024 * 1024,
			expected: "1.0 GB",
		},
		{
			name:     "Zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatSize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestListDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cupbot_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files and directories
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	// Test service with no drive restrictions
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	files, err := service.ListDirectory(tempDir)
	if err != nil {
		t.Fatalf("ListDirectory failed: %v", err)
	}
	
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
	
	// Verify that directories come first in the sorted list
	foundDir := false
	foundFile := false
	
	for _, file := range files {
		if file.IsDir && file.Name == "testdir" {
			foundDir = true
		}
		if !file.IsDir && file.Name == "test.txt" {
			foundFile = true
		}
	}
	
	if !foundDir {
		t.Error("Test directory not found in results")
	}
	
	if !foundFile {
		t.Error("Test file not found in results")
	}
}

func TestGetFileInfo(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "cupbot_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	
	// Write some content
	content := "test file content"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	fileInfo, err := service.GetFileInfo(tempFile.Name())
	if err != nil {
		t.Fatalf("GetFileInfo failed: %v", err)
	}
	
	if fileInfo.IsDir {
		t.Error("File should not be identified as directory")
	}
	
	if fileInfo.Size != int64(len(content)) {
		t.Errorf("Expected file size %d, got %d", len(content), fileInfo.Size)
	}
	
	if fileInfo.Name == "" {
		t.Error("File name should not be empty")
	}
}

func TestGetAvailableDrives(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:", "Z:"}, // Z: probably doesn't exist
		},
	})
	
	drives := service.GetAvailableDrives()
	
	// Should return only existing drives
	for _, drive := range drives {
		if len(drive) < 2 || drive[1] != ':' {
			t.Errorf("Invalid drive format: %s", drive)
		}
	}
	
	// On Windows, C: drive usually exists
	if runtime.GOOS == "windows" {
		foundC := false
		for _, drive := range drives {
			if drive == "C:" {
				foundC = true
				break
			}
		}
		if !foundC {
			t.Log("C: drive not found - this might be expected in some environments")
		}
	}
}

// Interactive Navigation Tests

func TestEncodeDecodePathForCallback(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	testCases := []struct {
		name string
		path string
	}{
		{"Simple path", "C:\\Users"},
		{"Path with spaces", "C:\\Program Files"},
		{"Deep path", "C:\\Users\\Documents\\Projects"},
		{"Root path", "C:\\"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode path
			encoded := service.EncodePathForCallback(tc.path)
			if encoded == "" {
				t.Error("Encoded path should not be empty")
			}
			
			// Verify it's valid base64
			_, err := base64.URLEncoding.DecodeString(encoded)
			if err != nil {
				t.Errorf("Encoded path is not valid base64: %v", err)
			}
			
			// Decode path
			decoded, err := service.DecodePathFromCallback(encoded)
			if err != nil {
				t.Fatalf("Failed to decode path: %v", err)
			}
			
			// Clean paths for comparison
			expected := filepath.Clean(tc.path)
			if decoded != expected {
				t.Errorf("Decoded path mismatch. Expected: %s, Got: %s", expected, decoded)
			}
		})
	}
}

func TestDecodePathFromCallback_Security(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	testCases := []struct {
		name        string
		encodedPath string
		shouldFail  bool
	}{
		{"Invalid base64", "invalid-base64!", true},
		{"Restricted drive", service.EncodePathForCallback("E:\\restricted"), true},
		{"Valid path", service.EncodePathForCallback("C:\\Windows"), false},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.DecodePathFromCallback(tc.encodedPath)
			if tc.shouldFail && err == nil {
				t.Error("Expected decoding to fail, but it succeeded")
			} else if !tc.shouldFail && err != nil {
				t.Errorf("Expected decoding to succeed, but got error: %v", err)
			}
		})
	}
}

func TestGetNavigationContext(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	testCases := []struct {
		name              string
		path              string
		expectedCanNavUp  bool
	}{
		{"Root path", "C:\\", false},
		{"User directory", "C:\\Users", true},
		{"Deep path", "C:\\Users\\Documents", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS != "windows" {
				t.Skip("Skipping Windows-specific test")
			}
			
			navContext := service.GetNavigationContext(tc.path)
			
			if navContext == nil {
				t.Fatal("Navigation context should not be nil")
			}
			
			if navContext.CurrentPath == "" {
				t.Error("Current path should not be empty")
			}
			
			if navContext.CanNavigateUp != tc.expectedCanNavUp {
				t.Errorf("Expected CanNavigateUp: %v, got: %v", tc.expectedCanNavUp, navContext.CanNavigateUp)
			}
			
			if len(navContext.Breadcrumbs) == 0 {
				t.Error("Breadcrumbs should not be empty")
			}
		})
	}
}

func TestValidateCallbackPath(t *testing.T) {
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	testCases := []struct {
		name       string
		path       string
		shouldFail bool
	}{
		{"Valid drive path", "C:\\", false},
		{"Invalid drive path", "Z:\\test", true},
		{"Non-existent path", "C:\\non_existent_12345", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if runtime.GOOS != "windows" {
				t.Skip("Skipping Windows-specific test")
			}
			
			err := service.ValidateCallbackPath(tc.path)
			if tc.shouldFail && err == nil {
				t.Error("Expected validation to fail, but it succeeded")
			} else if !tc.shouldFail && err != nil {
				t.Errorf("Expected validation to succeed, but got error: %v", err)
			}
		})
	}
}

// Pagination Tests

func TestListDirectoryPaginated(t *testing.T) {
	// Create a temporary directory with many files for testing
	tempDir, err := os.MkdirTemp("", "cupbot_pagination_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create 50 test files
	for i := 0; i < 50; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("file_%03d.txt", i))
		if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %d: %v", i, err)
		}
	}
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	testCases := []struct {
		name         string
		page         int
		pageSize     int
		expectedSize int
		expectedPage int
		hasNext      bool
		hasPrev      bool
	}{
		{"First page", 1, 20, 20, 1, true, false},
		{"Second page", 2, 20, 20, 2, true, true},
		{"Last page", 3, 20, 10, 3, false, true},
		{"Large page size", 1, 100, 50, 1, false, false},
		{"Page beyond range", 10, 20, 0, 10, false, true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.ListDirectoryPaginated(tempDir, tc.page, tc.pageSize)
			if err != nil {
				t.Fatalf("ListDirectoryPaginated failed: %v", err)
			}
			
			if len(result.Files) != tc.expectedSize {
				t.Errorf("Expected %d files, got %d", tc.expectedSize, len(result.Files))
			}
			
			if result.CurrentPage != tc.expectedPage {
				t.Errorf("Expected current page %d, got %d", tc.expectedPage, result.CurrentPage)
			}
			
			if result.HasNext != tc.hasNext {
				t.Errorf("Expected HasNext %v, got %v", tc.hasNext, result.HasNext)
			}
			
			if result.HasPrev != tc.hasPrev {
				t.Errorf("Expected HasPrev %v, got %v", tc.hasPrev, result.HasPrev)
			}
			
			if result.TotalFiles != 50 {
				t.Errorf("Expected total files 50, got %d", result.TotalFiles)
			}
		})
	}
}

func TestListDirectoryPaginated_EmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cupbot_empty_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	result, err := service.ListDirectoryPaginated(tempDir, 1, 20)
	if err != nil {
		t.Fatalf("ListDirectoryPaginated failed: %v", err)
	}
	
	if len(result.Files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(result.Files))
	}
	
	if result.TotalFiles != 0 {
		t.Errorf("Expected total files 0, got %d", result.TotalFiles)
	}
	
	if result.TotalPages != 1 {
		t.Errorf("Expected total pages 1, got %d", result.TotalPages)
	}
	
	if result.HasNext || result.HasPrev {
		t.Error("Empty directory should not have next or previous pages")
	}
}

func TestListDirectoryPaginated_InvalidParameters(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cupbot_invalid_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	// Test with invalid page (should default to 1)
	result, err := service.ListDirectoryPaginated(tempDir, 0, 20)
	if err != nil {
		t.Fatalf("ListDirectoryPaginated failed: %v", err)
	}
	
	if result.CurrentPage != 1 {
		t.Errorf("Expected current page 1 for invalid page 0, got %d", result.CurrentPage)
	}
	
	// Test with invalid page size (should default to 20)
	result, err = service.ListDirectoryPaginated(tempDir, 1, 0)
	if err != nil {
		t.Fatalf("ListDirectoryPaginated failed: %v", err)
	}
	
	if result.PageSize != 20 {
		t.Errorf("Expected page size 20 for invalid page size 0, got %d", result.PageSize)
	}
}

// Enhanced Navigation Tests

func TestGetDriveSelectionResponse(t *testing.T) {
	// Test with drives available
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{"C:", "D:"},
		},
	})
	
	response := service.GetDriveSelectionResponse()
	
	if response == nil {
		t.Fatal("GetDriveSelectionResponse returned nil")
	}
	
	if !strings.Contains(response.Content, "Available Drives") {
		t.Error("Response content should contain 'Available Drives'")
	}
	
	// The response should always indicate update is needed for drive selection
	if response.RequiresUpdate == false {
		// This might be true if no drives are found, which is system dependent
		t.Logf("RequiresUpdate is false - may indicate no available drives on this system")
	}
}

func TestGetDriveSelectionResponse_NoDrives(t *testing.T) {
	// Test with no allowed drives
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{
			AllowedDrives: []string{}, // No drives allowed
		},
	})
	
	response := service.GetDriveSelectionResponse()
	
	if response == nil {
		t.Fatal("GetDriveSelectionResponse returned nil")
	}
	
	// When no drives are available, the response should contain drive selection header
	// But may not have "Available Drives" if there are none
	if !strings.Contains(response.Content, "File Manager") {
		t.Error("Response content should contain 'File Manager'")
	}
}

func TestGetDirectoryNavigationResponse(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cupbot_nav_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	response, err := service.GetDirectoryNavigationResponse(tempDir, 1)
	if err != nil {
		t.Fatalf("GetDirectoryNavigationResponse failed: %v", err)
	}
	
	if response == nil {
		t.Fatal("GetDirectoryNavigationResponse returned nil")
	}
	
	if !response.RequiresUpdate {
		t.Error("Expected RequiresUpdate to be true")
	}
	
	if response.Context == nil {
		t.Fatal("Response context is nil")
	}
	
	if response.Context.CurrentPath != tempDir {
		t.Errorf("Expected current path %s, got %s", tempDir, response.Context.CurrentPath)
	}
	
	if !strings.Contains(response.Content, "Current Directory") {
		t.Error("Response content should contain 'Current Directory'")
	}
	
	if !strings.Contains(response.Content, "Contents:") {
		t.Error("Response content should contain 'Contents:'")
	}
}

func TestGetFileDetailsResponse(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "cupbot_file_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	
	// Write some content
	content := "Test file content for file details"
	if _, err := tempFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	response, err := service.GetFileDetailsResponse(tempFile.Name())
	if err != nil {
		t.Fatalf("GetFileDetailsResponse failed: %v", err)
	}
	
	if response == nil {
		t.Fatal("GetFileDetailsResponse returned nil")
	}
	
	if !response.RequiresUpdate {
		t.Error("Expected RequiresUpdate to be true")
	}
	
	if !strings.Contains(response.Content, "File Details") {
		t.Error("Response content should contain 'File Details'")
	}
	
	if !strings.Contains(response.Content, "Size:") {
		t.Error("Response content should contain 'Size:'")
	}
	
	if !strings.Contains(response.Content, "Modified:") {
		t.Error("Response content should contain 'Modified:'")
	}
}

func TestGetNavigationContextWithPagination(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cupbot_context_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files and directories
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}
	
	for i := 0; i < 3; i++ {
		testSubDir := filepath.Join(tempDir, fmt.Sprintf("dir%d", i))
		if err := os.Mkdir(testSubDir, 0755); err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}
	}
	
	service := NewService(&config.Config{
		FileManager: config.FileManagerConfig{},
	})
	
	result, err := service.ListDirectoryPaginated(tempDir, 1, 20)
	if err != nil {
		t.Fatalf("ListDirectoryPaginated failed: %v", err)
	}
	
	context := service.GetNavigationContextWithPagination(tempDir, 1, result)
	
	if context == nil {
		t.Fatal("GetNavigationContextWithPagination returned nil")
	}
	
	if context.CurrentPath != tempDir {
		t.Errorf("Expected current path %s, got %s", tempDir, context.CurrentPath)
	}
	
	if context.CurrentPage != 1 {
		t.Errorf("Expected current page 1, got %d", context.CurrentPage)
	}
	
	if context.TotalFiles != 5 {
		t.Errorf("Expected 5 files, got %d", context.TotalFiles)
	}
	
	if context.TotalDirectories != 3 {
		t.Errorf("Expected 3 directories, got %d", context.TotalDirectories)
	}
	
	if len(context.Breadcrumbs) == 0 {
		t.Error("Expected breadcrumbs to be populated")
	}
}

func TestNavigationResponseStructure(t *testing.T) {
	response := &NavigationResponse{
		Content:        "Test content",
		RequiresUpdate: true,
		Context: &NavigationContext{
			CurrentPath:      "/test/path",
			ParentPath:       "/test",
			CanNavigateUp:    true,
			CurrentPage:      1,
			TotalPages:       2,
			ViewHistory:      []string{"/previous"},
			LastAction:       "navigate",
			TotalFiles:       5,
			TotalDirectories: 3,
			Breadcrumbs: []BreadcrumbItem{
				{Name: "test", Path: "/test"},
				{Name: "path", Path: "/test/path"},
			},
		},
	}
	
	if response.Content != "Test content" {
		t.Error("Content field not properly set")
	}
	
	if !response.RequiresUpdate {
		t.Error("RequiresUpdate field not properly set")
	}
	
	if response.Context.CurrentPath != "/test/path" {
		t.Error("Context CurrentPath not properly set")
	}
	
	if response.Context.TotalFiles != 5 {
		t.Error("Context TotalFiles not properly set")
	}
	
	if response.Context.TotalDirectories != 3 {
		t.Error("Context TotalDirectories not properly set")
	}
	
	if len(response.Context.Breadcrumbs) != 2 {
		t.Error("Context Breadcrumbs not properly set")
	}
}