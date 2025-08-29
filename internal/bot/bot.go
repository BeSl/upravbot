package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cupbot/cupbot/internal/auth"
	"github.com/cupbot/cupbot/internal/config"
	"github.com/cupbot/cupbot/internal/database"
	"github.com/cupbot/cupbot/internal/events"
	"github.com/cupbot/cupbot/internal/filemanager"
	"github.com/cupbot/cupbot/internal/power"
	"github.com/cupbot/cupbot/internal/screenshot"
	"github.com/cupbot/cupbot/internal/system"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot представляет Telegram бота
type Bot struct {
	api               *tgbotapi.BotAPI
	config            *config.Config
	db                *database.DB
	authMw            *auth.Middleware
	systemService     *system.Service
	fileManager       *filemanager.Service
	screenshotService *screenshot.Service
	eventsService     *events.Service
	powerService      *power.Service
}

// New создает новый экземпляр бота
func New(cfg *config.Config, db *database.DB) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	api.Debug = cfg.Bot.Debug

	bot := &Bot{
		api:               api,
		config:            cfg,
		db:                db,
		authMw:            auth.NewMiddleware(cfg, db),
		systemService:     system.NewService(),
		fileManager:       filemanager.NewService(cfg),
		screenshotService: screenshot.NewService(cfg),
		eventsService:     events.NewService(cfg),
		powerService:      power.NewService(cfg),
	}

	log.Printf("Authorized on account %s", api.Self.UserName)
	return bot, nil
}

// Start запускает бота
func (b *Bot) Start() error {
	// Start events monitoring
	if err := b.eventsService.Start(); err != nil {
		log.Printf("Warning: Failed to start events service: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	log.Println("Bot started. Waiting for messages...")

	for update := range updates {
		go b.handleUpdate(update)
	}

	return nil
}

// Stop останавливает бота
func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
	b.eventsService.Stop()
	log.Println("Bot stopped")
}

// handleUpdate обрабатывает входящие обновления
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	// Авторизация пользователя
	authorized, user := b.authMw.AuthorizeUser(update)
	if !authorized {
		b.sendUnauthorizedMessage(update)
		return
	}

	if update.Message != nil {
		b.handleMessage(update.Message, user)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery, user)
	}
}

// handleMessage обрабатывает текстовые сообщения
func (b *Bot) handleMessage(message *tgbotapi.Message, user *database.User) {
	if !message.IsCommand() {
		return
	}

	command := message.Command()
	args := message.CommandArguments()

	log.Printf("User %d (%s) executed command: %s %s", user.ID, user.Username, command, args)

	var response string
	var success bool

	switch command {
	case "start":
		response, success = b.handleStart(message, user)
	case "help", "menu":
		response, success = b.handleHelp(message, user)
	case "status":
		response, success = b.handleStatusInternal(user)
	case "uptime":
		response, success = b.handleUptimeInternal(user)
	case "history":
		response, success = b.handleHistoryInternal(user, args)
	case "users":
		response, success = b.handleUsersInternal(user)
	case "stats":
		response, success = b.handleStatsInternal(user)
	case "cleanup":
		response, success = b.handleCleanup(message, user, args)
	case "addadmin":
		response, success = b.handleAddAdmin(message, user, args)
	case "removeadmin":
		response, success = b.handleRemoveAdmin(message, user, args)
	case "banuser":
		response, success = b.handleBanUser(message, user, args)
	case "unbanuser":
		response, success = b.handleUnbanUser(message, user, args)
	case "deleteuser":
		response, success = b.handleDeleteUser(message, user, args)
	case "files":
		response, success = b.handleFiles(message, user, args)
	case "screenshot":
		response, success = b.handleScreenshot(message, user, args)
	default:
		response = fmt.Sprintf("Неизвестная команда: %s\nИспользуйте /help для просмотра доступных команд", command)
		success = false
	}

	// Отправляем ответ
	if response != "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, response)
		msg.ParseMode = tgbotapi.ModeMarkdown

		// Добавляем кнопку меню после каждого ответа (кроме start)
		if command != "start" {
			msg.ReplyMarkup = b.getMenuKeyboard()
		}

		// Для команд help и menu показываем полную клавиатуру
		if command == "help" || command == "menu" {
			msg.ReplyMarkup = b.getMainKeyboard(user.IsAdmin)
		}

		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Failed to send message: %v", err)
		}
	}

	// Записываем в историю
	b.authMw.LogCommand(user.ID, command, args, success, response)
}

