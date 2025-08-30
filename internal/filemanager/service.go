package filemanager

import (
	"encoding/base64"
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
	
	// Ensure we don't go above drive root on Windows
	if parent == "." || (len(parent) == 2 && parent[1] == ':') {
		return cleanPath
	}
	
	return parent
}

// BreadcrumbItem represents a single item in breadcrumb navigation
type BreadcrumbItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// GetDirectoryBreadcrumb creates a breadcrumb navigation for the path
func (s *Service) GetDirectoryBreadcrumb(path string) []BreadcrumbItem {
	cleanPath := filepath.Clean(path)
	parts := strings.Split(cleanPath, string(filepath.Separator))
	
	var breadcrumb []BreadcrumbItem
	currentPath := ""
	
	for i, part := range parts {
		if part == "" {
			continue
		}
		
		var displayName string
		if i == 0 && strings.Contains(part, ":") {
			// Drive letter
			currentPath = part + "\\"
			displayName = part
		} else {
			currentPath = filepath.Join(currentPath, part)
			displayName = part
		}
		
		breadcrumb = append(breadcrumb, BreadcrumbItem{
			Name: displayName,
			Path: currentPath,
		})
	}
	
	return breadcrumb
}

// NavigationResponse represents a response for file manager navigation
type NavigationResponse struct {
	Content        string             `json:"content"`
	Context        *NavigationContext `json:"context"`
	RequiresUpdate bool               `json:"requires_update"`
}

// NavigationContext provides context for file manager navigation
type NavigationContext struct {
	CurrentPath     string           `json:"current_path"`
	ParentPath      string           `json:"parent_path"`
	Breadcrumbs     []BreadcrumbItem `json:"breadcrumbs"`
	CanNavigateUp   bool             `json:"can_navigate_up"`
	CurrentPage     int              `json:"current_page"`
	TotalPages      int              `json:"total_pages"`
	ViewHistory     []string         `json:"view_history"`      // For back button functionality
	LastAction      string           `json:"last_action"`       // Track user's last action
	TotalFiles      int              `json:"total_files"`
	TotalDirectories int             `json:"total_directories"`
}

// UserPreferences stores user-specific file manager preferences
type UserPreferences struct {
	PageSize        int    `json:"page_size"`
	SortBy          string `json:"sort_by"`    // name, size, date
	SortOrder       string `json:"sort_order"` // asc, desc
	ShowHiddenFiles bool   `json:"show_hidden_files"`
	ViewMode        string `json:"view_mode"`  // list, details
}

// GetNavigationContext returns enhanced navigation context for a given path
func (s *Service) GetNavigationContext(path string) *NavigationContext {
	cleanPath := filepath.Clean(path)
	parentPath := s.GetParentDirectory(cleanPath)
	canNavigateUp := parentPath != cleanPath && len(parentPath) > 0
	
	return &NavigationContext{
		CurrentPath:   cleanPath,
		ParentPath:    parentPath,
		Breadcrumbs:   s.GetDirectoryBreadcrumb(cleanPath),
		CanNavigateUp: canNavigateUp,
		CurrentPage:   1,
		TotalPages:    1,
		ViewHistory:   []string{},
		LastAction:    "navigate",
	}
}

