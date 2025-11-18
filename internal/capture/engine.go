package capture

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sync"
	"time"

	"worktracker/internal/config"
	"worktracker/internal/storage"
	"worktracker/pkg/logger"
	"worktracker/pkg/models"
	"worktracker/pkg/utils"

	"github.com/kbinani/screenshot"
)

// Engine 截屏引擎
type Engine struct {
	configMgr *config.Manager
	storage   *storage.Manager
	ticker    *time.Ticker
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	mu        sync.RWMutex
	lastCapture time.Time
}

// NewEngine 创建截屏引擎
func NewEngine(configMgr *config.Manager, storageMgr *storage.Manager) *Engine {
	return &Engine{
		configMgr: configMgr,
		storage:   storageMgr,
	}
}

// Start 启动截屏引擎
func (e *Engine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		logger.Warn("截屏引擎已在运行中")
		return fmt.Errorf("capture engine already running")
	}

	cfg := e.configMgr.GetCapture()
	if !cfg.Enabled {
		logger.Warn("截屏功能未启用")
		return fmt.Errorf("capture is disabled in config")
	}

	e.ctx, e.cancel = context.WithCancel(context.Background())
	e.ticker = time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	e.running = true

	go e.captureLoop()

	logger.Info("截屏引擎已启动,间隔: %d秒", cfg.Interval)
	return nil
}

// Stop 停止截屏引擎
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("capture engine not running")
	}

	e.cancel()
	e.ticker.Stop()
	e.running = false

	logger.Info("截屏引擎已停止")
	return nil
}

// IsRunning 检查是否运行中
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// GetLastCapture 获取最后一次截图时间
func (e *Engine) GetLastCapture() time.Time {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastCapture
}

// captureLoop 截屏循环
func (e *Engine) captureLoop() {
	logger.Info("截屏循环已启动")
	for {
		select {
		case <-e.ctx.Done():
			logger.Info("截屏循环已停止")
			return
		case <-e.ticker.C:
			if e.shouldCapture() {
				if err := e.captureAll(); err != nil {
					logger.Error("截屏失败: %v", err)
				}
			}
		}
	}
}

// shouldCapture 检查是否应该截屏
func (e *Engine) shouldCapture() bool {
	schedule := e.configMgr.GetSchedule()

	if !schedule.Enabled {
		return true
	}

	// 检查星期几
	now := time.Now()
	if !utils.IsDayInList(now.Weekday(), schedule.WorkDays) {
		return false
	}

	// 检查时间范围
	inRange, err := utils.TimeInRange(schedule.StartTime, schedule.EndTime)
	if err != nil {
		logger.Error("时间范围检查错误: %v", err)
		return false
	}

	return inRange
}

// captureAll 截取所有配置的屏幕
func (e *Engine) captureAll() error {
	cfg := e.configMgr.GetCapture()

	for _, screenIndex := range cfg.SelectedScreens {
		if err := e.captureScreen(screenIndex); err != nil {
			return fmt.Errorf("failed to capture screen %d: %w", screenIndex, err)
		}
	}

	e.mu.Lock()
	e.lastCapture = time.Now()
	e.mu.Unlock()

	logger.Debug("截屏完成,共 %d 个屏幕", len(cfg.SelectedScreens))
	return nil
}

// captureScreen 截取指定屏幕
func (e *Engine) captureScreen(screenIndex int) error {
	// 获取屏幕数量
	n := screenshot.NumActiveDisplays()
	if screenIndex < 0 || screenIndex >= n {
		return fmt.Errorf("invalid screen index: %d (total: %d)", screenIndex, n)
	}

	// 获取屏幕边界
	bounds := screenshot.GetDisplayBounds(screenIndex)

	// 截取屏幕
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	// 保存截图
	return e.saveScreenshot(img, screenIndex, bounds)
}

// saveScreenshot 保存截图
func (e *Engine) saveScreenshot(img *image.RGBA, screenIndex int, bounds image.Rectangle) error {
	cfg := e.configMgr.GetCapture()
	storageCfg := e.configMgr.GetStorage()

	// 生成文件名
	now := time.Now()
	filename := fmt.Sprintf("screenshot_%d_%s.jpg",
		screenIndex,
		now.Format("20060102_150405"),
	)

	// 确保目录存在 - 使用配置的截图目录
	screenshotsDir := storageCfg.ScreenshotsDir
	if screenshotsDir == "" {
		screenshotsDir = filepath.Join(storageCfg.DataDir, "screenshots")
	}
	dateDir := filepath.Join(screenshotsDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dateDir, filename)

	// 压缩为 JPEG
	var buf bytes.Buffer
	opt := jpeg.Options{Quality: cfg.Quality}
	if err := jpeg.Encode(&buf, img, &opt); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 保存到数据库
	ss := &models.Screenshot{
		Timestamp:   now,
		ScreenIndex: screenIndex,
		FilePath:    filePath,
		FileSize:    int64(buf.Len()),
		Resolution:  fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()),
		Analyzed:    false,
		CreatedAt:   now,
	}

	if err := e.storage.SaveScreenshot(ss); err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	logger.Debug("截图已保存: %s (%.2f KB)", filePath, float64(buf.Len())/1024)
	return nil
}

// CaptureNow 立即截取一次
func (e *Engine) CaptureNow(screenIndex int) (*models.Screenshot, error) {
	n := screenshot.NumActiveDisplays()
	if screenIndex < 0 || screenIndex >= n {
		return nil, fmt.Errorf("invalid screen index: %d (total: %d)", screenIndex, n)
	}

	bounds := screenshot.GetDisplayBounds(screenIndex)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	if err := e.saveScreenshot(img, screenIndex, bounds); err != nil {
		return nil, err
	}

	// 返回最新的截图记录
	screenshots, err := e.storage.GetRecentScreenshots(1)
	if err != nil || len(screenshots) == 0 {
		return nil, fmt.Errorf("failed to retrieve saved screenshot")
	}

	return screenshots[0], nil
}

// GetScreens 获取所有屏幕信息
func GetScreens() []models.ScreenInfo {
	n := screenshot.NumActiveDisplays()
	screens := make([]models.ScreenInfo, n)

	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		screens[i] = models.ScreenInfo{
			Index:     i,
			Name:      fmt.Sprintf("Display %d", i+1),
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			IsPrimary: i == 0, // 假设第一个是主屏幕
		}
	}

	return screens
}
