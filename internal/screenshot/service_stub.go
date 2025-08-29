//go:build !windows

package screenshot

import (
	"fmt"
	"time"

	"github.com/cupbot/cupbot/internal/config"
)

// ScreenshotInfo represents information about a screenshot file
type ScreenshotInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// Service provides screenshot functionality (stub for non-Windows platforms)
type Service struct {
	config *config.Config
}

// NewService creates a new screenshot service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// TakeScreenshot captures the desktop and saves it as an image file
func (s *Service) TakeScreenshot() (string, error) {
	return "", fmt.Errorf("screenshot functionality is only available on Windows")
}

// GetScreenshotList returns a list of available screenshots
func (s *Service) GetScreenshotList() ([]ScreenshotInfo, error) {
	return nil, fmt.Errorf("screenshot functionality is only available on Windows")
}

// DeleteScreenshot deletes a screenshot file
func (s *Service) DeleteScreenshot(filename string) error {
	return fmt.Errorf("screenshot functionality is only available on Windows")
}