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

	"WorkTrackerAI/internal/config"
	"WorkTrackerAI/internal/storage"
	"WorkTrackerAI/pkg/logger"
	"WorkTrackerAI/pkg/models"
	"WorkTrackerAI/pkg/screenstate"
	"WorkTrackerAI/pkg/utils"

	"github.com/kbinani/screenshot"
	"github.com/nfnt/resize"
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
	// 检测屏幕状态：如果屏幕被锁定或屏保运行中，跳过截屏
	active, screensaverRunning, screenLocked := screenstate.GetScreenStateInfo()
	
	// 记录详细的屏幕状态信息
	logger.Info("屏幕状态检测 - 活跃:%v, 屏保运行:%v, 屏幕锁定:%v", active, screensaverRunning, screenLocked)
	
	if !active {
		if screensaverRunning {
			logger.Info("⏸️  屏保正在运行，跳过本次截屏")
		} else if screenLocked {
			logger.Info("🔒 屏幕已锁定，跳过本次截屏")
		} else {
			logger.Info("⏸️  屏幕未激活，跳过本次截屏")
		}
		return nil
	}
	
	logger.Debug("✅ 屏幕状态正常，开始截屏")

	cfg := e.configMgr.GetCapture()

	// 如果启用多屏幕拼接，则拼接所有屏幕
	if cfg.MergeScreens {
		n := screenshot.NumActiveDisplays()
		if n > 1 {
			return e.captureMergedScreens()
		}
		// 只有一个屏幕时，正常截取
		if err := e.captureScreen(0); err != nil {
			return fmt.Errorf("failed to capture screen 0: %w", err)
		}
	} else {
		// 不拼接时，按配置截取选定的屏幕
		for _, screenIndex := range cfg.SelectedScreens {
			if err := e.captureScreen(screenIndex); err != nil {
				return fmt.Errorf("failed to capture screen %d: %w", screenIndex, err)
			}
		}
	}

	e.mu.Lock()
	e.lastCapture = time.Now()
	e.mu.Unlock()

	logger.Debug("截屏完成")
	return nil
}

// captureMergedScreens 截取并拼接所有屏幕
func (e *Engine) captureMergedScreens() error {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return fmt.Errorf("no active displays found")
	}

	// 1. 获取所有屏幕的边界和截图
	type screenCapture struct {
		bounds image.Rectangle
		img    *image.RGBA
	}
	captures := make([]screenCapture, n)

	// 计算整体画布的边界
	var minX, minY, maxX, maxY int
	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			return fmt.Errorf("failed to capture screen %d: %w", i, err)
		}

		captures[i] = screenCapture{bounds: bounds, img: img}

		// 更新整体边界
		if i == 0 {
			minX, minY = bounds.Min.X, bounds.Min.Y
			maxX, maxY = bounds.Max.X, bounds.Max.Y
		} else {
			if bounds.Min.X < minX {
				minX = bounds.Min.X
			}
			if bounds.Min.Y < minY {
				minY = bounds.Min.Y
			}
			if bounds.Max.X > maxX {
				maxX = bounds.Max.X
			}
			if bounds.Max.Y > maxY {
				maxY = bounds.Max.Y
			}
		}
	}

	// 2. 创建拼接画布
	canvasWidth := maxX - minX
	canvasHeight := maxY - minY
	merged := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))

	// 3. 将各屏幕图像绘制到画布上
	for i, cap := range captures {
		// 计算在画布上的位置
		offsetX := cap.bounds.Min.X - minX
		offsetY := cap.bounds.Min.Y - minY

		// 绘制图像
		for y := 0; y < cap.img.Bounds().Dy(); y++ {
			for x := 0; x < cap.img.Bounds().Dx(); x++ {
				merged.Set(offsetX+x, offsetY+y, cap.img.At(x, y))
			}
		}
		logger.Debug("已拼接屏幕 %d (位置: %d, %d)", i, offsetX, offsetY)
	}

	// 4. 保存拼接后的图像
	mergedBounds := image.Rect(minX, minY, maxX, maxY)
	if err := e.saveScreenshot(merged, -1, mergedBounds); err != nil {
		return fmt.Errorf("failed to save merged screenshot: %w", err)
	}

	logger.Info("多屏幕拼接完成：%d 个屏幕，分辨率 %dx%d", n, canvasWidth, canvasHeight)
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

// saveScreenshot 保存截图（支持智能压缩和缩放）
func (e *Engine) saveScreenshot(img *image.RGBA, screenIndex int, bounds image.Rectangle) error {
	cfg := e.configMgr.GetCapture()
	storageCfg := e.configMgr.GetStorage()

	// 1. 智能缩放（如果启用）
	processedImg := image.Image(img)
	finalWidth := bounds.Dx()
	finalHeight := bounds.Dy()

	if cfg.EnableResize && (cfg.MaxWidth > 0 || cfg.MaxHeight > 0) {
		width := bounds.Dx()
		height := bounds.Dy()

		// 计算是否需要缩放
		needResize := false
		scaleWidth, scaleHeight := width, height

		if cfg.MaxWidth > 0 && width > cfg.MaxWidth {
			scaleWidth = cfg.MaxWidth
			scaleHeight = height * cfg.MaxWidth / width
			needResize = true
		}

		if cfg.MaxHeight > 0 && scaleHeight > cfg.MaxHeight {
			scaleHeight = cfg.MaxHeight
			scaleWidth = width * cfg.MaxHeight / height
			needResize = true
		}

		if needResize {
			// 使用 Lanczos3 算法进行高质量缩放
			processedImg = resize.Resize(uint(scaleWidth), uint(scaleHeight), img, resize.Lanczos3)
			finalWidth = scaleWidth
			finalHeight = scaleHeight
			logger.Debug("图像缩放: %dx%d -> %dx%d", width, height, scaleWidth, scaleHeight)
		}
	}

	// 2. 确定文件扩展名（暂时只支持 JPEG）
	fileExt := ".jpg"

	// 3. 生成文件名
	now := time.Now()
	var filename string
	if screenIndex == -1 {
		filename = fmt.Sprintf("screenshot_merged_%s%s", now.Format("20060102_150405"), fileExt)
	} else {
		filename = fmt.Sprintf("screenshot_%d_%s%s", screenIndex, now.Format("20060102_150405"), fileExt)
	}

	// 4. 确保目录存在
	screenshotsDir := storageCfg.ScreenshotsDir
	if screenshotsDir == "" {
		screenshotsDir = filepath.Join(storageCfg.DataDir, "screenshots")
	}
	dateDir := filepath.Join(screenshotsDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(dateDir, filename)

	// 5. JPEG 压缩编码
	var buf bytes.Buffer
	encodeErr := jpeg.Encode(&buf, processedImg, &jpeg.Options{
		Quality: cfg.Quality,
	})

	if encodeErr != nil {
		return fmt.Errorf("failed to encode JPEG: %w", encodeErr)
	}

	// 6. 写入文件
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// 7. 保存到数据库
	ss := &models.Screenshot{
		Timestamp:   now,
		ScreenIndex: screenIndex,
		FilePath:    filePath,
		FileSize:    int64(buf.Len()),
		Resolution:  fmt.Sprintf("%dx%d", finalWidth, finalHeight),
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
		// 主屏幕的左上角坐标为 (0, 0)
		isPrimary := bounds.Min.X == 0 && bounds.Min.Y == 0
		screens[i] = models.ScreenInfo{
			Index:     i,
			Name:      fmt.Sprintf("Display %d", i+1),
			Width:     bounds.Dx(),
			Height:    bounds.Dy(),
			IsPrimary: isPrimary,
		}
	}

	return screens
}
