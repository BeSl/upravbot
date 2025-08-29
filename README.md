# CupBot - Advanced Telegram Bot for Windows Computer Management

CupBot is a comprehensive Telegram bot written in Go for remote Windows computer management. It features Windows service integration, user management, command history, file management, screenshot capabilities, system event notifications, power management, and an intuitive button-based admin interface.

## 🚀 **New Features**

### 🔌 **Power Management** (NEW!)
- ✅ **Remote Shutdown/Reboot** - control system power remotely
- ✅ **Scheduled Operations** - delay shutdown/reboot with configurable timeouts
- ✅ **Force Operations** - emergency power operations for unresponsive systems
- ✅ **Operation Cancellation** - cancel scheduled power operations
- ✅ **Power Status Monitoring** - view current scheduled operations
- ✅ **Admin-Only Access** - power operations restricted to administrators
- ✅ **Confirmation System** - safety confirmations for destructive operations

### 🎛️ **Enhanced Admin Menu System** (NEW!)
- ✅ **Comprehensive Admin Panel** - centralized control interface
- ✅ **Power Management Menu** - intuitive power control buttons
- ✅ **User Management Interface** - button-based user administration
- ✅ **Enhanced File Manager** - admin-specific file operations
- ✅ **System Tools Menu** - advanced monitoring and management
- ✅ **Service Context Detection** - smart feature availability based on runtime mode

### 📸 **Smart Screenshot Service** (ENHANCED!)
- ✅ **Service Mode Detection** - automatically detects Windows Service context
- ✅ **Interactive Mode Support** - full functionality when run interactively
- ✅ **Context-Aware Messages** - clear guidance on service limitations
- ✅ **Alternative Operation Suggestions** - helps users switch to interactive mode

### 🔧 **Windows Service Integration**
- ✅ **Run as Windows Service** - automatic startup and background operation
- ✅ **Service Management Scripts** - easy installation, uninstallation, and management
- ✅ **Event Log Integration** - proper Windows logging
- ✅ **Graceful Shutdown** - proper service lifecycle management

### 👥 **Advanced User Management**
- ✅ **Admin-only User Control** - only administrators can manage users
- ✅ **User Roles** - administrators and regular users
- ✅ **User Status Management** - activate/deactivate users
- ✅ **Safety Protections** - prevent removing the last admin

### 📱 **Modern Button Interface**
- ✅ **Interactive Buttons** - no more typing commands
- ✅ **Context-aware Menus** - different options for admins and users
- ✅ **Quick Actions** - instant access to system information
- ✅ **Admin Panel** - dedicated management interface
- ✅ **Menu Button** - added after each response for easy navigation

### 📁 **File Manager**
- ✅ **Browse Files and Directories** - explore filesystem remotely
- ✅ **Configurable Drive Access** - restrict access to specific drives
- ✅ **Security Controls** - protected system directories and size limits
- ✅ **File Operations** - list, download (configurable actions)

### 📸 **Screenshot Capability**
- ✅ **Desktop Screenshots** - capture current desktop state
- ✅ **Multiple Formats** - PNG/JPEG support with quality controls
- ✅ **Size Management** - automatic cleanup and storage limits
- ✅ **Instant Delivery** - screenshots sent directly to Telegram

### 🔔 **System Event Notifications**
- ✅ **Login/Logout Events** - monitor user sessions
- ✅ **Process Monitoring** - track system processes
- ✅ **Service Monitoring** - Windows service status changes
- ✅ **Error Detection** - system error log monitoring
- ✅ **Configurable Events** - choose what to monitor

## Возможности

### 🔧 Базовые функции
- ✅ **Статус системы** - просмотр полной информации о системе
- ✅ **Время работы** - получение uptime системы
- ✅ **История команд** - логирование и просмотр истории выполненных команд
- ✅ **Управление пользователями** - авторизация и разграничение доступа
- ✅ **Безопасность** - только авторизованные пользователи могут использовать бота
- ✅ **Файловый менеджер** - обзор файлов и папок с конфигурируемыми дисками
- ✅ **Скриншоты** - создание скриншотов рабочего стола
- ✅ **Уведомления о событиях** - мониторинг системных событий

