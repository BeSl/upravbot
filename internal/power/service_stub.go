//go:build !windows

package power

import (
	"errors"
	"time"

	"github.com/cupbot/cupbot/internal/config"
)

// OperationType represents the type of power operation
type OperationType string

const (
	OperationShutdown      OperationType = "shutdown"
	OperationReboot        OperationType = "reboot"
	OperationForceShutdown OperationType = "force_shutdown"
	OperationForceReboot   OperationType = "force_reboot"
)

// Operation represents a scheduled power operation
type Operation struct {
	Type        OperationType `json:"type"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	UserID      int64         `json:"user_id"`
	Message     string        `json:"message"`
	Timeout     time.Duration `json:"timeout"`
}

// Service provides power management functionality (stub for non-Windows)
type Service struct {
	config *config.Config
}

// NewService creates a new power management service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// ScheduleShutdown schedules a system shutdown (not supported on non-Windows)
func (s *Service) ScheduleShutdown(userID int64, delay time.Duration, force bool) error {
	return errors.New("power management is only supported on Windows")
}

// ScheduleReboot schedules a system reboot (not supported on non-Windows)
func (s *Service) ScheduleReboot(userID int64, delay time.Duration, force bool) error {
	return errors.New("power management is only supported on Windows")
}

// CancelScheduledOperation cancels any scheduled power operation (not supported on non-Windows)
func (s *Service) CancelScheduledOperation() error {
	return errors.New("power management is only supported on Windows")
}

// GetScheduledOperation returns the currently scheduled operation (always nil on non-Windows)
func (s *Service) GetScheduledOperation() *Operation {
	return nil
}

// GetPowerStatus returns information about the current power state
func (s *Service) GetPowerStatus() map[string]interface{} {
	return map[string]interface{}{
		"supported":           false,
		"platform":           "non-windows",
		"scheduled_operation": nil,
	}
}