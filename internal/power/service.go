//go:build windows

package power

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/cupbot/cupbot/internal/config"
)

var (
	advapi32                  = syscall.NewLazyDLL("advapi32.dll")
	user32                    = syscall.NewLazyDLL("user32.dll")
	procOpenProcessToken      = advapi32.NewProc("OpenProcessToken")
	procLookupPrivilegeValue  = advapi32.NewProc("LookupPrivilegeValueW")
	procAdjustTokenPrivileges = advapi32.NewProc("AdjustTokenPrivileges")
	procExitWindowsEx         = user32.NewProc("ExitWindowsEx")
	procAbortSystemShutdown   = advapi32.NewProc("AbortSystemShutdownW")
	procInitiateSystemShutdown = advapi32.NewProc("InitiateSystemShutdownExW")
)

const (
	// Token access rights
	TOKEN_ADJUST_PRIVILEGES = 0x0020
	TOKEN_QUERY             = 0x0008

	// Privilege names
	SE_SHUTDOWN_NAME = "SeShutdownPrivilege"

	// Shutdown flags
	EWX_LOGOFF   = 0x00000000
	EWX_SHUTDOWN = 0x00000001
	EWX_REBOOT   = 0x00000002
	EWX_FORCE    = 0x00000004
	EWX_POWEROFF = 0x00000008
	EWX_FORCEIFHUNG = 0x00000010

	// Shutdown reasons
	SHTDN_REASON_MAJOR_OTHER = 0x00000000
	SHTDN_REASON_MINOR_OTHER = 0x00000000
	SHTDN_REASON_FLAG_PLANNED = 0x80000000
)

type LUID struct {
	LowPart  uint32
	HighPart int32
}

type TOKEN_PRIVILEGES struct {
	PrivilegeCount uint32
	Privileges     [1]LUID_AND_ATTRIBUTES
}

type LUID_AND_ATTRIBUTES struct {
	Luid       LUID
	Attributes uint32
}

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

// Service provides power management functionality
type Service struct {
	config           *config.Config
	scheduledOp      *Operation
	mutex            sync.RWMutex
	shutdownTimer    *time.Timer
	privilegesSet    bool
}

// NewService creates a new power management service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// ScheduleShutdown schedules a system shutdown
func (s *Service) ScheduleShutdown(userID int64, delay time.Duration, force bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.scheduledOp != nil {
		return errors.New("power operation already scheduled")
	}

	opType := OperationShutdown
	if force {
		opType = OperationForceShutdown
	}

	operation := &Operation{
		Type:        opType,
		ScheduledAt: time.Now().Add(delay),
		UserID:      userID,
		Message:     "System shutdown initiated by CupBot",
		Timeout:     delay,
	}

	if delay == 0 {
		// Immediate shutdown
		return s.executeOperation(operation)
	}

	// Schedule shutdown
	s.scheduledOp = operation
	s.shutdownTimer = time.AfterFunc(delay, func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		
		if s.scheduledOp != nil && s.scheduledOp.Type == operation.Type {
			s.executeOperation(s.scheduledOp)
			s.scheduledOp = nil
		}
	})

	return nil
}

// ScheduleReboot schedules a system reboot
func (s *Service) ScheduleReboot(userID int64, delay time.Duration, force bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.scheduledOp != nil {
		return errors.New("power operation already scheduled")
	}

	opType := OperationReboot
	if force {
		opType = OperationForceReboot
	}

	operation := &Operation{
		Type:        opType,
		ScheduledAt: time.Now().Add(delay),
		UserID:      userID,
		Message:     "System reboot initiated by CupBot",
		Timeout:     delay,
	}

	if delay == 0 {
		// Immediate reboot
		return s.executeOperation(operation)
	}

	// Schedule reboot
	s.scheduledOp = operation
	s.shutdownTimer = time.AfterFunc(delay, func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		
		if s.scheduledOp != nil && s.scheduledOp.Type == operation.Type {
			s.executeOperation(s.scheduledOp)
			s.scheduledOp = nil
		}
	})

	return nil
}

// CancelScheduledOperation cancels any scheduled power operation
func (s *Service) CancelScheduledOperation() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.scheduledOp == nil {
		return errors.New("no power operation scheduled")
	}

	if s.shutdownTimer != nil {
		s.shutdownTimer.Stop()
		s.shutdownTimer = nil
	}

	// Try to abort system shutdown if it was initiated with InitiateSystemShutdown
	ret, _, _ := procAbortSystemShutdown.Call(0) // NULL for local computer
	if ret == 0 {
		// If abort fails, the shutdown may have been initiated with ExitWindowsEx
		// In that case, we can only notify that cancellation attempt was made
	}

	s.scheduledOp = nil
	return nil
}

