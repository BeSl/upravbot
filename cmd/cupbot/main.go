package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cupbot/cupbot/internal/bot"
	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
)

func main() {
	// Парсинг флагов командной строки
	configPath := flag.String("config", "config/config.yaml", "Путь к файлу конфигурации")
	flag.Parse()

	// Загрузка конфигурации
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Проверка токена бота
	if cfg.Bot.Token == "" {
		log.Fatal("Bot token is required. Set BOT_TOKEN environment variable or configure it in config.yaml")
	}

	// Инициализация базы данных
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Создание и запуск бота
	cupBot, err := bot.New(cfg, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Обработка сигналов для graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запуск бота в отдельной горутине
	go func() {
		if err := cupBot.Start(); err != nil {
			log.Printf("Bot error: %v", err)
		}
	}()

	log.Println("CupBot is running. Press Ctrl+C to stop.")

	// Ожидание сигнала завершения
	<-sigChan
	log.Println("Shutting down...")

	// Остановка бота
	cupBot.Stop()
	log.Println("Bot stopped successfully")
}
