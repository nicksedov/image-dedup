package thumbnail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/deepteams/webp"
	"github.com/disintegration/imaging"
)

// Config конфигурация ThumbnailService
type Config struct {
	CacheDir      string
	MaxSize       int // Максимальный размер миниатюры в пикселях
	Quality       int // Качество сжатия (0-100)
	Enabled       bool
	Format        string // "webp" или "jpeg"
	CacheTTL      time.Duration
	PreloadOnScan bool
}

// Service управляет кэшированием миниатюр
type Service struct {
	cfg         *Config
	storage     *ThumbnailCacheStorage
	mu          sync.RWMutex
	stats       ThumbnailStats
	initialized bool
}

// ThumbnailStats статистика кэша миниатюр
type ThumbnailStats struct {
	TotalSize   int64  `json:"totalSize"`
	TotalFiles  int    `json:"totalFiles"`
	CacheDir    string `json:"cacheDir"`
	Enabled     bool   `json:"enabled"`
	Initialized bool   `json:"initialized"`
}

// DefaultCacheDir возвращает путь по умолчанию для кэша миниатюр (Linux XDG Compatible)
func DefaultCacheDir() string {
	home := os.Getenv("HOME")
	// В Docker/Alpine HOME может быть "/" или пустым - в этих случаях используем /tmp
	if home == "" || home == "/" {
		return filepath.Join(os.TempDir(), "image-tool", "thumbnails")
	}
	return filepath.Join(home, ".cache", "image-tool", "thumbnails")
}

// NewService создает новый ThumbnailService
func NewService(cfg *Config) (*Service, error) {
	if cfg == nil {
		cfg = &Config{
			CacheDir:      DefaultCacheDir(),
			MaxSize:       ThumbnailMaxSize,
			Quality:       ThumbnailQuality,
			Enabled:       true,
			Format:        ThumbnailFormat,
			CacheTTL:      30 * 24 * time.Hour, // 30 дней
			PreloadOnScan: true,
		}
	}

	if cfg.CacheDir == "" {
		cfg.CacheDir = DefaultCacheDir()
	}

	storage, err := NewThumbnailCacheStorage(cfg.CacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create thumbnail storage: %w", err)
	}

	s := &Service{
		cfg:         cfg,
		storage:     storage,
		initialized: true,
	}

	// Получаем статистику при инициализации
	s.updateStats()

	return s, nil
}

// Start инициализирует сервис (вызывается при старте приложения)
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled {
		return nil
	}

	// Убедимся, что структура кэша создана
	if err := s.storage.initStructure(); err != nil {
		return fmt.Errorf("failed to initialize thumbnail cache: %w", err)
	}

	s.initialized = true
	s.updateStats()
	return nil
}

// Stop останавливает сервис (вызывается при остановке приложения)
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initialized = false
}

// IsEnabled проверяет, включен ли кэш
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Enabled && s.initialized
}

// GetThumbnailPath возвращает путь к миниатюре для указанного файла
func (s *Service) GetThumbnailPath(filePath string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.cfg.Enabled || !s.initialized {
		return ""
	}

	path := s.storage.Get(filePath)
	return path
}

// HasThumbnail проверяет наличие миниатюры в кэше
func (s *Service) HasThumbnail(filePath string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.cfg.Enabled || !s.initialized {
		return false
	}

	return s.storage.Exists(filePath)
}

// GetOrGenerate получает миниатюру из кэша или генерирует её
func (s *Service) GetOrGenerate(filePath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled || !s.initialized {
		return "", ErrThumbnailCacheDisabled
	}

	// Проверяем кэш
	cachedPath := s.storage.Get(filePath)
	if cachedPath != "" {
		// Прочитать данные из файла
		data, err := os.ReadFile(cachedPath)
		if err != nil {
			// Если файл утерян, удаляем запись из кэша
			s.storage.Delete(filePath)
			return "", &ErrCacheReadFailed{Path: filePath, Err: err}
		}

		s.stats.TotalFiles++
		return s.encodeToDataURL(data), nil
	}

	// Генерируем новую миниатюру
	encodedData, err := s.generateThumbnail(filePath)
	if err != nil {
		return "", err
	}

	// Сохраняем в кэш
	if err := s.storage.Set(filePath, encodedData); err != nil {
		return "", err
	}

	s.stats.TotalFiles++
	s.stats.TotalSize += int64(len(encodedData))

	return s.encodeToDataURL(encodedData), nil
}

// encodeToDataURL кодирует бинарные данные в data URL с правильным MIME типом
func (s *Service) encodeToDataURL(data []byte) string {
	mimeType := "image/webp"
	switch s.cfg.Format {
	case "jpeg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// GenerateThumbnail генерирует миниатюру для указанного файла (без сохранения в кэш)
func (s *Service) GenerateThumbnail(filePath string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.generateThumbnail(filePath)
}

// generateThumbnail внутренняя функция генерации миниатюры
func (s *Service) generateThumbnail(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var newWidth, newHeight int
	if width >= height {
		newWidth = s.cfg.MaxSize
		newHeight = 0
	} else {
		newWidth = 0
		newHeight = s.cfg.MaxSize
	}

	thumbnail := imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)

	var buf bytes.Buffer
	switch s.cfg.Format {
	case "webp":
		if err := webp.Encode(&buf, thumbnail, &webp.Options{Quality: float32(s.cfg.Quality)}); err != nil {
			buf.Reset()
			if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: s.cfg.Quality}); err != nil {
				return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
			}
		}
	case "jpeg":
		if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: s.cfg.Quality}); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
	case "png":
		if err := png.Encode(&buf, thumbnail); err != nil {
			return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// Invalidate invalидирует миниатюру для указанного файла (удаляет из кэша)
func (s *Service) Invalidate(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled || !s.initialized {
		return nil
	}

	return s.storage.Delete(filePath)
}

// saveThumbnailToCache внутренний метод для сохранения миниатюры (для интеграции со сканером)
func (s *Service) saveThumbnailToCache(filePath string, thumbnailData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled || !s.initialized {
		return ErrThumbnailCacheDisabled
	}

	return s.storage.Set(filePath, thumbnailData)
}

// InvalidateAll invalидирует все миниатюры
func (s *Service) InvalidateAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled {
		return nil
	}

	if err := s.storage.ClearAll(); err != nil {
		return err
	}

	s.stats = ThumbnailStats{}
	return nil
}

