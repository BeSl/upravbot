//go:build windows

package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cupbot/cupbot/internal/bot"
	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type cupBotService struct {
	bot    *bot.Bot
	db     *database.DB
	ctx    context.Context
	cancel context.CancelFunc
}

func (m *cupBotService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// Initialize the service
	if err := m.start(); err != nil {
		elog.Error(1, fmt.Sprintf("Failed to start CupBot: %v", err))
		return false, 1
	}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	elog.Info(1, "CupBot service started successfully")

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, "CupBot service stopping...")
				m.stop()
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				elog.Info(1, "CupBot service paused")
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				elog.Info(1, "CupBot service resumed")
			default:
				elog.Error(1, fmt.Sprintf("Unexpected control request #%d", c))
			}
		case <-m.ctx.Done():
			break loop
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	elog.Info(1, "CupBot service stopped")
	return false, 0
}

func (m *cupBotService) start() error {
	// Get executable directory for config file
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, "config", "config.yaml")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Bot.Token == "" {
		return fmt.Errorf("bot token is required")
	}

	// Initialize database
	dbPath := cfg.Database.Path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(execDir, dbPath)
	}

	m.db, err = database.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create bot
	m.bot, err = bot.New(cfg, m.db)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Create context for graceful shutdown
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Start bot in a goroutine
	go func() {
		if err := m.bot.Start(); err != nil {
			elog.Error(1, fmt.Sprintf("Bot error: %v", err))
		}
	}()

	return nil
}

func (m *cupBotService) stop() {
	if m.cancel != nil {
		m.cancel()
	}
	if m.bot != nil {
		m.bot.Stop()
	}
	if m.db != nil {
		m.db.Close()
	}
}

func RunService(name string, isDebug bool) error {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return err
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting %s service", name))

	service := &cupBotService{}

	if isDebug {
		err = debug.Run(name, service)
	} else {
		err = svc.Run(name, service)
	}

	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return err
	}

	elog.Info(1, fmt.Sprintf("%s service stopped", name))
	return nil
}

func RunInteractive() error {
	// Get executable directory for config file
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	configPath := filepath.Join(execDir, "config", "config.yaml")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Bot.Token == "" {
		return fmt.Errorf("bot token is required")
	}

	// Initialize database
	dbPath := cfg.Database.Path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(execDir, dbPath)
	}

	db, err := database.New(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Create bot
	cupBot, err := bot.New(cfg, db)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bot in goroutine
	go func() {
		if err := cupBot.Start(); err != nil {
			log.Printf("Bot error: %v", err)
		}
	}()

	log.Println("CupBot is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Stop bot gracefully
	cupBot.Stop()
	log.Println("Bot stopped successfully")

	return nil
}