// handleCallbackQuery обрабатывает callback запросы
func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery, user *database.User) {
	// Отвечаем на callback
	callbackResponse := tgbotapi.NewCallback(callback.ID, "")
	b.api.Request(callbackResponse)

	var response string
	var success bool

	// Обрабатываем каллбэк данные
	switch {
	// Basic commands
	case callback.Data == "status":
		response, success = b.handleStatusCallback(user)
	case callback.Data == "uptime":
		response, success = b.handleUptimeCallback(user)
	case callback.Data == "history":
		response, success = b.handleHistoryCallback(user)
	case callback.Data == "users":
		response, success = b.handleUsersCallback(user)
	case callback.Data == "stats":
		response, success = b.handleStatsCallback(user)
	case callback.Data == "files":
		response, success = b.handleFilesCallback(user)
	case callback.Data == "screenshot":
		response, success = b.handleScreenshotCallback(user)
	case callback.Data == "events":
		response, success = b.handleEventsCallback(user)

	// Menu navigation
	case callback.Data == "admin_menu":
		response, success = b.handleAdminMenuCallback(user)
	case callback.Data == "main_menu":
		response, success = b.handleMainMenuCallback(user)
	case callback.Data == "menu":
		response, success = b.handleMenuCallback(user)

	// Power management
	case callback.Data == "power_menu":
		response, success = b.handlePowerMenuCallback(user)
	case callback.Data == "shutdown_now":
		response, success = b.handleShutdownNowCallback(user)
	case callback.Data == "shutdown_1min":
		response, success = b.handleShutdownDelayCallback(user, 1*time.Minute, false)
	case callback.Data == "shutdown_5min":
		response, success = b.handleShutdownDelayCallback(user, 5*time.Minute, false)
	case callback.Data == "shutdown_10min":
		response, success = b.handleShutdownDelayCallback(user, 10*time.Minute, false)
	case callback.Data == "shutdown_30min":
		response, success = b.handleShutdownDelayCallback(user, 30*time.Minute, false)
	case callback.Data == "reboot_now":
		response, success = b.handleRebootNowCallback(user)
	case callback.Data == "reboot_1min":
		response, success = b.handleRebootDelayCallback(user, 1*time.Minute, false)
	case callback.Data == "reboot_5min":
		response, success = b.handleRebootDelayCallback(user, 5*time.Minute, false)
	case callback.Data == "reboot_10min":
		response, success = b.handleRebootDelayCallback(user, 10*time.Minute, false)
	case callback.Data == "reboot_30min":
		response, success = b.handleRebootDelayCallback(user, 30*time.Minute, false)
	case callback.Data == "force_shutdown":
		response, success = b.handleShutdownDelayCallback(user, 0, true)
	case callback.Data == "force_reboot":
		response, success = b.handleRebootDelayCallback(user, 0, true)
	case callback.Data == "cancel_power":
		response, success = b.handleCancelPowerCallback(user)
	case callback.Data == "power_status":
		response, success = b.handlePowerStatusCallback(user)

	// User management
	case callback.Data == "user_menu":
		response, success = b.handleUserMenuCallback(user)
	case callback.Data == "add_admin_menu":
		response, success = b.handleAddAdminMenuCallback(user)
	case callback.Data == "remove_admin_menu":
		response, success = b.handleRemoveAdminMenuCallback(user)
	case callback.Data == "ban_user_menu":
		response, success = b.handleBanUserMenuCallback(user)
	case callback.Data == "unban_user_menu":
		response, success = b.handleUnbanUserMenuCallback(user)
	case callback.Data == "delete_user_menu":
		response, success = b.handleDeleteUserMenuCallback(user)
	case callback.Data == "list_users":
		response, success = b.handleListUsersCallback(user)

	// Enhanced services
	case callback.Data == "file_manager_admin":
		response, success = b.handleFileManagerAdminCallback(user)
	case callback.Data == "screenshot_admin":
		response, success = b.handleScreenshotAdminCallback(user)
	case callback.Data == "system_tools":
		response, success = b.handleSystemToolsCallback(user)

	default:
		response = "Неизвестная команда"
		success = false
	}

	// Отправляем ответ
	if response != "" {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, response)
		msg.ParseMode = tgbotapi.ModeMarkdown

		// Добавляем кнопку меню после каждого callback ответа
		if !isMenuCallback(callback.Data) && !isPowerCallback(callback.Data) && !isUserManagementCallback(callback.Data) {
			msg.ReplyMarkup = b.getMenuKeyboard()
		}

		// Для меню показываем соответствующую клавиатуру
		switch callback.Data {
		case "main_menu", "menu":
			msg.ReplyMarkup = b.getMainKeyboard(user.IsAdmin)
		case "admin_menu":
			msg.ReplyMarkup = b.getAdminKeyboard()
		case "power_menu":
			msg.ReplyMarkup = b.getPowerMenuKeyboard()
		case "user_menu":
			msg.ReplyMarkup = b.getUserManagementKeyboard()
		case "file_manager_admin":
			msg.ReplyMarkup = b.getFileManagerKeyboard()
		case "system_tools":
			msg.ReplyMarkup = b.getSystemToolsKeyboard()
		}

		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Failed to send callback response: %v", err)
		}
	}

	// Записываем в историю
	b.authMw.LogCommand(user.ID, "callback:"+callback.Data, "", success, response)

	log.Printf("Callback from user %d: %s", user.ID, callback.Data)
}

