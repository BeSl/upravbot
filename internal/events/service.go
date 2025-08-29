package events

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cupbot/cupbot/internal/config"
)

// EventType represents different types of system events
type EventType string

const (
	EventLogin    EventType = "login"
	EventLogout   EventType = "logout"
	EventStartup  EventType = "startup"
	EventShutdown EventType = "shutdown"
	EventError    EventType = "error"
	EventProcess  EventType = "process"
	EventService  EventType = "service"
)

// SystemEvent represents a system event
type SystemEvent struct {
	Type      EventType `json:"type"`
	Message   string    `json:"message"`
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"` // info, warning, error, critical
	Source    string    `json:"source"`
	UserID    string    `json:"user_id,omitempty"`
}

// EventHandler is a function that handles system events
type EventHandler func(event SystemEvent)

// Service provides system event monitoring
type Service struct {
	config   *config.Config
	handlers []EventHandler
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex

	// Event state tracking
	lastBootTime   time.Time
	lastLoginCheck time.Time
	knownProcesses map[string]bool
	knownServices  map[string]string
}

// NewService creates a new events service
func NewService(cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:         cfg,
		handlers:       make([]EventHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
		knownProcesses: make(map[string]bool),
		knownServices:  make(map[string]string),
	}
}

// AddHandler adds an event handler
func (s *Service) AddHandler(handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

// Start begins monitoring system events
func (s *Service) Start() error {
	if !s.config.Events.Enabled {
		log.Println("Events monitoring is disabled")
		return nil
	}

	log.Println("Starting system events monitoring...")

	// Initialize tracking data
	s.initializeTracking()

	// Start monitoring routines
	s.wg.Add(1)
	go s.monitorEvents()

	// Send startup event
	s.emitEvent(SystemEvent{
		Type:      EventStartup,
		Message:   "CupBot system monitoring started",
		Timestamp: time.Now(),
		Severity:  "info",
		Source:    "cupbot",
	})

	return nil
}

// Stop stops the events monitoring
func (s *Service) Stop() {
	log.Println("Stopping system events monitoring...")

	// Send shutdown event
	s.emitEvent(SystemEvent{
		Type:      EventShutdown,
		Message:   "CupBot system monitoring stopped",
		Timestamp: time.Now(),
		Severity:  "info",
		Source:    "cupbot",
	})

	s.cancel()
	s.wg.Wait()
}

// IsEventWatched checks if an event type is being monitored
func (s *Service) IsEventWatched(eventType EventType) bool {
	return s.config.IsEventWatched(string(eventType))
}

// monitorEvents runs the main event monitoring loop
func (s *Service) monitorEvents() {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.config.Events.PollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkEvents()
		}
	}
}

// checkEvents performs various system checks
func (s *Service) checkEvents() {
	// Check for login/logout events
	if s.IsEventWatched(EventLogin) || s.IsEventWatched(EventLogout) {
		s.checkLoginEvents()
	}

	// Check for process events
	if s.IsEventWatched(EventProcess) {
		s.checkProcessEvents()
	}

	// Check for service events
	if s.IsEventWatched(EventService) {
		s.checkServiceEvents()
	}

	// Check for errors in event log
	if s.IsEventWatched(EventError) {
		s.checkErrorEvents()
	}
}

// initializeTracking initializes the tracking state
func (s *Service) initializeTracking() {
	// Get current boot time
	s.lastBootTime = s.getSystemBootTime()
	s.lastLoginCheck = time.Now()

	// Initialize known processes
	processes := s.getCurrentProcesses()
	for _, proc := range processes {
		s.knownProcesses[proc] = true
	}

	// Initialize known services
	services := s.getCurrentServices()
	for name, status := range services {
		s.knownServices[name] = status
	}
}

// checkLoginEvents monitors for user login/logout events
func (s *Service) checkLoginEvents() {
	// Use PowerShell to check recent logon events
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-WinEvent -FilterHashtable @{LogName='Security'; ID=4624,4634; StartTime='%s'} -MaxEvents 10 | Select-Object TimeCreated, Id, Message",
			s.lastLoginCheck.Format("2006-01-02T15:04:05")))

	output, err := cmd.Output()
	if err != nil {
		return // No events or error
	}

	events := s.parseLogonEvents(string(output))
	for _, event := range events {
		s.emitEvent(event)
	}

	s.lastLoginCheck = time.Now()
}

