# Image Dedup

Приложение для поиска и управления дубликатами изображений в медиатеке на локальном диске.

## Возможности

- Сканирование одной или нескольких директорий на наличие дубликатов изображений
- Определение дубликатов по совпадению размера файла и контрольной суммы (MD5)
- Веб-интерфейс с миниатюрами изображений (до 128px)
- Генерация bash-скрипта для перемещения выбранных файлов в папку для удаления
- Кэширование метаданных в PostgreSQL для ускорения повторных сканирований

## Поддерживаемые форматы

JPG, JPEG, PNG, GIF, BMP, TIFF, TIF, WEBP

## Требования

- Go 1.21 или выше
- PostgreSQL 12 или выше

## Установка

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd image-dedup
```

### 2. Создание базы данных PostgreSQL

```sql
CREATE DATABASE image_dedup;
```

### 3. Сборка приложения

```bash
go mod tidy
go build -o image-dedup .
```

На Windows:
```bash
go build -o image-dedup.exe .
```

## Настройка

Приложение использует переменные окружения для подключения к базе данных:

| Переменная    | Описание                | По умолчанию  |
|---------------|-------------------------|---------------|
| `DB_HOST`     | Хост PostgreSQL         | `localhost`   |
| `DB_PORT`     | Порт PostgreSQL         | `5432`        |
| `DB_USER`     | Пользователь            | `postgres`    |
| `DB_PASSWORD` | Пароль                  | `postgres`    |
| `DB_NAME`     | Имя базы данных         | `image_dedup` |

## Запуск

### Базовый запуск

```bash
./image-dedup /path/to/photos
```

### Сканирование нескольких директорий

```bash
./image-dedup /path/to/photos /path/to/backup /path/to/downloads
```

### Указание порта

```bash
./image-dedup --port 3000 /path/to/photos
```

### Пример с переменными окружения

Linux/macOS:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=myuser
export DB_PASSWORD=mypassword
export DB_NAME=image_dedup

./image-dedup /home/user/Pictures
```

Windows (PowerShell):
```powershell
$env:DB_HOST="localhost"
$env:DB_PORT="5432"
$env:DB_USER="myuser"
$env:DB_PASSWORD="mypassword"
$env:DB_NAME="image_dedup"

.\image-dedup.exe C:\Users\user\Pictures
```

## Использование веб-интерфейса

1. После запуска откройте в браузере: `http://localhost:8080`
2. Просмотрите найденные группы дубликатов
3. Отметьте чекбоксами файлы, которые хотите удалить
4. Нажмите "Generate Removal Script"
5. Укажите путь для сохранения скрипта и папку для перемещения файлов
6. Просмотрите сгенерированный скрипт перед выполнением

### Кнопки управления

- **Rescan Directories** - повторное сканирование директорий
- **Reset Selection** - сброс всех отмеченных чекбоксов
- **Generate Removal Script** - генерация bash-скрипта для перемещения файлов

## Генерируемый скрипт

Скрипт перемещает выбранные файлы в указанную папку (trash):

```bash
#!/bin/bash
TRASH_DIR="/path/to/trash"
mkdir -p "$TRASH_DIR"
mv "/path/to/duplicate1.jpg" "$TRASH_DIR/duplicate1.jpg"
mv "/path/to/duplicate2.jpg" "$TRASH_DIR/duplicate2.jpg"
```

Перед выполнением скрипта рекомендуется его просмотреть.

## Структура проекта

```
image-dedup/
├── main.go          # Точка входа, обработка аргументов CLI
├── database.go      # Подключение к PostgreSQL
├── scanner.go       # Сканирование и поиск дубликатов
├── thumbnail.go     # Генерация миниатюр
├── handlers.go      # HTTP обработчики
├── templates/
│   └── index.html   # HTML шаблон веб-интерфейса
├── go.mod
└── go.sum
```

## Лицензия

MIT