// sendUnauthorizedMessage отправляет сообщение о недостатке прав
func (b *Bot) sendUnauthorizedMessage(update tgbotapi.Update) {
	var chatID int64
	if update.Message != nil {
		chatID = update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		chatID = update.CallbackQuery.Message.Chat.ID
	} else {
		return
	}

	msg := tgbotapi.NewMessage(chatID, "❌ У вас нет прав для использования этого бота.")
	b.api.Send(msg)
}

// handleStart обрабатывает команду /start
func (b *Bot) handleStart(message *tgbotapi.Message, user *database.User) (string, bool) {
	welcome := fmt.Sprintf(`🤖 *Добро пожаловать в CupBot!*

Привет, %s! Этот бот позволяет удаленно управлять компьютером.

📊 *Основные возможности:*
• Просмотр статуса системы
• Мониторинг времени работы
• Просмотр истории команд`, user.FirstName)

	if user.IsAdmin {
		welcome += `

🔑 *Вы — администратор!*
• Управление пользователями
• Просмотр статистики
• Очистка данных`
	}

	welcome += `

📱 *Используйте кнопки ниже для управления:*`

	// Отправляем сообщение с клавиатурой
	msg := tgbotapi.NewMessage(message.Chat.ID, welcome)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = b.getMainKeyboard(user.IsAdmin)
	b.api.Send(msg)

	return "", true // Пустой ответ, так как мы уже отправили сообщение
}

// handleHelp обрабатывает команду /help
func (b *Bot) handleHelp(message *tgbotapi.Message, user *database.User) (string, bool) {
	help := `📖 *Справка по командам*

*Основные команды:*
/start - Начать работу с ботом
/help - Показать эту справку
/status - Полный статус системы
/uptime - Время работы системы
/history [N] - История команд (по умолчанию 10)
/files [путь] - Файловый менеджер
/screenshot - Создать скриншот рабочего стола`

	if user.IsAdmin {
		help += `

*Команды администратора:*
/users - Список всех пользователей
/stats - Статистика использования бота
/cleanup [дни] - Очистка истории старше N дней (по умолчанию 30)
/addadmin [ID] - Назначить администратора
/removeadmin [ID] - Убрать права администратора
/banuser [ID] - Заблокировать пользователя
/unbanuser [ID] - Разблокировать пользователя
/deleteuser [ID] - Удалить пользователя`
	}

	help += `

*Информация:*
• Все команды записываются в историю
• Только авторизованные пользователи могут использовать бота
• Администраторы имеют расширенный доступ`

	return help, true
}

// handleStatus обрабатывает команду /status
func (b *Bot) handleStatus(message *tgbotapi.Message, user *database.User) (string, bool) {
	sysInfo, err := b.systemService.GetSystemInfo()
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения информации о системе: %v", err), false
	}

	response := "💻 *Статус системы*\n\n"

	// Основная информация
	response += fmt.Sprintf("🖥️ *Хост:* %s\n", sysInfo.Hostname)
	response += fmt.Sprintf("🔧 *ОС:* %s %s\n", sysInfo.OS, sysInfo.Platform)
	response += fmt.Sprintf("⏰ *Время работы:* %s\n", formatDuration(sysInfo.Uptime))
	response += fmt.Sprintf("🔄 *Процессов:* %d\n\n", sysInfo.ProcessCount)

	// Информация о CPU
	response += "🧠 *Процессор:*\n"
	response += fmt.Sprintf("   • Модель: %s\n", sysInfo.CPUInfo.ModelName)
	response += fmt.Sprintf("   • Ядер: %d\n", sysInfo.CPUInfo.Cores)
	if len(sysInfo.CPUInfo.Usage) > 0 {
		avgUsage := 0.0
		for _, usage := range sysInfo.CPUInfo.Usage {
			avgUsage += usage
		}
		avgUsage /= float64(len(sysInfo.CPUInfo.Usage))
		response += fmt.Sprintf("   • Загрузка: %.1f%%\n", avgUsage)
	}
	if sysInfo.CPUInfo.Temperature > 0 {
		response += fmt.Sprintf("   • Температура: %.1f°C\n", sysInfo.CPUInfo.Temperature)
	}
	response += "\n"

	// Информация о памяти
	response += "🧮 *Память:*\n"
	response += fmt.Sprintf("   • Всего: %s\n", system.FormatBytes(sysInfo.MemoryInfo.Total))
	response += fmt.Sprintf("   • Используется: %s (%.1f%%)\n",
		system.FormatBytes(sysInfo.MemoryInfo.Used), sysInfo.MemoryInfo.UsedPercent)
	response += fmt.Sprintf("   • Доступно: %s\n\n", system.FormatBytes(sysInfo.MemoryInfo.Available))

	// Информация о дисках
	response += "💾 *Диски:*\n"
	for _, disk := range sysInfo.DiskInfo {
		if disk.Total > 0 {
			response += fmt.Sprintf("   • %s (%s)\n", disk.Device, disk.Fstype)
			response += fmt.Sprintf("     Всего: %s | Свободно: %s (%.1f%%)\n",
				system.FormatBytes(disk.Total), system.FormatBytes(disk.Free), 100-disk.UsedPercent)
		}
	}

	// Сетевая статистика (показываем только активные интерфейсы)
	activeInterfaces := 0
	for _, net := range sysInfo.NetworkInfo {
		if net.BytesSent > 0 || net.BytesRecv > 0 {
			activeInterfaces++
		}
	}

	if activeInterfaces > 0 {
		response += "\n🌐 *Сеть (активные интерфейсы):*\n"
		for _, net := range sysInfo.NetworkInfo {
			if net.BytesSent > 0 || net.BytesRecv > 0 {
				response += fmt.Sprintf("   • %s\n", net.Name)
				response += fmt.Sprintf("     Отправлено: %s | Получено: %s\n",
					system.FormatBytes(net.BytesSent), system.FormatBytes(net.BytesRecv))
			}
		}
	}

	return response, true
}
func (b *Bot) handleUptime(message *tgbotapi.Message, user *database.User) (string, bool) {
	return b.handleUptimeInternal(user)
}

