package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Bot         BotConfig         `yaml:"bot"`
	Database    DatabaseConfig    `yaml:"database"`
	Users       UsersConfig       `yaml:"users"`
	FileManager FileManagerConfig `yaml:"file_manager"`
	Screenshot  ScreenshotConfig  `yaml:"screenshot"`
	Events      EventsConfig      `yaml:"events"`
}

type BotConfig struct {
	Token string `yaml:"token"`
	Debug bool   `yaml:"debug"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type UsersConfig struct {
	AdminUserIDs []int64 `yaml:"admin_user_ids"`
	AllowedUsers []int64 `yaml:"allowed_users"`
}

type FileManagerConfig struct {
	AllowedDrives  []string `yaml:"allowed_drives"`
	MaxFileSize    int64    `yaml:"max_file_size"`   // bytes
	AllowedActions []string `yaml:"allowed_actions"` // list, download, upload, delete
	DownloadPath   string   `yaml:"download_path"`
	UploadPath     string   `yaml:"upload_path"`
}

type ScreenshotConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Quality     int    `yaml:"quality"` // 1-100
	Format      string `yaml:"format"`  // png, jpg
	MaxWidth    int    `yaml:"max_width"`
	MaxHeight   int    `yaml:"max_height"`
	StoragePath string `yaml:"storage_path"`
}

type EventsConfig struct {
	Enabled         bool     `yaml:"enabled"`
	NotifyUsers     []int64  `yaml:"notify_users"`     // Users to notify about events
	WatchEvents     []string `yaml:"watch_events"`     // login, logout, startup, shutdown, error
	PollingInterval int      `yaml:"polling_interval"` // seconds
}

func Load(configPath string) (*Config, error) {
	// Сначала загружаем из файла
	config := &Config{}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Переопределяем значения переменными окружения
	if token := os.Getenv("BOT_TOKEN"); token != "" {
		config.Bot.Token = token
	}

	if debug := os.Getenv("BOT_DEBUG"); debug != "" {
		config.Bot.Debug = debug == "true"
	}

	if dbPath := os.Getenv("DB_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}

	if adminIDs := os.Getenv("ADMIN_USER_IDS"); adminIDs != "" {
		ids := parseUserIDs(adminIDs)
		if len(ids) > 0 {
			config.Users.AdminUserIDs = ids
		}
	}

	if allowedIDs := os.Getenv("ALLOWED_USER_IDS"); allowedIDs != "" {
		ids := parseUserIDs(allowedIDs)
		if len(ids) > 0 {
			config.Users.AllowedUsers = ids
		}
	}

	// Устанавливаем значения по умолчанию
	if config.Database.Path == "" {
		config.Database.Path = "cupbot.db"
	}

	// File Manager defaults
	if len(config.FileManager.AllowedDrives) == 0 {
		config.FileManager.AllowedDrives = []string{"C:", "D:"}
	}
	if config.FileManager.MaxFileSize == 0 {
		config.FileManager.MaxFileSize = 10 * 1024 * 1024 // 10MB
	}
	if len(config.FileManager.AllowedActions) == 0 {
		config.FileManager.AllowedActions = []string{"list", "download"}
	}
	if config.FileManager.DownloadPath == "" {
		config.FileManager.DownloadPath = "./downloads"
	}
	if config.FileManager.UploadPath == "" {
		config.FileManager.UploadPath = "./uploads"
	}

	// Screenshot defaults
	if config.Screenshot.Quality == 0 {
		config.Screenshot.Quality = 80
	}
	if config.Screenshot.Format == "" {
		config.Screenshot.Format = "png"
	}
	if config.Screenshot.MaxWidth == 0 {
		config.Screenshot.MaxWidth = 1920
	}
	if config.Screenshot.MaxHeight == 0 {
		config.Screenshot.MaxHeight = 1080
	}
	if config.Screenshot.StoragePath == "" {
		config.Screenshot.StoragePath = "./screenshots"
	}

	// Events defaults
	if config.Events.PollingInterval == 0 {
		config.Events.PollingInterval = 30 // 30 seconds
	}
	if len(config.Events.WatchEvents) == 0 {
		config.Events.WatchEvents = []string{"login", "logout", "error"}
	}
	if config.Events.NotifyUsers == nil {
		config.Events.NotifyUsers = make([]int64, 0)
	}

	// Ensure slices are never nil
	if config.Users.AdminUserIDs == nil {
		config.Users.AdminUserIDs = make([]int64, 0)
	}
	if config.Users.AllowedUsers == nil {
		config.Users.AllowedUsers = make([]int64, 0)
	}

	return config, nil
}

func parseUserIDs(s string) []int64 {
	parts := strings.Split(s, ",")
	ids := make([]int64, 0) // Always return non-nil slice

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if id, err := strconv.ParseInt(part, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}

	return ids
}

func (c *Config) IsAdmin(userID int64) bool {
	for _, id := range c.Users.AdminUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (c *Config) IsAllowed(userID int64) bool {
	// Администраторы всегда разрешены
	if c.IsAdmin(userID) {
		return true
	}

	// Если список разрешенных пользователей пуст, то разрешены только админы
	if len(c.Users.AllowedUsers) == 0 {
		return false
	}

	for _, id := range c.Users.AllowedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// IsDriveAllowed checks if a drive is in the allowed drives list
func (c *Config) IsDriveAllowed(drive string) bool {
	for _, allowedDrive := range c.FileManager.AllowedDrives {
		if allowedDrive == drive {
			return true
		}
	}
	return false
}

// IsActionAllowed checks if a file manager action is allowed
func (c *Config) IsActionAllowed(action string) bool {
	for _, allowedAction := range c.FileManager.AllowedActions {
		if allowedAction == action {
			return true
		}
	}
	return false
}

// IsEventWatched checks if an event type is being watched
func (c *Config) IsEventWatched(eventType string) bool {
	for _, watchedEvent := range c.Events.WatchEvents {
		if watchedEvent == eventType {
			return true
		}
	}
	return false
}

// ShouldNotifyUser checks if a user should be notified about events
func (c *Config) ShouldNotifyUser(userID int64) bool {
	for _, notifyUser := range c.Events.NotifyUsers {
		if notifyUser == userID {
			return true
		}
	}
	return false
}
