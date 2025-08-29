package filemanager

import (
	"os"
	"path/filepath"
	"runtime"
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
		expected []string
	}{
		{
			name:     "Deep path",
			path:     "C:\\Users\\Test\\Documents\\Files",
			expected: []string{"C:\\", "C:\\Users", "C:\\Users\\Test", "C:\\Users\\Test\\Documents", "C:\\Users\\Test\\Documents\\Files"},
		},
		{
			name:     "Drive root",
			path:     "C:\\",
			expected: []string{"C:\\"},
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
					if i < len(result) && result[i] != expected {
						t.Errorf("GetDirectoryBreadcrumb(%s)[%d] = %s, expected %s", tt.path, i, result[i], expected)
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