func (b *Bot) handleUptimeInternal(user *database.User) (string, bool) {
	uptime, err := b.systemService.GetUptime()
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения времени работы: %v", err), false
	}

	return fmt.Sprintf("⏰ *Время работы системы:* %s", formatDuration(uptime)), true
}

// handleHistory обрабатывает команду /history
func (b *Bot) handleHistory(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	limit := 10
	if args != "" {
		if n, err := parseLimit(args); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}

	history, err := b.authMw.GetUserHistory(user.ID, limit)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения истории: %v", err), false
	}

	if len(history) == 0 {
		return "📝 История команд пуста", true
	}

	response := fmt.Sprintf("📝 *История команд* (последние %d):\n\n", len(history))
	for i, cmd := range history {
		status := "✅"
		if !cmd.Success {
			status = "❌"
		}
		response += fmt.Sprintf("%d. %s `/%s %s`\n   _Время: %s_\n\n",
			i+1, status, cmd.Command, cmd.Arguments, cmd.ExecutedAt.Format("02.01.2006 15:04:05"))
	}

	return response, true
}

// handleUsers обрабатывает команду /users (только админы)
func (b *Bot) handleUsers(message *tgbotapi.Message, user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	users, err := b.authMw.GetAllUsers(user.ID)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения списка пользователей: %v", err), false
	}

	if len(users) == 0 {
		return "👥 Список пользователей пуст", true
	}

	response := "👥 *Список пользователей:*\n\n"
	for i, u := range users {
		status := "🟢"
		if !u.IsActive {
			status = "🔴"
		}
		role := "Пользователь"
		if u.IsAdmin {
			role = "Администратор"
		}
		response += fmt.Sprintf("%d. %s *%s %s* (@%s)\n   ID: %d | %s\n   Создан: %s\n\n",
			i+1, status, u.FirstName, u.LastName, u.Username, u.ID, role,
			u.CreatedAt.Format("02.01.2006 15:04"))
	}

	return response, true
}

// handleStats обрабатывает команду /stats (только админы)
func (b *Bot) handleStats(message *tgbotapi.Message, user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	stats, err := b.authMw.GetStats(user.ID)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка получения статистики: %v", err), false
	}

	response := "📊 *Статистика использования:*\n\n"
	response += fmt.Sprintf("👥 Всего пользователей: %v\n", stats["total_users"])
	response += fmt.Sprintf("🟢 Активных пользователей: %v\n", stats["active_users"])
	response += fmt.Sprintf("📝 Всего команд: %v\n", stats["total_commands"])
	response += fmt.Sprintf("✅ Успешных команд: %v\n", stats["successful_commands"])
	response += fmt.Sprintf("🕐 Команд за 24 часа: %v\n", stats["recent_commands"])

	// Добавляем процент успешности
	if total := stats["total_commands"].(int); total > 0 {
		successful := stats["successful_commands"].(int)
		successRate := float64(successful) * 100 / float64(total)
		response += fmt.Sprintf("📈 Процент успешности: %.1f%%", successRate)
	}

	return response, true
}

// handleCleanup обрабатывает команду /cleanup (только админы)
func (b *Bot) handleCleanup(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	days := 30
	if args != "" {
		if n, err := parseLimit(args); err == nil && n > 0 && n <= 365 {
			days = n
		}
	}

	err := b.authMw.CleanupOldData(user.ID, days)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка очистки данных: %v", err), false
	}

	return fmt.Sprintf("🧹 Очистка завершена. Удалены записи старше %d дней.", days), true
}

// Вспомогательные функции

// formatDuration форматирует продолжительность в читаемый вид
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d дн. %d ч. %d мин.", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d ч. %d мин.", hours, minutes)
	}
	return fmt.Sprintf("%d мин.", minutes)
}

