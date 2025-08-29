//go:build !windows

package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cupbot/cupbot/internal/bot"
	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
)

func RunService(name string, isDebug bool) error {
	return fmt.Errorf("Windows service functionality is only available on Windows")
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

	log.Printf("Starting CupBot in interactive mode...")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start bot in a goroutine
	go func() {
		if err := cupBot.Start(); err != nil {
			log.Printf("Bot error: %v", err)
			cancel()
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down...")

	cupBot.Stop()
	return nil
}