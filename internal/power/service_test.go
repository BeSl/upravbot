package power

import (
	"runtime"
	"testing"
	"time"

	"github.com/cupbot/cupbot/internal/config"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{}
	service := NewService(cfg)

	if service == nil {
		t.Fatal("NewService returned nil")
	}

	if service.config != cfg {
		t.Error("Service config not set correctly")
	}
}

func TestScheduleShutdown(t *testing.T) {
	service := NewService(&config.Config{})

	tests := []struct {
		name       string
		userID     int64
		delay      time.Duration
		force      bool
		expectErr  bool
		skipOnNonWindows bool
	}{
		{
			name:             "schedule shutdown with delay",
			userID:           123,
			delay:            5 * time.Second,
			force:            false,
			expectErr:        runtime.GOOS != "windows",
			skipOnNonWindows: false,
		},
		{
			name:             "force shutdown with delay",
			userID:           123,
			delay:            3 * time.Second,
			force:            true,
			expectErr:        runtime.GOOS != "windows",
			skipOnNonWindows: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnNonWindows && runtime.GOOS != "windows" {
				t.Skip("Skipping Windows-specific test on non-Windows platform")
			}

			err := service.ScheduleShutdown(tt.userID, tt.delay, tt.force)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Clean up - cancel any scheduled operation
			if err == nil {
				service.CancelScheduledOperation()
			}
		})
	}
}

func TestScheduleReboot(t *testing.T) {
	service := NewService(&config.Config{})

	tests := []struct {
		name       string
		userID     int64
		delay      time.Duration
		force      bool
		expectErr  bool
	}{
		{
			name:      "schedule reboot with delay",
			userID:    123,
			delay:     5 * time.Second,
			force:     false,
			expectErr: runtime.GOOS != "windows",
		},
		{
			name:      "force reboot with delay",
			userID:    123,
			delay:     3 * time.Second,
			force:     true,
			expectErr: runtime.GOOS != "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ScheduleReboot(tt.userID, tt.delay, tt.force)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Clean up - cancel any scheduled operation
			if err == nil {
				service.CancelScheduledOperation()
			}
		})
	}
}

func TestCancelScheduledOperation(t *testing.T) {
	service := NewService(&config.Config{})

	// Test canceling when no operation is scheduled
	err := service.CancelScheduledOperation()
	if runtime.GOOS == "windows" {
		if err == nil || err.Error() != "no power operation scheduled" {
			t.Errorf("Expected 'no power operation scheduled' error, got: %v", err)
		}
	} else {
		if err == nil || err.Error() != "power management is only supported on Windows" {
			t.Errorf("Expected 'power management is only supported on Windows' error, got: %v", err)
		}
	}

	// On Windows, test canceling a scheduled operation
	if runtime.GOOS == "windows" {
		// Schedule an operation first
		err := service.ScheduleShutdown(123, 10*time.Second, false)
		if err != nil {
			t.Fatalf("Failed to schedule shutdown: %v", err)
		}

		// Verify operation is scheduled
		op := service.GetScheduledOperation()
		if op == nil {
			t.Error("Expected scheduled operation but got nil")
		}

		// Cancel the operation
		err = service.CancelScheduledOperation()
		if err != nil {
			t.Errorf("Failed to cancel operation: %v", err)
		}

		// Verify operation is canceled
		op = service.GetScheduledOperation()
		if op != nil {
			t.Error("Expected no scheduled operation after cancellation")
		}
	}
}

func TestGetScheduledOperation(t *testing.T) {
	service := NewService(&config.Config{})

	// Initially no operation should be scheduled
	op := service.GetScheduledOperation()
	if op != nil {
		t.Error("Expected no scheduled operation initially")
	}

	// On Windows, test with a scheduled operation
	if runtime.GOOS == "windows" {
		userID := int64(123)
		delay := 5 * time.Second

		err := service.ScheduleShutdown(userID, delay, false)
		if err != nil {
			t.Fatalf("Failed to schedule shutdown: %v", err)
		}

		op = service.GetScheduledOperation()
		if op == nil {
			t.Fatal("Expected scheduled operation but got nil")
		}

		if op.Type != OperationShutdown {
			t.Errorf("Expected operation type %s, got %s", OperationShutdown, op.Type)
		}

		if op.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, op.UserID)
		}

		if op.Timeout != delay {
			t.Errorf("Expected timeout %v, got %v", delay, op.Timeout)
		}

		// Clean up
		service.CancelScheduledOperation()
	}
}

func TestGetPowerStatus(t *testing.T) {
	service := NewService(&config.Config{})

	status := service.GetPowerStatus()
	if status == nil {
		t.Fatal("GetPowerStatus returned nil")
	}

	// Check common fields
	if _, exists := status["scheduled_operation"]; !exists {
		t.Error("Status should contain 'scheduled_operation' field")
	}

	// Platform-specific checks
	if runtime.GOOS == "windows" {
		if _, exists := status["privileges_set"]; !exists {
			t.Error("Windows status should contain 'privileges_set' field")
		}
	} else {
		if supported, exists := status["supported"]; !exists || supported != false {
			t.Error("Non-Windows status should have 'supported' field set to false")
		}
		if platform, exists := status["platform"]; !exists || platform != "non-windows" {
			t.Error("Non-Windows status should have 'platform' field set to 'non-windows'")
		}
	}
}

func TestOperationType(t *testing.T) {
	// Test operation type constants
	if OperationShutdown != "shutdown" {
		t.Errorf("Expected OperationShutdown to be 'shutdown', got '%s'", OperationShutdown)
	}
	if OperationReboot != "reboot" {
		t.Errorf("Expected OperationReboot to be 'reboot', got '%s'", OperationReboot)
	}
	if OperationForceShutdown != "force_shutdown" {
		t.Errorf("Expected OperationForceShutdown to be 'force_shutdown', got '%s'", OperationForceShutdown)
	}
	if OperationForceReboot != "force_reboot" {
		t.Errorf("Expected OperationForceReboot to be 'force_reboot', got '%s'", OperationForceReboot)
	}
}

func TestDoubleScheduling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	service := NewService(&config.Config{})

	// Schedule first operation
	err := service.ScheduleShutdown(123, 5*time.Second, false)
	if err != nil {
		t.Fatalf("Failed to schedule first shutdown: %v", err)
	}

	// Try to schedule second operation - should fail
	err = service.ScheduleReboot(456, 3*time.Second, false)
	if err == nil || err.Error() != "power operation already scheduled" {
		t.Errorf("Expected 'power operation already scheduled' error, got: %v", err)
	}

	// Clean up
	service.CancelScheduledOperation()
}