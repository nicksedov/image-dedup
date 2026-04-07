# Image Dedup

Приложение для поиска и управления дубликатами изображений в медиатеке на локальном диске.

Проект разделен на два подпроекта:
- **backend/** -- Go REST API с бизнес-логикой (сканирование, БД, генерация скриптов)
- **frontend/** -- React + TypeScript веб-интерфейс (Vite, Tailwind CSS, shadcn/ui)

## Возможности

- Сканирование одной или нескольких директорий на наличие дубликатов изображений
- Определение дубликатов по совпадению размера файла и контрольной суммы (MD5)
- Веб-интерфейс с миниатюрами изображений (до 192px)
- Прямое удаление или перемещение файлов в корзину
- Генерация bash/PowerShell скриптов для перемещения файлов
- Пакетная дедупликация по шаблонам папок
- Асинхронное сканирование с отображением прогресса
- Кэширование метаданных в PostgreSQL для ускорения повторных сканирований

## Поддерживаемые форматы

JPG, JPEG, PNG, GIF, BMP, TIFF, TIF, WEBP

## Требования

- Go 1.23 или выше
- Node.js 18 или выше (для фронтенда)
- PostgreSQL 12 или выше

## Структура проекта

```
image-dedup/
├── backend/
│   ├── main.go           # Точка входа, CLI, запуск сервера
│   ├── config.go         # Конфигурация (переменные окружения)
│   ├── database.go       # Подключение к PostgreSQL
│   ├── models.go         # Модели данных (ImageFile, DuplicateGroup)
│   ├── scanner.go        # Сканирование и поиск дубликатов
│   ├── thumbnail.go      # Генерация миниатюр
│   ├── handlers.go       # HTTP обработчики (JSON API)
│   ├── dto.go            # DTO для запросов/ответов API
│   ├── middleware.go      # CORS middleware
│   ├── scan_manager.go   # Асинхронное сканирование
│   ├── .env.example      # Пример конфигурации
│   ├── go.mod
│   └── go.sum
│
├── frontend/
│   ├── src/
│   │   ├── App.tsx       # Главный компонент
│   │   ├── api/          # HTTP клиент и функции API
│   │   ├── types/        # TypeScript интерфейсы
│   │   ├── hooks/        # React хуки
│   │   ├── components/   # UI компоненты
│   │   └── lib/          # Утилиты
│   ├── .env.example      # Пример конфигурации
│   ├── package.json
│   └── vite.config.ts
│
├── .gitignore
└── README.md
```

## Конфигурация

### Backend (`backend/.env`)

| Переменная     | Описание                              | По умолчанию             |
|----------------|---------------------------------------|--------------------------|
| `DB_HOST`      | Хост PostgreSQL                       | `localhost`              |
| `DB_PORT`      | Порт PostgreSQL                       | `5432`                   |
| `DB_USER`      | Пользователь PostgreSQL               | `postgres`               |
| `DB_PASSWORD`  | Пароль PostgreSQL                     | `postgres`               |
| `DB_NAME`      | Имя базы данных                       | `image_dedup`            |
| `SERVER_HOST`  | Адрес привязки API сервера            | `0.0.0.0`               |
| `SERVER_PORT`  | Порт API сервера                      | `5170`                   |
| `CORS_ORIGINS` | Разрешенные источники (через запятую), или `*` для разрешения всех | `http://localhost:5173`  |

### Frontend (`frontend/.env`)

| Переменная     | Описание                            | По умолчанию |
|----------------|-------------------------------------|--------------|
| `VITE_API_URL` | URL бэкенд API                      | (пусто -- используется Vite прокси) |

В режиме разработки Vite проксирует запросы `/api/*` на бэкенд (`http://localhost:5170`).
Для продакшена укажите полный URL бэкенда в `VITE_API_URL`.

## Сборка и запуск

### 1. Создание базы данных PostgreSQL

```sql
CREATE DATABASE image_dedup;
```

### 2. Настройка окружения

```bash
# Backend
cp backend/.env.example backend/.env
# Отредактируйте backend/.env -- укажите параметры подключения к БД

# Frontend
cp frontend/.env.example frontend/.env
# Для разработки можно оставить VITE_API_URL пустым (используется прокси)
```

### 3. Сборка бэкенда

```bash
cd backend
go mod tidy
go build -o image-dedup.exe .    # Windows
go build -o image-dedup .        # Linux/macOS
```

### 4. Сборка фронтенда

```bash
cd frontend
npm install
npm run build    # Собирает в frontend/dist/
```

### 5. Запуск (режим разработки)

**Терминал 1 -- бэкенд:**

```bash
cd backend
go run .
```

Бэкенд запустится на `http://0.0.0.0:5170` по умолчанию.

**Терминал 2 -- фронтенд:**

```bash
cd frontend
npm run dev
```

Откройте в браузере: `http://localhost:5173`

### 6. Запуск (продакшен)

```bash
# Бэкенд
cd backend
./image-dedup      # Windows: image-dedup.exe

# Фронтенд -- статические файлы из frontend/dist/
# Раздайте через nginx, Caddy или любой другой веб-сервер
```

## Доступ с удалённой машины (тестирование в локальной сети)

Оба сервера (бэкенд и фронтенд) по умолчанию слушают на `0.0.0.0`, что делает их доступными с любой машины в локальной сети.

### 1. Узнайте IP-адрес сервера

На машине, где запущены серверы:

```bash
# Windows
ipconfig

# Linux / macOS
ip addr        # или: hostname -I
```

Запомните IPv4-адрес (например, `192.168.1.100`).

### 2. Настройте бэкенд

В `backend/.env` убедитесь, что заданы:

```env
SERVER_HOST=0.0.0.0
CORS_ORIGINS=*
```

- `SERVER_HOST=0.0.0.0` -- бэкенд слушает на всех сетевых интерфейсах
- `CORS_ORIGINS=*` -- разрешает запросы с любого источника (подходит для разработки в доверенной сети)

Для более строгой настройки перечислите конкретные источники:

```env
CORS_ORIGINS=http://192.168.1.100:5173,http://localhost:5173
```

### 3. Настройте фронтенд

В `frontend/.env` оставьте `VITE_API_URL` пустым:

```env
VITE_API_URL=
```

Это обеспечивает проксирование через Vite -- все `/api/*` запросы будут перенаправляться на бэкенд на той же машине. Vite dev-сервер уже слушает на `0.0.0.0` (`host: true` в `vite.config.ts`).

### 4. Запустите оба сервера

```bash
# Терминал 1 -- бэкенд
cd backend
go run .

# Терминал 2 -- фронтенд
cd frontend
npm run dev
```

### 5. Откройте с удалённой машины

На удалённой машине в браузере откройте:

```
http://192.168.1.100:5173
```

Замените `192.168.1.100` на реальный IP-адрес сервера.

### 6. Файрвол

Если удалённая машина не может подключиться, проверьте, что файрвол на серверной машине разрешает входящие соединения на порты **5170** (бэкенд) и **5173** (фронтенд).

Windows (PowerShell от администратора):

```powershell
netsh advfirewall firewall add rule name="Image Dedup Backend" dir=in action=allow protocol=TCP localport=5170
netsh advfirewall firewall add rule name="Image Dedup Frontend" dir=in action=allow protocol=TCP localport=5173
```

Linux:

```bash
sudo ufw allow 5170/tcp
sudo ufw allow 5173/tcp
```

## API

Все маршруты с префиксом `/api/`. Ответы в формате JSON.

| Метод | Маршрут               | Описание                                |
|-------|-----------------------|-----------------------------------------|
| GET   | `/api/duplicates`     | Группы дубликатов с пагинацией          |
| POST  | `/api/scan`           | Запуск асинхронного сканирования        |
| GET   | `/api/status`         | Статус текущего сканирования            |
| GET   | `/api/thumbnail`      | Миниатюра для файла                     |
| POST  | `/api/generate-script`| Генерация скрипта удаления              |
| POST  | `/api/delete-files`   | Прямое удаление файлов                  |
| GET   | `/api/folder-patterns`| Шаблоны папок для пакетной дедупликации |
| POST  | `/api/batch-delete`   | Пакетное удаление по правилам           |

## Лицензия

MIT