// checkProcessEvents monitors for new/terminated processes
func (s *Service) checkProcessEvents() {
	currentProcesses := s.getCurrentProcesses()
	currentMap := make(map[string]bool)

	// Check for new processes
	for _, proc := range currentProcesses {
		currentMap[proc] = true
		if !s.knownProcesses[proc] {
			s.emitEvent(SystemEvent{
				Type:      EventProcess,
				Message:   fmt.Sprintf("New process started: %s", proc),
				Timestamp: time.Now(),
				Severity:  "info",
				Source:    "process_monitor",
			})
		}
	}

	// Check for terminated processes
	for proc := range s.knownProcesses {
		if !currentMap[proc] {
			s.emitEvent(SystemEvent{
				Type:      EventProcess,
				Message:   fmt.Sprintf("Process terminated: %s", proc),
				Timestamp: time.Now(),
				Severity:  "info",
				Source:    "process_monitor",
			})
		}
	}

	s.knownProcesses = currentMap
}

// checkServiceEvents monitors for service status changes
func (s *Service) checkServiceEvents() {
	currentServices := s.getCurrentServices()

	for name, status := range currentServices {
		if oldStatus, exists := s.knownServices[name]; exists {
			if oldStatus != status {
				s.emitEvent(SystemEvent{
					Type:      EventService,
					Message:   fmt.Sprintf("Service %s changed from %s to %s", name, oldStatus, status),
					Timestamp: time.Now(),
					Severity:  s.getServiceSeverity(status),
					Source:    "service_monitor",
				})
			}
		}
	}

	s.knownServices = currentServices
}

// checkErrorEvents monitors for system errors
func (s *Service) checkErrorEvents() {
	// Check system event log for recent errors
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2; StartTime='%s'} -MaxEvents 5 | Select-Object TimeCreated, LevelDisplayName, Message",
			time.Now().Add(-time.Duration(s.config.Events.PollingInterval*2)*time.Second).Format("2006-01-02T15:04:05")))

	output, err := cmd.Output()
	if err != nil {
		return
	}

	errorEvents := s.parseErrorEvents(string(output))
	for _, event := range errorEvents {
		s.emitEvent(event)
	}
}

// Helper methods for system information gathering

func (s *Service) getSystemBootTime() time.Time {
	cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -ClassName Win32_OperatingSystem).LastBootUpTime")
	output, err := cmd.Output()
	if err != nil {
		return time.Now()
	}

	bootTimeStr := strings.TrimSpace(string(output))
	bootTime, err := time.Parse("2006-01-02 15:04:05", bootTimeStr[:19])
	if err != nil {
		return time.Now()
	}

	return bootTime
}

func (s *Service) getCurrentProcesses() []string {
	cmd := exec.Command("powershell", "-Command", "Get-Process | Select-Object -ExpandProperty ProcessName")
	output, err := cmd.Output()
	if err != nil {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	processes := make([]string, 0, len(lines))
	for _, line := range lines {
		if proc := strings.TrimSpace(line); proc != "" {
			processes = append(processes, proc)
		}
	}

	return processes
}

func (s *Service) getCurrentServices() map[string]string {
	cmd := exec.Command("powershell", "-Command", "Get-Service | Select-Object Name, Status")
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}
	}

	services := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines[3:] { // Skip header lines
		if fields := strings.Fields(line); len(fields) >= 2 {
			name := fields[0]
			status := fields[len(fields)-1]
			services[name] = status
		}
	}

	return services
}

// Event parsing methods

func (s *Service) parseLogonEvents(output string) []SystemEvent {
	var events []SystemEvent
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "4624") { // Logon
			events = append(events, SystemEvent{
				Type:      EventLogin,
				Message:   "User logged in",
				Details:   line,
				Timestamp: time.Now(),
				Severity:  "info",
				Source:    "security_log",
			})
		} else if strings.Contains(line, "4634") { // Logoff
			events = append(events, SystemEvent{
				Type:      EventLogout,
				Message:   "User logged out",
				Details:   line,
				Timestamp: time.Now(),
				Severity:  "info",
				Source:    "security_log",
			})
		}
	}

	return events
}

func (s *Service) parseErrorEvents(output string) []SystemEvent {
	var events []SystemEvent
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		severity := "warning"
		if strings.Contains(strings.ToLower(line), "error") || strings.Contains(strings.ToLower(line), "critical") {
			severity = "error"
		}

		events = append(events, SystemEvent{
			Type:      EventError,
			Message:   "System error detected",
			Details:   line,
			Timestamp: time.Now(),
			Severity:  severity,
			Source:    "system_log",
		})
	}

	return events
}

// Helper methods

func (s *Service) getServiceSeverity(status string) string {
	switch strings.ToLower(status) {
	case "stopped":
		return "warning"
	case "running":
		return "info"
	default:
		return "info"
	}
}

// emitEvent sends an event to all registered handlers
func (s *Service) emitEvent(event SystemEvent) {
	s.mu.RLock()
	handlers := make([]EventHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}