### 📊 Мониторинг системы
- Информация о процессоре (модель, количество ядер, загрузка)
- Состояние оперативной памяти (общая, используемая, доступная)
- Информация о дисках (размер, свободное место, файловая система)
- Сетевая статистика (отправлено/получено данных)
- Количество активных процессов

### 👥 Управление пользователями
- Авторизация по Telegram ID
- Роли: администратор и обычный пользователь
- Логирование всех действий пользователей
- Статистика использования

## 🛠️ **Quick Installation & Setup**

> 📋 **For detailed GitHub repository setup instructions, see [REPOSITORY_SETUP.md](REPOSITORY_SETUP.md)**

### Option 1: Automated Installation (Recommended)
```bash
# 1. Build and install as Windows service
install.bat

# 2. Configure your bot token and admin ID
set BOT_TOKEN=your_bot_token_here
set ADMIN_USER_IDS=your_telegram_id_here
```

### Option 2: Manual Setup
```bash
# Build the project
build.bat

# Install as Windows service (run as Administrator)
install-service.bat

# Or run directly for testing
cupbot.exe
```

### Management Scripts
- **`build.bat`** - Build the project
- **`install-service.bat`** - Install as Windows service
- **`uninstall-service.bat`** - Remove the service
- **`service-manager.bat`** - Interactive service management
- **`install.bat`** - Complete build and install process

## 📋 **Usage**

### 📱 **Enhanced Button Interface**
CupBot features an advanced button-based interface with admin-specific capabilities:

#### **Regular User Interface**
- 💻 **System Status** - View complete system information
- ⏰ **Uptime** - Check system uptime
- 📝 **Command History** - View your recent commands
- 📁 **File Manager** - Browse files and directories
- 📸 **Screenshot** - Take desktop screenshots
- 🔔 **Events** - System event monitoring status
- 📜 **Menu** - Quick access menu button after each response

#### **Admin Interface** 🔑
- 🔌 **Power Management** - Control system shutdown/reboot
  - 🔴 Immediate shutdown/reboot
  - ⏱️ Scheduled operations (1min, 5min, 10min, 30min)
  - ⚠️ Force operations for emergency situations
  - ❌ Cancel active operations
  - ℹ️ View power operation status

- 👥 **User Management** - Complete user administration
  - 👥 List all users with status
  - ➕ Add new administrators
  - ➖ Remove administrator privileges
  - 🚫 Ban/unban users
  - 🗑️ Delete users (with warnings)
  - 📊 View user statistics

- 📁 **Enhanced File Manager** - Advanced file operations
  - 📂 Browse all accessible drives
  - 📄 Upload and download files
  - 🔍 View file details and permissions
  - 🚪 Safe access controls

- 📸 **Smart Screenshot Service**
  - ✅ Full desktop capture (interactive mode)
  - ⚠️ Service mode detection with helpful guidance
  - 🔧 Alternative operation instructions

- 🔧 **System Tools** - Advanced monitoring
  - 💻 Enhanced system status
  - 📝 Detailed command history
  - 🔔 System event monitoring
  - 📊 Performance statistics

### 🔒 **Power Management Features**

#### **Available Power Operations**
- **Immediate Shutdown** - Graceful system shutdown
- **Immediate Reboot** - Graceful system restart
- **Scheduled Shutdown/Reboot** - Delayed operations with multiple timeout options
- **Force Operations** - Emergency power operations (use with caution)
- **Operation Cancellation** - Cancel any scheduled power operation

#### **Safety Features**
- 🔒 **Admin-Only Access** - Power operations restricted to administrators
- ⚠️ **Confirmation Dialogs** - Clear warnings for destructive operations
- 📝 **Operation Logging** - All power operations logged for audit trail
- ⏰ **Status Monitoring** - Real-time view of scheduled operations
- ❌ **Cancellation Support** - Ability to cancel operations before execution

#### **Windows API Integration**
- **Native Windows Power Management** - Uses Windows API for reliable operations
- **Privilege Management** - Automatic elevation of shutdown privileges
- **Service Compatibility** - Works in both interactive and service modes
- **Cross-Platform Stubs** - Graceful handling on non-Windows platforms

### Admin User Management
Administrators can manage users through button interface or commands:

```bash
# Grant admin privileges
/addadmin 123456789

# Remove admin privileges
/removeadmin 123456789

# Ban/unban users
/banuser 123456789
/unbanuser 123456789

# Delete user (non-admins only)
/deleteuser 123456789
```