// parseLimit парсит строку в число
func parseLimit(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

// handleAddAdmin обрабатывает команду /addadmin (только админы)
func (b *Bot) handleAddAdmin(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	if args == "" {
		return "❌ Необходимо указать ID пользователя. Пример: /addadmin 123456789", false
	}

	userID, err := parseUserID(args)
	if err != nil {
		return "❌ Неверный ID пользователя", false
	}

	err = b.authMw.SetUserAdmin(user.ID, userID, true)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err), false
	}

	return fmt.Sprintf("✅ Пользователь %d назначен администратором", userID), true
}

// handleRemoveAdmin обрабатывает команду /removeadmin (только админы)
func (b *Bot) handleRemoveAdmin(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	if args == "" {
		return "❌ Необходимо указать ID пользователя. Пример: /removeadmin 123456789", false
	}

	userID, err := parseUserID(args)
	if err != nil {
		return "❌ Неверный ID пользователя", false
	}

	if userID == user.ID {
		return "❌ Нельзя убрать права администратора у себя", false
	}

	err = b.authMw.SetUserAdmin(user.ID, userID, false)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err), false
	}

	return fmt.Sprintf("✅ Права администратора у пользователя %d убраны", userID), true
}

// handleBanUser обрабатывает команду /banuser (только админы)
func (b *Bot) handleBanUser(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	if args == "" {
		return "❌ Необходимо указать ID пользователя. Пример: /banuser 123456789", false
	}

	userID, err := parseUserID(args)
	if err != nil {
		return "❌ Неверный ID пользователя", false
	}

	if userID == user.ID {
		return "❌ Нельзя заблокировать себя", false
	}

	err = b.authMw.SetUserActive(user.ID, userID, false)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err), false
	}

	return fmt.Sprintf("✅ Пользователь %d заблокирован", userID), true
}

// handleUnbanUser обрабатывает команду /unbanuser (только админы)
func (b *Bot) handleUnbanUser(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	if args == "" {
		return "❌ Необходимо указать ID пользователя. Пример: /unbanuser 123456789", false
	}

	userID, err := parseUserID(args)
	if err != nil {
		return "❌ Неверный ID пользователя", false
	}

	err = b.authMw.SetUserActive(user.ID, userID, true)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err), false
	}

	return fmt.Sprintf("✅ Пользователь %d разблокирован", userID), true
}

// handleDeleteUser обрабатывает команду /deleteuser (только админы)
func (b *Bot) handleDeleteUser(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if !user.IsAdmin {
		return "❌ Доступ запрещен. Требуются права администратора.", false
	}

	if args == "" {
		return "❌ Необходимо указать ID пользователя. Пример: /deleteuser 123456789", false
	}

	userID, err := parseUserID(args)
	if err != nil {
		return "❌ Неверный ID пользователя", false
	}

	if userID == user.ID {
		return "❌ Нельзя удалить себя", false
	}

	err = b.authMw.DeleteUser(user.ID, userID)
	if err != nil {
		return fmt.Sprintf("❌ Ошибка: %v", err), false
	}

	return fmt.Sprintf("✅ Пользователь %d удален", userID), true
}

// parseUserID парсит ID пользователя из строки
func parseUserID(s string) (int64, error) {
	var userID int64
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &userID)
	return userID, err
}

// getMainKeyboard returns the main keyboard
func (b *Bot) getMainKeyboard(isAdmin bool) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Basic buttons
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("💻 System Status", "status"),
		tgbotapi.NewInlineKeyboardButtonData("⏰ Uptime", "uptime"),
	})

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("📝 Command History", "history"),
		tgbotapi.NewInlineKeyboardButtonData("📁 File Manager", "files"),
	})

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("📸 Screenshot", "screenshot"),
		tgbotapi.NewInlineKeyboardButtonData("🔔 Events", "events"),
	})

	// Admin buttons
	if isAdmin {
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("👥 Users", "users"),
			tgbotapi.NewInlineKeyboardButtonData("📊 Statistics", "stats"),
		})
		rows = append(rows, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("🔑 Admin Menu", "admin_menu"),
		})
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// Callback handlers
func (b *Bot) handleStatusCallback(user *database.User) (string, bool) {
	return b.handleStatusInternal(user)
}

func (b *Bot) handleUptimeCallback(user *database.User) (string, bool) {
	return b.handleUptimeInternal(user)
}

func (b *Bot) handleHistoryCallback(user *database.User) (string, bool) {
	return b.handleHistoryInternal(user, "")
}

func (b *Bot) handleUsersCallback(user *database.User) (string, bool) {
	return b.handleUsersInternal(user)
}

func (b *Bot) handleStatsCallback(user *database.User) (string, bool) {
	return b.handleStatsInternal(user)
}

func (b *Bot) handleAdminMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied", false
	}
	return "🔑 *Admin Menu*\n\nSelect an action:", true
}

