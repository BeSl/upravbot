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
	switch callback.Data {
	case "status":
		response, success = b.handleStatusCallback(user)
	case "uptime":
		response, success = b.handleUptimeCallback(user)
	case "history":
		response, success = b.handleHistoryCallback(user)
	case "users":
		response, success = b.handleUsersCallback(user)
	case "stats":
		response, success = b.handleStatsCallback(user)
	case "admin_menu":
		response, success = b.handleAdminMenuCallback(user)
	case "main_menu":
		response, success = b.handleMainMenuCallback(user)
	case "files":
		response, success = b.handleFilesCallback(user)
	case "screenshot":
		response, success = b.handleScreenshotCallback(user)
	case "events":
		response, success = b.handleEventsCallback(user)
	case "menu":
		response, success = b.handleMenuCallback(user)
	default:
		response = "Неизвестная команда"
		success = false
	}

	// Отправляем ответ
	if response != "" {
		msg := tgbotapi.NewMessage(callback.Message.Chat.ID, response)
		msg.ParseMode = tgbotapi.ModeMarkdown

		// Добавляем кнопку меню после каждого callback ответа
		if callback.Data != "main_menu" && callback.Data != "admin_menu" && callback.Data != "menu" {
			msg.ReplyMarkup = b.getMenuKeyboard()
		}

		// Для меню показываем соответствующую клавиатуру
		if callback.Data == "main_menu" || callback.Data == "menu" {
			msg.ReplyMarkup = b.getMainKeyboard(user.IsAdmin)
		} else if callback.Data == "admin_menu" {
			msg.ReplyMarkup = b.getAdminKeyboard()
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
			tgbotapi.NewInlineKeyboardButtonData("👥 Manage Users", "users"),
			tgbotapi.NewInlineKeyboardButtonData("📊 View Stats", "stats"),
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