// GetScheduledOperation returns the currently scheduled operation
func (s *Service) GetScheduledOperation() *Operation {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.scheduledOp == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	op := *s.scheduledOp
	return &op
}

// executeOperation executes the power operation
func (s *Service) executeOperation(op *Operation) error {
	if err := s.setShutdownPrivileges(); err != nil {
		return fmt.Errorf("failed to set shutdown privileges: %w", err)
	}

	var flags uint32

	switch op.Type {
	case OperationShutdown:
		flags = EWX_SHUTDOWN | EWX_POWEROFF
	case OperationReboot:
		flags = EWX_REBOOT
	case OperationForceShutdown:
		flags = EWX_SHUTDOWN | EWX_POWEROFF | EWX_FORCE
	case OperationForceReboot:
		flags = EWX_REBOOT | EWX_FORCE
	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	// Use InitiateSystemShutdown for better control and cancellation support
	if op.Type == OperationShutdown || op.Type == OperationReboot {
		messagePtr, _ := syscall.UTF16PtrFromString(op.Message)
		timeoutSeconds := uint32(op.Timeout.Seconds())
		forceApps := uint32(0)
		if op.Type == OperationForceShutdown || op.Type == OperationForceReboot {
			forceApps = 1
		}
		rebootAfterShutdown := uint32(0)
		if op.Type == OperationReboot || op.Type == OperationForceReboot {
			rebootAfterShutdown = 1
		}

		ret, _, err := procInitiateSystemShutdown.Call(
			0, // NULL for local computer
			uintptr(unsafe.Pointer(messagePtr)),
			uintptr(timeoutSeconds),
			uintptr(forceApps),
			uintptr(rebootAfterShutdown),
		)
		if ret == 0 {
			return fmt.Errorf("InitiateSystemShutdown failed: %v", err)
		}
	} else {
		// Use ExitWindowsEx for force operations
		ret, _, err := procExitWindowsEx.Call(
			uintptr(flags),
			uintptr(SHTDN_REASON_MAJOR_OTHER|SHTDN_REASON_MINOR_OTHER|SHTDN_REASON_FLAG_PLANNED),
		)
		if ret == 0 {
			return fmt.Errorf("ExitWindowsEx failed: %v", err)
		}
	}

	return nil
}

// setShutdownPrivileges enables shutdown privileges for the current process
func (s *Service) setShutdownPrivileges() error {
	if s.privilegesSet {
		return nil
	}

	var token syscall.Handle
	currentProcess := syscall.Handle(^uintptr(0)) // GetCurrentProcess() equivalent

	// Open process token
	ret, _, err := procOpenProcessToken.Call(
		uintptr(currentProcess),
		TOKEN_ADJUST_PRIVILEGES|TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)
	if ret == 0 {
		return fmt.Errorf("OpenProcessToken failed: %v", err)
	}
	defer syscall.CloseHandle(token)

	// Lookup shutdown privilege
	var luid LUID
	privilegeName, _ := syscall.UTF16PtrFromString(SE_SHUTDOWN_NAME)
	ret, _, err = procLookupPrivilegeValue.Call(
		0, // NULL for local system
		uintptr(unsafe.Pointer(privilegeName)),
		uintptr(unsafe.Pointer(&luid)),
	)
	if ret == 0 {
		return fmt.Errorf("LookupPrivilegeValue failed: %v", err)
	}

	// Adjust token privileges
	tp := TOKEN_PRIVILEGES{
		PrivilegeCount: 1,
		Privileges: [1]LUID_AND_ATTRIBUTES{
			{
				Luid:       luid,
				Attributes: 0x00000002, // SE_PRIVILEGE_ENABLED
			},
		},
	}

	ret, _, err = procAdjustTokenPrivileges.Call(
		uintptr(token),
		0, // FALSE - don't disable all privileges
		uintptr(unsafe.Pointer(&tp)),
		0, // BufferLength
		0, // PreviousState
		0, // ReturnLength
	)
	if ret == 0 {
		return fmt.Errorf("AdjustTokenPrivileges failed: %v", err)
	}

	s.privilegesSet = true
	return nil
}

// GetPowerStatus returns information about the current power state
func (s *Service) GetPowerStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := map[string]interface{}{
		"privileges_set": s.privilegesSet,
		"scheduled_operation": nil,
	}

	if s.scheduledOp != nil {
		status["scheduled_operation"] = map[string]interface{}{
			"type":         string(s.scheduledOp.Type),
			"scheduled_at": s.scheduledOp.ScheduledAt,
			"user_id":      s.scheduledOp.UserID,
			"timeout":      s.scheduledOp.Timeout.String(),
		}
	}

	return status
}