func (b *Bot) handleMainMenuCallback(user *database.User) (string, bool) {
	return fmt.Sprintf("🏠 *Main Menu*\n\nHello, %s! Choose an action:", user.FirstName), true
}

// Missing internal handler methods
func (b *Bot) handleStatusInternal(user *database.User) (string, bool) {
	return b.handleStatus(nil, user)
}

func (b *Bot) handleHistoryInternal(user *database.User, args string) (string, bool) {
	return b.handleHistory(nil, user, args)
}

func (b *Bot) handleUsersInternal(user *database.User) (string, bool) {
	return b.handleUsers(nil, user)
}

func (b *Bot) handleStatsInternal(user *database.User) (string, bool) {
	return b.handleStats(nil, user)
}

// getAdminKeyboard returns admin-specific keyboard
func (b *Bot) getAdminKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔌 Power Management", "power_menu"),
			tgbotapi.NewInlineKeyboardButtonData("👥 User Management", "user_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📁 File Manager+", "file_manager_admin"),
			tgbotapi.NewInlineKeyboardButtonData("📸 Screenshot+", "screenshot_admin"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("💻 System Monitoring", "status"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 System Tools", "system_tools"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🏠 Main Menu", "main_menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// getMenuKeyboard returns simple menu button
func (b *Bot) getMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📜 Menu", "menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// New service handlers
func (b *Bot) handleFiles(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	if args == "" {
		// List available drives
		drives := b.fileManager.GetAvailableDrives()
		if len(drives) == 0 {
			return "❌ No drives available in configuration", false
		}

		response := "📁 *File Manager*\n\nAvailable drives:\n"
		for _, drive := range drives {
			response += fmt.Sprintf("• %s\n", drive)
		}
		response += "\nUsage: `/files <drive>` to browse\nExample: `/files C:`"
		return response, true
	}

	// List directory contents
	files, err := b.fileManager.ListDirectory(args)
	if err != nil {
		return fmt.Sprintf("❌ Error listing directory: %v", err), false
	}

	response := fmt.Sprintf("📁 *Directory: %s*\n\n", args)
	for i, file := range files {
		if i >= 20 { // Limit to 20 items
			response += "... and more\n"
			break
		}
		icon := "📄"
		if file.IsDir {
			icon = "📁"
		}
		sizeStr := "<DIR>"
		if !file.IsDir {
			sizeStr = filemanager.FormatSize(file.Size)
		}
		response += fmt.Sprintf("%s %s (%s)\n", icon, file.Name, sizeStr)
	}

	return response, true
}

func (b *Bot) handleScreenshot(message *tgbotapi.Message, user *database.User, args string) (string, bool) {
	filename, err := b.screenshotService.TakeScreenshot()
	if err != nil {
		return fmt.Sprintf("❌ Error taking screenshot: %v", err), false
	}

	// Send screenshot as photo
	photo := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FilePath(filename))
	photo.Caption = fmt.Sprintf("📸 Desktop Screenshot\nTaken at: %s", time.Now().Format("2006-01-02 15:04:05"))

	if _, err := b.api.Send(photo); err != nil {
		return fmt.Sprintf("❌ Error sending screenshot: %v", err), false
	}

	return "📸 Screenshot taken and sent!", true
}

// Callback handlers for new services
func (b *Bot) handleFilesCallback(user *database.User) (string, bool) {
	drives := b.fileManager.GetAvailableDrives()
	if len(drives) == 0 {
		return "❌ No drives available in configuration", false
	}

	response := "📁 *File Manager*\n\nAvailable drives:\n"
	for _, drive := range drives {
		response += fmt.Sprintf("• %s\n", drive)
	}
	response += "\nUse command `/files <drive>` to browse\nExample: `/files C:`"
	return response, true
}

func (b *Bot) handleScreenshotCallback(user *database.User) (string, bool) {
	return "📸 *Screenshot Service*\n\nUse `/screenshot` command to take a desktop screenshot.", true
}

func (b *Bot) handleEventsCallback(user *database.User) (string, bool) {
	// For now, just return that the service is enabled
	status := "running"
	if !b.config.Events.Enabled {
		status = "disabled"
	}

	return fmt.Sprintf("🔔 *System Events Monitor*\n\nStatus: %s\n\nMonitoring system events and sending notifications.", status), true
}

func (b *Bot) handleMenuCallback(user *database.User) (string, bool) {
	return fmt.Sprintf("📜 *Menu*\n\nHello, %s! Choose an action:", user.FirstName), true
}

// Helper functions for callback type checking
func isMenuCallback(data string) bool {
	menuCallbacks := []string{"main_menu", "admin_menu", "menu", "power_menu", "user_menu", "file_manager_admin", "system_tools"}
	for _, callback := range menuCallbacks {
		if data == callback {
			return true
		}
	}
	return false
}

func isPowerCallback(data string) bool {
	powerCallbacks := []string{"power_menu", "shutdown_now", "shutdown_1min", "shutdown_5min", "shutdown_10min", "shutdown_30min",
		"reboot_now", "reboot_1min", "reboot_5min", "reboot_10min", "reboot_30min", "force_shutdown", "force_reboot", "cancel_power", "power_status"}
	for _, callback := range powerCallbacks {
		if data == callback {
			return true
		}
	}
	return false
}

func isUserManagementCallback(data string) bool {
	userCallbacks := []string{"user_menu", "add_admin_menu", "remove_admin_menu", "ban_user_menu", "unban_user_menu", "delete_user_menu", "list_users"}
	for _, callback := range userCallbacks {
		if data == callback {
			return true
		}
	}
	return false
}

func (b *Bot) getPowerMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("🔴 Shutdown Now", "shutdown_now"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Reboot Now", "reboot_now"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏱️ Shutdown in 1min", "shutdown_1min"),
			tgbotapi.NewInlineKeyboardButtonData("⏱️ Reboot in 1min", "reboot_1min"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⏰ Shutdown in 5min", "shutdown_5min"),
			tgbotapi.NewInlineKeyboardButtonData("⏰ Reboot in 5min", "reboot_5min"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🕒 Shutdown in 10min", "shutdown_10min"),
			tgbotapi.NewInlineKeyboardButtonData("🕒 Reboot in 10min", "reboot_10min"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🕥 Shutdown in 30min", "shutdown_30min"),
			tgbotapi.NewInlineKeyboardButtonData("🕥 Reboot in 30min", "reboot_30min"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⚠️ Force Shutdown", "force_shutdown"),
			tgbotapi.NewInlineKeyboardButtonData("⚠️ Force Reboot", "force_reboot"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel Operation", "cancel_power"),
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Power Status", "power_status"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Admin Menu", "admin_menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) getUserManagementKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("👥 List All Users", "list_users"),
			tgbotapi.NewInlineKeyboardButtonData("📊 User Statistics", "stats"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("➕ Add Administrator", "add_admin_menu"),
			tgbotapi.NewInlineKeyboardButtonData("➖ Remove Administrator", "remove_admin_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🚫 Ban User", "ban_user_menu"),
			tgbotapi.NewInlineKeyboardButtonData("✅ Unban User", "unban_user_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete User", "delete_user_menu"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Admin Menu", "admin_menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) getFileManagerKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📁 Browse Files", "files"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Admin Menu", "admin_menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) getSystemToolsKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("💻 System Status", "status"),
			tgbotapi.NewInlineKeyboardButtonData("⏰ Uptime", "uptime"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📝 Command History", "history"),
			tgbotapi.NewInlineKeyboardButtonData("🔔 System Events", "events"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("🔙 Admin Menu", "admin_menu"),
		},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// Power Management Callback Handlers
func (b *Bot) handlePowerMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	// Check current power status
	response := "🔌 *Power Management*\n\n"

	if op := b.powerService.GetScheduledOperation(); op != nil {
		timeLeft := time.Until(op.ScheduledAt)
		response += fmt.Sprintf("⚠️ *Active Operation:* %s\n", op.Type)
		response += fmt.Sprintf("⏰ *Time Remaining:* %v\n\n", timeLeft.Round(time.Second))
	}

	response += "Choose a power operation:"
	return response, true
}

func (b *Bot) handleShutdownNowCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	err := b.powerService.ScheduleShutdown(user.ID, 0, false)
	if err != nil {
		return fmt.Sprintf("❌ Error initiating shutdown: %v", err), false
	}

	return "🔴 *Immediate shutdown initiated*\n\nThe system will shut down now.", true
}

func (b *Bot) handleRebootNowCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	err := b.powerService.ScheduleReboot(user.ID, 0, false)
	if err != nil {
		return fmt.Sprintf("❌ Error initiating reboot: %v", err), false
	}

	return "🔄 *Immediate reboot initiated*\n\nThe system will restart now.", true
}

func (b *Bot) handleShutdownDelayCallback(user *database.User, delay time.Duration, force bool) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	err := b.powerService.ScheduleShutdown(user.ID, delay, force)
	if err != nil {
		return fmt.Sprintf("❌ Error scheduling shutdown: %v", err), false
	}

	if delay == 0 {
		if force {
			return "⚠️ *Force shutdown initiated*\n\nThe system will shut down immediately, closing all applications.", true
		}
		return "🔴 *Immediate shutdown initiated*\n\nThe system will shut down now.", true
	}

	actionType := "Shutdown"
	if force {
		actionType = "Force shutdown"
	}

	return fmt.Sprintf("⏰ *%s scheduled*\n\nThe system will shut down in %v.", actionType, delay), true
}

func (b *Bot) handleRebootDelayCallback(user *database.User, delay time.Duration, force bool) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	err := b.powerService.ScheduleReboot(user.ID, delay, force)
	if err != nil {
		return fmt.Sprintf("❌ Error scheduling reboot: %v", err), false
	}

	if delay == 0 {
		if force {
			return "⚠️ *Force reboot initiated*\n\nThe system will restart immediately, closing all applications.", true
		}
		return "🔄 *Immediate reboot initiated*\n\nThe system will restart now.", true
	}

	actionType := "Reboot"
	if force {
		actionType = "Force reboot"
	}

	return fmt.Sprintf("⏰ *%s scheduled*\n\nThe system will restart in %v.", actionType, delay), true
}

func (b *Bot) handleCancelPowerCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	err := b.powerService.CancelScheduledOperation()
	if err != nil {
		return fmt.Sprintf("❌ Error canceling operation: %v", err), false
	}

	return "✅ *Power operation canceled*\n\nAny scheduled shutdown or reboot has been canceled.", true
}

func (b *Bot) handlePowerStatusCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	status := b.powerService.GetPowerStatus()
	response := "ℹ️ *Power Management Status*\n\n"

	if op := b.powerService.GetScheduledOperation(); op != nil {
		timeLeft := time.Until(op.ScheduledAt)
		response += fmt.Sprintf("⚠️ *Active Operation:* %s\n", op.Type)
		response += fmt.Sprintf("👤 *Initiated by:* User %d\n", op.UserID)
		response += fmt.Sprintf("⏰ *Scheduled for:* %s\n", op.ScheduledAt.Format("15:04:05"))
		response += fmt.Sprintf("⏱️ *Time remaining:* %v\n", timeLeft.Round(time.Second))
	} else {
		response += "✅ No active power operations\n"
	}

	// Add platform-specific information
	if supported, exists := status["supported"]; exists && !supported.(bool) {
		response += "\n⚠️ Platform: Non-Windows (limited support)"
	} else {
		response += "\n🟢 Platform: Windows (full support)"
	}

	return response, true
}

// User Management Callback Handlers
func (b *Bot) handleUserMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "👥 *User Management*\n\nSelect a user management action:", true
}

