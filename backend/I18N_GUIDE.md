# Руководство по интернационализации (i18n) Backend API

## Обзор

Все строковые сообщения, которые отображаются непосредственно в UI через API ответы, теперь передаются как **MessageKeys** - специальные литералы, по которым фронтенд находит переводы в i18n ресурсах.

## Структура i18n сообщений

### Файл с сообщениями
`internal/interfaces/i18n/messages.go`

### Типы сообщений
```go
type MessageKey string

const (
    // Общие сообщения
    Success        MessageKey = "success"
    Error          MessageKey = "error"
    ValidationError MessageKey = "validation_error"

    // Сообщения аутентификации
    MsgAuthInternalError         MessageKey = "auth.internal_error"
    MsgAuthInvalidCredentials    MessageKey = "auth.invalid_credentials"
    MsgAuthLogoutSuccess         MessageKey = "auth.logout_success"
    MsgAuthUnauthorized          MessageKey = "auth.unauthorized"
    MsgAuthInvalidRequestFormat  MessageKey = "auth.invalid_request_format"
    MsgAuthPasswordLength        MessageKey = "auth.password_length"
    MsgAuthInvalidCurrentPassword MessageKey = "auth.invalid_current_password"
    MsgAuthPasswordChangeFailed  MessageKey = "auth.password_change_failed"
    MsgAuthUserCreated           MessageKey = "auth.user_created"
    MsgAuthUserNotFound          MessageKey = "auth.user_not_found"
    MsgAuthUserUpdated           MessageKey = "auth.user_updated"
    MsgAuthUserDeleted           MessageKey = "auth.user_deleted"
    
    // Сообщения сканирования
    MsgScanStarted         MessageKey = "scan.started"
    MsgScanFailed          MessageKey = "scan.failed"
    MsgScanNoFilesSelected MessageKey = "scan.no_files_selected"
    
    // Сообщения папок
    MsgFolderAdded         MessageKey = "folder.added"
    MsgFolderNotFound      MessageKey = "folder.not_found"
    MsgFolderConflictTrash MessageKey = "folder.conflict_trash"
    
    // Сообщения изображений
    MsgImagePathRequired MessageKey = "image.path_required"
    MsgImageNotFound     MessageKey = "image.not_found"
    
    // Сообщения middleware
    MsgMiddlewareUnauthorized MessageKey = "middleware.unauthorized"
    MsgMiddlewareForbidden    MessageKey = "middleware.forbidden"
    MsgMiddlewareCSRFFailed   MessageKey = "middleware.csrf_failed"
    
    // Сообщения корзины
    MsgTrashNotConfigured MessageKey = "trash.not_configured"
    MsgTrashNotExists     MessageKey = "trash.not_exists"
)
```

## Использование в API handlers

### Примеры ответов

**Раньше (hardcoded строки):**
```go
c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный логин или пароль"})
c.JSON(http.StatusOK, gin.H{"message": "Выход выполнен"})
```

**Теперь (MessageKeys):**
```go
c.JSON(http.StatusBadRequest, i18n.ErrorResponse(i18n.MsgAuthInvalidCredentials))
c.JSON(http.StatusOK, gin.H{"message": i18n.MsgAuthLogoutSuccess})
```

### Вспомогательные функции

```go
// Создание ответа об ошибке
i18n.ErrorResponse(msg MessageKey) map[string]interface{}

// Создание успешного ответа
i18n.SuccessResponse(msg MessageKey, data ...interface{}) (map[string]interface{}, MessageKey)

// Создание ошибки валидации
i18n.ValidationError(msg MessageKey) map[string]interface{}
```

## Формат JSON ответов

### Успешные ответы
```json
{
  "message": "auth.logout_success",
  "user": {
    "id": 1,
    "login": "admin"
  }
}
```

### Ошибки
```json
{
  "error": "auth.invalid_credentials"
}
```

### Ошибки валидации
```json
{
  "error": "validation_error",
  "type": "validation"
}
```

## Frontend i18n ресурсы

Фронтенд должен иметь файлы локализации с соответствующими ключами:

### ru.json
```json
{
  "auth": {
    "invalid_credentials": "Неверный логин или пароль",
    "logout_success": "Выход выполнен",
    "unauthorized": "Требуется авторизация"
  },
  "scan": {
    "started": "Scan started",
    "no_files_selected": "No files selected"
  },
  "folder": {
    "added": "Folder added to gallery",
    "not_found": "Folder not found"
  }
}
```

### en.json
```json
{
  "auth": {
    "invalid_credentials": "Invalid login or password",
    "logout_success": "Logged out successfully",
    "unauthorized": "Authorization required"
  },
  "scan": {
    "started": "Scan started",
    "no_files_selected": "No files selected"
  },
  "folder": {
    "added": "Folder added to gallery",
    "not_found": "Folder not found"
  }
}
```

## DTO Responses с Message

Некоторые DTO responses содержат поле `Message string` для обратной совместимости:

```go
type ScanResponse struct {
    Message string `json:"message"`  // i18n key
}
```

## Добавление новых сообщений

1. Добавьте константу в `internal/interfaces/i18n/messages.go`
2. Используйте её в handlers/middleware
3. Обновите i18n ресурсы фронтенда

## Миграция старого кода

Для старых ответов с hardcoded строками:
- Замените `"Неверный логин или пароль"` на `i18n.MsgAuthInvalidCredentials`
- Замените `"Выход выполнен"` на `i18n.MsgAuthLogoutSuccess`
- Используйте `i18n.ErrorResponse()` вместо `gin.H{"error": ...}`

## Преимущества

1. **Интернационализация** - простое добавление новых языков
2. **Поддержка** - централизованное управление сообщениями
3. **Типобезопасность** - константы вместо строк
4. **Проверка** - linter может проверить использование ключей