// Warmup предварительно генерирует миниатюры для всех файлов
func (s *Service) Warmup(imagePaths []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled || !s.initialized {
		return nil
	}

	for _, path := range imagePaths {
		if !s.storage.Exists(path) {
			encodedData, err := s.generateThumbnail(path)
			if err != nil {
				continue // Пропускаем файлы с ошибками
			}

			if err := s.storage.Set(path, encodedData); err != nil {
				continue
			}

			s.stats.TotalFiles++
			s.stats.TotalSize += int64(len(encodedData))
		}
	}

	return nil
}

// GenerateThumbnailPath возвращает относительный путь к миниатюре для указанного файла (относительно корня кэша)
func (s *Service) GenerateThumbnailPath(filePath string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.cfg.Enabled || !s.initialized {
		return ""
	}

	// Возвращаем относительный путь
	return s.storage.GetPathRelative(filePath)
}

// UpdateStats обновляет статистику кэша
func (s *Service) updateStats() {
	if !s.initialized {
		return
	}

	count, size, err := s.storage.Stats()
	if err != nil {
		return
	}

	s.stats = ThumbnailStats{
		TotalFiles:  count,
		TotalSize:   size,
		CacheDir:    s.cfg.CacheDir,
		Enabled:     s.cfg.Enabled,
		Initialized: true,
	}
}

// Stats возвращает текущую статистику кэша
func (s *Service) Stats() ThumbnailStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		log.Printf("Stats: service not initialized")
		return ThumbnailStats{}
	}

	// Always refresh from disk to ensure accuracy
	s.updateStats()

	log.Printf("Stats: returning %+v", s.stats)
	return s.stats
}

// UpdateCachePath обновляет путь кэша с физическим перемещением файлов
func (s *Service) UpdateCachePath(newPath string) error {
	if newPath == "" {
		return &ErrInvalidCachePath{Path: newPath}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldPath := s.cfg.CacheDir
	log.Printf("UpdateCachePath: old=%s, new=%s", oldPath, newPath)

	// Если путь не изменился, просто обновляем статистику
	if oldPath == newPath {
		log.Printf("UpdateCachePath: paths are the same, updating stats only")
		s.updateStats()
		log.Printf("UpdateCachePath: stats after update: %+v", s.stats)
		return nil
	}

	// Перемещаем файлы из старого хранилища в новое
	if err := s.moveCacheTo(newPath); err != nil {
		log.Printf("UpdateCachePath: moveCacheTo failed: %v", err)
		return err
	}

	// Создаем новое хранилище и заменяем старое
	newStorage, err := NewThumbnailCacheStorage(newPath)
	if err != nil {
		log.Printf("UpdateCachePath: NewThumbnailCacheStorage failed: %v", err)
		return &ErrCacheInitFailed{Path: newPath, Err: err}
	}

	s.storage = newStorage
	s.cfg.CacheDir = newPath
	s.cfg.Enabled = true
	s.initialized = true

	s.updateStats()
	log.Printf("UpdateCachePath: success, stats: %+v", s.stats)
	return nil
}

// moveCacheTo перемещает кэш в новое место
func (s *Service) moveCacheTo(newPath string) error {
	oldPath := s.cfg.CacheDir

	// Если старый путь пустой, ничего не перемещаем
	if oldPath == "" {
		return nil
	}

	// Проверяем, существует ли старая директория
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		// Старая директория не существует, просто создаем новую
		if err := os.MkdirAll(newPath, 0755); err != nil {
			return &ErrCacheInitFailed{Path: newPath, Err: err}
		}
		return nil
	}

	// Получаем список файлов в текущем кэше
	files, err := s.storage.ListFiles()
	if err != nil {
		// Если не удалось получить список файлов, пробуем продолжить
		// Возможно, кэш пустой или структура не создана
		files = []string{}
	}

	// Перемещаем каждый файл
	for _, srcPath := range files {
		// Считываем содержимое
		data, err := os.ReadFile(srcPath)
		if err != nil {
			// Пропускаем файлы, которые не удалось прочитать
			continue
		}

		// Записываем в новое место
		dstPath := filepath.Join(newPath, filepath.Base(filepath.Dir(srcPath)), filepath.Base(srcPath))
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			continue
		}

		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			continue
		}

		// Удаляем старый файл после успешного копирования
		os.Remove(srcPath)
	}

	// Удаляем старую папку кэша (она должна быть пустой после перемещения файлов)
	s.cleanupEmptyDirs(oldPath)

	return nil
}

// cleanupEmptyDirs удаляет пустые подпапки
func (s *Service) cleanupEmptyDirs(dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Пытаемся удалить пустую папку
			os.Remove(path)
		}

		return nil
	})
}

// Enable включает кэш
func (s *Service) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Enabled = true
	if s.storage != nil {
		s.storage.Enable()
	}
}

// Disable выключает кэш
func (s *Service) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cfg.Enabled = false
	if s.storage != nil {
		s.storage.Disable()
	}
}