// GetNavigationContextWithPagination returns navigation context with pagination info
func (s *Service) GetNavigationContextWithPagination(path string, page int, result *PaginatedDirectoryResult) *NavigationContext {
	cleanPath := filepath.Clean(path)
	parentPath := s.GetParentDirectory(cleanPath)
	canNavigateUp := parentPath != cleanPath && len(parentPath) > 0
	
	// Count directories and files
	dirCount := 0
	fileCount := 0
	for _, file := range result.Files {
		if file.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}
	
	return &NavigationContext{
		CurrentPath:      cleanPath,
		ParentPath:       parentPath,
		Breadcrumbs:      s.GetDirectoryBreadcrumb(cleanPath),
		CanNavigateUp:    canNavigateUp,
		CurrentPage:      result.CurrentPage,
		TotalPages:       result.TotalPages,
		ViewHistory:      []string{},
		LastAction:       "navigate",
		TotalFiles:       fileCount,
		TotalDirectories: dirCount,
	}
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

// PaginatedDirectoryResult represents a paginated directory listing
type PaginatedDirectoryResult struct {
	Files       []FileInfo `json:"files"`
	TotalFiles  int        `json:"total_files"`
	CurrentPage int        `json:"current_page"`
	TotalPages  int        `json:"total_pages"`
	PageSize    int        `json:"page_size"`
	HasNext     bool       `json:"has_next"`
	HasPrev     bool       `json:"has_prev"`
}

// ListDirectoryPaginated lists files and directories with pagination support
func (s *Service) ListDirectoryPaginated(path string, page int, pageSize int) (*PaginatedDirectoryResult, error) {
	// Get all files first
	allFiles, err := s.ListDirectory(path)
	if err != nil {
		return nil, err
	}
	
	// Calculate pagination
	totalFiles := len(allFiles)
	if pageSize <= 0 {
		pageSize = 20 // Default page size
	}
	
	if page < 1 {
		page = 1
	}
	
	totalPages := (totalFiles + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	
	// Calculate start and end indices
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize
	
	if startIdx >= totalFiles {
		// Page is beyond available data, return empty result
		return &PaginatedDirectoryResult{
			Files:       []FileInfo{},
			TotalFiles:  totalFiles,
			CurrentPage: page,
			TotalPages:  totalPages,
			PageSize:    pageSize,
			HasNext:     false,
			HasPrev:     page > 1,
		}, nil
	}
	
	if endIdx > totalFiles {
		endIdx = totalFiles
	}
	
	// Extract the page slice
	pageFiles := allFiles[startIdx:endIdx]
	
	return &PaginatedDirectoryResult{
		Files:       pageFiles,
		TotalFiles:  totalFiles,
		CurrentPage: page,
		TotalPages:  totalPages,
		PageSize:    pageSize,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}, nil
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

// Callback data encoding/decoding utilities

// EncodePathForCallback encodes a file path for use in callback data
func (s *Service) EncodePathForCallback(path string) string {
	return base64.URLEncoding.EncodeToString([]byte(path))
}

// DecodePathFromCallback decodes a file path from callback data
func (s *Service) DecodePathFromCallback(encoded string) (string, error) {
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode callback path: %w", err)
	}
	
	path := string(decoded)
	cleanPath := filepath.Clean(path)
	
	// Validate path security
	if !s.isDriveAllowed(cleanPath) {
		return "", fmt.Errorf("access to drive not allowed")
	}
	
	return cleanPath, nil
}

// ValidateCallbackPath validates if a decoded path is safe and accessible
func (s *Service) ValidateCallbackPath(path string) error {
	cleanPath := filepath.Clean(path)
	
	// Check drive access
	if !s.isDriveAllowed(cleanPath) {
		return fmt.Errorf("access to drive not allowed")
	}
	
	// Check if path exists
	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("path not accessible: %w", err)
	}
	
	return nil
}

// GetDriveSelectionResponse returns navigation response for drive selection
func (s *Service) GetDriveSelectionResponse() *NavigationResponse {
	drives := s.GetAvailableDrives()
	
	content := "📁 *File Manager - Drive Selection*\n\n"
	if len(drives) == 0 {
		content += "❌ No drives available in configuration"
		return &NavigationResponse{
			Content:        content,
			RequiresUpdate: false,
		}
	}
	
	content += "💾 **Available Drives:**\n\n"
	for _, drive := range drives {
		content += fmt.Sprintf("• %s\\\n", drive)
	}
	content += "\n💡 *Click on a drive below to start browsing:*"
	
	return &NavigationResponse{
		Content:        content,
		RequiresUpdate: true,
	}
}

// GetDirectoryNavigationResponse returns navigation response for directory browsing
func (s *Service) GetDirectoryNavigationResponse(path string, page int) (*NavigationResponse, error) {
	if !s.IsValidPath(path) {
		return nil, fmt.Errorf("invalid or inaccessible path: %s", path)
	}
	
	pageSize := 15 // Default page size for better mobile experience
	result, err := s.ListDirectoryPaginated(path, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}
	
	context := s.GetNavigationContextWithPagination(path, page, result)
	content := s.generateDirectoryContent(context, result)
	
	return &NavigationResponse{
		Content:        content,
		Context:        context,
		RequiresUpdate: true,
	}, nil
}

// GetFileDetailsResponse returns navigation response for file details
func (s *Service) GetFileDetailsResponse(filePath string) (*NavigationResponse, error) {
	fileInfo, err := s.GetFileInfo(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	content := s.generateFileDetailsContent(fileInfo)
	
	return &NavigationResponse{
		Content:        content,
		RequiresUpdate: true,
	}, nil
}

// generateDirectoryContent creates the text content for directory listing
func (s *Service) generateDirectoryContent(context *NavigationContext, result *PaginatedDirectoryResult) string {
	content := fmt.Sprintf("📁 *Current Directory*\n`%s`\n\n", context.CurrentPath)
	
	// Add breadcrumb path
	if len(context.Breadcrumbs) > 0 {
		breadcrumb := "📍 **Path:** "
		for i, item := range context.Breadcrumbs {
			if i > 0 {
				breadcrumb += " > "
			}
			breadcrumb += item.Name
		}
		content += breadcrumb + "\n\n"
	}
	
	// Add directory statistics
	content += fmt.Sprintf("📊 **Contents:** %d folders, %d files", context.TotalDirectories, context.TotalFiles)
	
	// Add pagination info if needed
	if result.TotalPages > 1 {
		content += fmt.Sprintf(" (Page %d/%d)", result.CurrentPage, result.TotalPages)
	}
	content += "\n\n"
	
	if len(result.Files) == 0 {
		content += "📭 *This directory is empty*\n\n"
	} else {
		content += "💡 *Click on any item below to navigate:*\n"
	}
	
	return content
}

// generateFileDetailsContent creates content for file details view
func (s *Service) generateFileDetailsContent(fileInfo *FileInfo) string {
	content := "📄 *File Details*\n\n"
	content += fmt.Sprintf("📛 **Name:** %s\n", fileInfo.Name)
	content += fmt.Sprintf("📏 **Size:** %s\n", FormatSize(fileInfo.Size))
	content += fmt.Sprintf("📅 **Modified:** %s\n", fileInfo.ModTime.Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("🔒 **Permissions:** %s\n", fileInfo.Mode)
	content += fmt.Sprintf("📍 **Path:** `%s`\n\n", fileInfo.Path)
	
	return content
}