func (b *Bot) handleAddAdminMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "➕ *Add Administrator*\n\nTo promote a user to administrator, use the command:\n`/addadmin [User_ID]`\n\nExample: `/addadmin 123456789`", true
}

func (b *Bot) handleRemoveAdminMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "➖ *Remove Administrator*\n\nTo remove admin privileges from a user, use the command:\n`/removeadmin [User_ID]`\n\nExample: `/removeadmin 123456789`", true
}

func (b *Bot) handleBanUserMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "🚫 *Ban User*\n\nTo ban a user from using the bot, use the command:\n`/banuser [User_ID]`\n\nExample: `/banuser 123456789`", true
}

func (b *Bot) handleUnbanUserMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "✅ *Unban User*\n\nTo unban a user and restore access, use the command:\n`/unbanuser [User_ID]`\n\nExample: `/unbanuser 123456789`", true
}

func (b *Bot) handleDeleteUserMenuCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "🗑️ *Delete User*\n\nTo permanently delete a user from the database, use the command:\n`/deleteuser [User_ID]`\n\nExample: `/deleteuser 123456789`\n\n⚠️ *Warning:* This action cannot be undone!", true
}

func (b *Bot) handleListUsersCallback(user *database.User) (string, bool) {
	return b.handleUsersInternal(user)
}