#### Способ 1: Переменные окружения (рекомендуется)
Создайте файл `.env` или установите переменные окружения:
```bash
set BOT_TOKEN=ваш_токен_бота
set ADMIN_USER_IDS=ваш_telegram_id
set ALLOWED_USER_IDS=список_разрешенных_пользователей
set DB_PATH=cupbot.db
set BOT_DEBUG=false
```

#### Способ 2: Файл конфигурации
Или отредактируйте `config/config.yaml`:
```yaml
bot:
  token: "ваш_токен_бота"
  debug: false

database:
  path: "cupbot.db"

users:
  admin_user_ids: [ваш_telegram_id]
  allowed_users: []  # пустой список = только админы

file_manager:
  # Разрешенные диски для файлового менеджера
  allowed_drives: ["C:", "D:"]
  
  # Максимальный размер загружаемого файла (в байтах)
  max_file_size: 10485760  # 10MB
  
  # Разрешенные действия: list, download, upload, delete
  allowed_actions: ["list", "download"]
  
  # Путь для скачанных файлов
  download_path: "./downloads"

screenshot:
  # Максимальный размер скриншота (в байтах)
  max_file_size: 5242880  # 5MB
  
  # Качество JPEG (если используется)
  jpeg_quality: 85
  
  # Папка для сохранения скриншотов
  storage_path: "./screenshots"
  
  # Максимальное количество сохраняемых скриншотов
  max_screenshots: 10

events:
  # Включить мониторинг событий
  enabled: true
  
  # Интервал опроса (секунды)
  polling_interval: 30
  
  # Отслеживаемые события
  watched_events: ["login", "logout", "error"]
  
  # Уведомлять пользователей
  notify_users: [ваш_telegram_id]
```

## 🔌 **Power Management Configuration**

Power management features are enabled by default for administrators. No additional configuration is required, but consider these security implications:

### **Safety Considerations**
- ⚠️ **Admin-Only Access**: Power operations are restricted to users with admin privileges
- 🔒 **Confirmation Required**: Immediate operations require explicit confirmation
- 📝 **Audit Logging**: All power operations are logged with user ID and timestamps
- ❌ **Cancellation Support**: Scheduled operations can be canceled before execution
- ⏱️ **Minimum Delays**: Consider implementing minimum delay requirements for scheduled operations

### **Service vs Interactive Mode**
- **Windows Service Mode**: Power management works in service mode with full functionality
- **Interactive Mode**: All features available, including screenshots
- **Context Detection**: The bot automatically detects its runtime context

### **Platform Support**
- **Windows**: Full power management support via Windows API
- **Non-Windows**: Graceful degradation with informative error messages

## 📸 **Screenshot Service Notes**

### **Service Context Limitations**
When running as a Windows Service, screenshot functionality is automatically disabled because:
- Services run in Session 0 (non-interactive desktop)
- No access to user desktop for screen capture
- Windows security model prevents desktop access from services

### **Solutions for Screenshot Access**
1. **Run Interactively**: Start CupBot from command line instead of as a service
2. **Dual Setup**: Run main bot as service, separate screenshot instance interactively
3. **User Session Service**: Configure service to run in user session (advanced)

### **Automatic Detection and Guidance**
The bot automatically:
- Detects when running as a Windows Service
- Provides clear error messages explaining limitations
- Suggests alternative approaches (switching to interactive mode)
- Shows step-by-step instructions for enabling screenshots

