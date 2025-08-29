package filemanager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cupbot/cupbot/internal/config"
)

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

// Service provides file management operations
type Service struct {
	config *config.Config
}

// NewService creates a new file manager service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// ListDirectory lists files and directories in the specified path
func (s *Service) ListDirectory(path string) ([]FileInfo, error) {
	// Check if the drive is allowed
	if !s.isDriveAllowed(path) {
		return nil, fmt.Errorf("access to drive not allowed")
	}

	// Clean and validate path
	cleanPath := filepath.Clean(path)

	entries, err := os.ReadDir(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't read
		}

		fileInfo := FileInfo{
			Name:    info.Name(),
			Path:    filepath.Join(cleanPath, info.Name()),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime(),
			Mode:    info.Mode().String(),
		}
		files = append(files, fileInfo)
	}

	// Sort: directories first, then files, alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// GetFileInfo gets information about a specific file or directory
func (s *Service) GetFileInfo(path string) (*FileInfo, error) {
	if !s.isDriveAllowed(path) {
		return nil, fmt.Errorf("access to drive not allowed")
	}

	cleanPath := filepath.Clean(path)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &FileInfo{
		Name:    info.Name(),
		Path:    cleanPath,
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
		Mode:    info.Mode().String(),
	}, nil
}

// DownloadFile prepares a file for download (copies to download directory)
func (s *Service) DownloadFile(sourcePath string) (string, error) {
	if !s.config.IsActionAllowed("download") {
		return "", fmt.Errorf("download action not allowed")
	}

	if !s.isDriveAllowed(sourcePath) {
		return "", fmt.Errorf("access to drive not allowed")
	}

	cleanPath := filepath.Clean(sourcePath)

	// Check if it's a file
	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("cannot download directory")
	}

	// Check file size
	if info.Size() > s.config.FileManager.MaxFileSize {
		return "", fmt.Errorf("file too large (max: %d bytes)", s.config.FileManager.MaxFileSize)
	}

	// Create download directory if it doesn't exist
	downloadDir := s.config.FileManager.DownloadPath
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}

	// Generate unique filename
	filename := filepath.Base(cleanPath)
	timestamp := time.Now().Format("20060102_150405")
	downloadPath := filepath.Join(downloadDir, fmt.Sprintf("%s_%s", timestamp, filename))

	// Copy file
	if err := s.copyFile(cleanPath, downloadPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return downloadPath, nil
}

// UploadFile saves an uploaded file to the upload directory
func (s *Service) UploadFile(filename string, data io.Reader) (string, error) {
	if !s.config.IsActionAllowed("upload") {
		return "", fmt.Errorf("upload action not allowed")
	}

	// Create upload directory if it doesn't exist
	uploadDir := s.config.FileManager.UploadPath
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Generate safe filename
	safeFilename := s.sanitizeFilename(filename)
	timestamp := time.Now().Format("20060102_150405")
	uploadPath := filepath.Join(uploadDir, fmt.Sprintf("%s_%s", timestamp, safeFilename))

	// Create file
	file, err := os.Create(uploadPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data with size limit
	written, err := io.CopyN(file, data, s.config.FileManager.MaxFileSize+1)
	if err != nil && err != io.EOF {
		os.Remove(uploadPath) // Clean up on error
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	if written > s.config.FileManager.MaxFileSize {
		os.Remove(uploadPath) // Clean up oversized file
		return "", fmt.Errorf("file too large (max: %d bytes)", s.config.FileManager.MaxFileSize)
	}

	return uploadPath, nil
}

// DeleteFile deletes a file (if allowed)
func (s *Service) DeleteFile(path string) error {
	if !s.config.IsActionAllowed("delete") {
		return fmt.Errorf("delete action not allowed")
	}

	if !s.isDriveAllowed(path) {
		return fmt.Errorf("access to drive not allowed")
	}

	cleanPath := filepath.Clean(path)

	// Additional safety check - don't allow deleting system directories
	if s.isSystemPath(cleanPath) {
		return fmt.Errorf("cannot delete system files/directories")
	}

	return os.Remove(cleanPath)
}

// GetParentDirectory returns the parent directory path
func (s *Service) GetParentDirectory(path string) string {
	cleanPath := filepath.Clean(path)
	parent := filepath.Dir(cleanPath)
	
	// Don't go above drive root
	if len(parent) <= 3 && strings.Contains(parent, ":") {
		return parent
	}
	
	return parent
}

// GetDirectoryBreadcrumb creates a breadcrumb navigation for the path
func (s *Service) GetDirectoryBreadcrumb(path string) []string {
	cleanPath := filepath.Clean(path)
	parts := strings.Split(cleanPath, string(filepath.Separator))
	
	var breadcrumb []string
	currentPath := ""
	
	for i, part := range parts {
		if part == "" {
			continue
		}
		
		if i == 0 && strings.Contains(part, ":") {
			// Drive letter
			currentPath = part + "\\"
		} else {
			currentPath = filepath.Join(currentPath, part)
		}
		
		breadcrumb = append(breadcrumb, currentPath)
	}
	
	return breadcrumb
}

// IsValidPath checks if a path is valid and accessible
func (s *Service) IsValidPath(path string) bool {
	if !s.isDriveAllowed(path) {
		return false
	}
	
	cleanPath := filepath.Clean(path)
	_, err := os.Stat(cleanPath)
	return err == nil
}

// GetAvailableDrives returns list of available drives based on configuration
func (s *Service) GetAvailableDrives() []string {
	var availableDrives []string

	for _, drive := range s.config.FileManager.AllowedDrives {
		// Check if drive exists and is accessible
		if _, err := os.Stat(drive + "\\"); err == nil {
			availableDrives = append(availableDrives, drive)
		}
	}

	return availableDrives
}

// FormatSize formats file size in human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper methods

func (s *Service) isDriveAllowed(path string) bool {
	if len(s.config.FileManager.AllowedDrives) == 0 {
		return true // No restrictions
	}

	// Extract drive letter from path
	cleanPath := filepath.Clean(path)
	if len(cleanPath) >= 2 && cleanPath[1] == ':' {
		drive := strings.ToUpper(cleanPath[:2])
		return s.config.IsDriveAllowed(drive)
	}

	// If we can't determine the drive, allow it (might be relative path)
	return true
}

func (s *Service) isSystemPath(path string) bool {
	systemPaths := []string{
		"C:\\Windows",
		"C:\\System32",
		"C:\\Program Files",
		"C:\\Program Files (x86)",
		"C:\\ProgramData",
	}

	cleanPath := strings.ToUpper(filepath.Clean(path))
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(cleanPath, strings.ToUpper(sysPath)) {
			return true
		}
	}
	return false
}

func (s *Service) sanitizeFilename(filename string) string {
	// Remove dangerous characters
	unsafe := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	safe := filename
	for _, char := range unsafe {
		safe = strings.ReplaceAll(safe, char, "_")
	}
	return safe
}

func (s *Service) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