// Enhanced Service Callback Handlers
func (b *Bot) handleFileManagerAdminCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "📁 *Enhanced File Manager*\n\nAdmin file management features:\n\n• Browse all accessible drives\n• Upload and download files\n• View file details and permissions\n\nUse the buttons below or `/files` command to start browsing.", true
}

func (b *Bot) handleScreenshotAdminCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	// Check if running as service
	response := "📸 *Enhanced Screenshot Service*\n\n"

	// Try to take a screenshot to test functionality
	_, err := b.screenshotService.TakeScreenshot()
	if err != nil {
		if strings.Contains(err.Error(), "service") {
			response += "⚠️ *Service Mode Detected*\n\n"
			response += "Screenshots are not available when running as a Windows Service.\n\n"
			response += "📝 *Alternative:* Run CupBot in interactive mode to enable screenshots.\n\n"
			response += "🔧 *How to run interactively:*\n"
			response += "1. Stop the Windows service\n"
			response += "2. Run `cupbot.exe` directly from command line\n"
			response += "3. Screenshot functionality will be available"
			return response, false
		}
		response += fmt.Sprintf("❌ Error testing screenshot: %v\n\n", err)
	} else {
		response += "✅ Screenshot functionality is available\n\n"
	}

	response += "Admin screenshot features:\n\n"
	response += "• Capture full desktop\n"
	response += "• Configurable quality and format\n"
	response += "• Automatic timestamping\n\n"
	response += "Use `/screenshot` command to capture the desktop."

	return response, true
}

func (b *Bot) handleSystemToolsCallback(user *database.User) (string, bool) {
	if !user.IsAdmin {
		return "❌ Access denied: Admin privileges required", false
	}

	return "🔧 *System Tools*\n\nAdvanced system monitoring and management tools.\n\nSelect a tool from the menu below:", true
}