### 5. Получение вашего Telegram ID
1. Напишите [@userinfobot](https://t.me/userinfobot)
2. Скопируйте ваш ID
3. Добавьте его в конфигурацию как admin_user_id

### 6. Запуск бота
```bash
# Использование go run
go run main.go

# Или сборка и запуск
go build -o cupbot.exe main.go
cupbot.exe

# С указанием пути к конфигурации
cupbot.exe -config path/to/config.yaml
```

## Использование

### Доступные команды

#### Основные команды (все пользователи):
- `/start` - Начать работу с ботом
- `/help` - Показать справку по командам
- `/status` - Полный статус системы (CPU, память, диски, сеть)
- `/uptime` - Время работы системы
- `/history [N]` - История команд (по умолчанию 10 последних)
- `/files [путь]` - Файловый менеджер
- `/screenshot` - Создать скриншот рабочего стола

#### Команды администратора:
- `/users` - Список всех пользователей
- `/stats` - Статистика использования бота
- `/cleanup [дни]` - Очистка истории команд старше N дней

### Примеры использования

#### Просмотр статуса системы:
```
/status
```
Вернет подробную информацию:
```
💻 Статус системы

🖥️ Хост: DESKTOP-EXAMPLE
🔧 ОС: windows 10.0.19045
⏰ Время работы: 2 дн. 5 ч. 30 мин.
🔄 Процессов: 234

🧠 Процессор:
   • Модель: Intel(R) Core(TM) i7-10700K CPU @ 3.80GHz
   • Ядер: 8
   • Загрузка: 15.2%

🧮 Память:
   • Всего: 16.0 GB
   • Используется: 8.5 GB (53.1%)
   • Доступно: 7.5 GB
```

#### Просмотр истории команд:
```
/history 5
```

#### Файловый менеджер:
```
# Посмотреть доступные диски
/files

# Просмотреть содержимое диска
/files C:

# Просмотреть папку
/files C:\Users
```

#### Скриншот рабочего стола:
```
/screenshot
```
Скриншот будет автоматически отправлен в чат.

#### Получение статистики (только админы):
```
/stats
```

## Структура проекта

```
cupbot/
├── cmd/
│   └── cupbot/
│       └── main.go          # Точка входа приложения
├── config/
│   └── config.yaml          # Файл конфигурации
├── internal/
│   ├── auth/                # Аутентификация и авторизация
│   │   ├── middleware.go
│   │   └── models.go
│   ├── bot/                 # Логика Telegram бота
│   │   └── bot.go
│   ├── config/              # Конфигурация
│   │   └── config.go
│   ├── database/            # Работа с базой данных
│   │   └── database.go
│   └── system/              # Системная информация
│       └── service.go
├── go.mod
├── go.sum
├── main.go                  # Основной файл запуска
└── README.md
```

## База данных

Бот использует SQLite для хранения:
- Информации о пользователях
- Истории выполненных команд
- Сессий пользователей

База данных создается автоматически при первом запуске.

## Безопасность

### Авторизация
- Только пользователи из списка `admin_user_ids` и `allowed_users` могут использовать бота
- Все действия логируются в базу данных
- Разделение ролей: администраторы и обычные пользователи

### Логирование
- Все команды записываются в историю с временными метками
- Хранится информация об успешности выполнения команд
- Возможность очистки старых логов

## Развитие проекта

### Планируемые возможности
- ✅ Управление питанием (выключение/перезагрузка) - **COMPLETED**
- ✅ Удаленное выключение/перезагрузка - **COMPLETED** 
- ✅ Скриншоты рабочего стола - **COMPLETED**
- ✅ Уведомления о событиях системы - **COMPLETED**
- ✅ Управление файлами (просмотр, скачивание) - **COMPLETED**
- ✅ Интерфейс с кнопками - **COMPLETED**
- ✅ Меню администратора - **COMPLETED**
- 🔲 Управление процессами (список, завершение)
- 🔲 Управление службами Windows
- 🔲 Выполнение произвольных команд CMD/PowerShell
- 🔲 Web интерфейс для управления

### Как добавить новую команду
1. Добавьте обработчик в `internal/bot/bot.go`
2. Зарегистрируйте команду в методе `handleMessage`
3. Обновите справку в методе `handleHelp`

## Устранение неполадок

### Бот не отвечает
1. Проверьте правильность токена бота
2. Убедитесь, что ваш Telegram ID добавлен в список разрешенных пользователей
3. Проверьте логи на наличие ошибок

### Ошибки получения системной информации
- Убедитесь, что бот запущен от имени пользователя с достаточными правами
- Некоторые системные функции могут требовать повышенных привилегий

### Проблемы с базой данных
- Проверьте права записи в директорию с базой данных
- Убедитесь, что файл базы данных не используется другими процессами

## Лицензия

MIT License

## Вклад в проект

Мы приветствуем вклад в развитие проекта! Пожалуйста, создавайте Issues и Pull Requests.

## Поддержка

При возникновении проблем или вопросов:
1. Проверьте раздел "Устранение неполадок"
2. Создайте Issue с описанием проблемы
3. Приложите логи и описание окружения