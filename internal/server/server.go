package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"WorkTrackerAI/internal/ai"
	"WorkTrackerAI/internal/capture"
	"WorkTrackerAI/internal/config"
	"WorkTrackerAI/internal/storage"
	"WorkTrackerAI/pkg/models"

	"github.com/gin-gonic/gin"
)

// Server Web 服务器
type Server struct {
	router      *gin.Engine
	configMgr   *config.Manager
	storageMgr  *storage.Manager
	captureEng  *capture.Engine
	aiAnalyzer  *ai.Analyzer
	addr        string
	version     string
	httpServer  *http.Server
}

// NewServer 创建 Web 服务器
func NewServer(
	configMgr *config.Manager,
	storageMgr *storage.Manager,
	captureEng *capture.Engine,
	aiAnalyzer *ai.Analyzer,
	version string,
) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	serverCfg := configMgr.GetServer()
	addr := fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port)

	s := &Server{
		router:     router,
		configMgr:  configMgr,
		storageMgr: storageMgr,
		captureEng: captureEng,
		aiAnalyzer: aiAnalyzer,
		addr:       addr,
		version:    version,
	}

	s.setupRoutes()
	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 静态文件
	s.router.Static("/static", "./web/static")
	s.router.Static("/asserts", "./asserts")
	s.router.LoadHTMLGlob("./web/templates/*")

	// 首页
	s.router.GET("/", s.handleIndex)

	// API 路由组
	api := s.router.Group("/api")
	{
		// 系统信息
		api.GET("/version", s.handleGetVersion)

		// 配置管理
		api.GET("/config", s.handleGetConfig)
		api.PUT("/config", s.handleUpdateConfig)
		api.GET("/screens", s.handleGetScreens)

		// AI 相关
		api.POST("/ai/test-connection", s.handleTestAIConnection)

		// 截图管理
		api.GET("/screenshots", s.handleGetScreenshots)
		api.GET("/screenshots/:id", s.handleGetScreenshot)
		api.DELETE("/screenshots/:id", s.handleDeleteScreenshot)
		api.POST("/screenshots/capture", s.handleCaptureNow)

		// 工作总结
		api.GET("/summaries", s.handleGetSummaries)
		api.GET("/summaries/:date", s.handleGetSummariesByDate)
		api.POST("/summaries/analyze", s.handleAnalyzeNow)

		// 统计数据
		api.GET("/stats/today", s.handleGetTodayStats)
		api.GET("/stats/storage", s.handleGetStorageStats)
		api.POST("/stats/open-folder", s.handleOpenStorageFolder)

		// 服务控制
		api.POST("/service/start", s.handleStartService)
		api.POST("/service/stop", s.handleStopService)
		api.GET("/service/status", s.handleGetStatus)
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    s.addr,
		Handler: s.router,
	}

	fmt.Printf("🌐 Web服务器启动: http://%s\n", s.addr)

	// 启动服务器（会阻塞）
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭服务器
func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	fmt.Println("🛑 正在关闭 Web 服务器...")

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 优雅关闭
	if err := s.httpServer.Shutdown(ctx); err != nil {
		fmt.Printf("⚠️ 服务器关闭错误: %v\n", err)
		return err
	}

	fmt.Println("✅ Web 服务器已关闭")
	return nil
}

// ===== 处理函数 =====

// handleIndex 首页
func (s *Server) handleIndex(c *gin.Context) {
	// 禁用缓存，确保总是获取最新版本
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Version": s.version,
	})
}

// handleGetVersion 获取版本信息
func (s *Server) handleGetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": s.version,
		"name":    "WorkTracker AI",
	})
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	cfg := s.configMgr.Get()
	c.JSON(http.StatusOK, cfg)
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
	var newConfig models.AppConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.configMgr.Update(func(cfg *models.AppConfig) {
		*cfg = newConfig
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已更新"})
}

// handleGetScreens 获取屏幕列表
func (s *Server) handleGetScreens(c *gin.Context) {
	screens := capture.GetScreens()
	c.JSON(http.StatusOK, screens)
}

// handleGetScreenshots 获取截图列表
func (s *Server) handleGetScreenshots(c *gin.Context) {
	// 分页参数
	limit := 50
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	screenshots, err := s.storageMgr.GetRecentScreenshots(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, screenshots)
}

// handleGetScreenshot 获取单个截图
func (s *Server) handleGetScreenshot(c *gin.Context) {
	// 这里可以返回图片文件
	c.JSON(http.StatusOK, gin.H{"message": "待实现"})
}

// handleDeleteScreenshot 删除截图
func (s *Server) handleDeleteScreenshot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "待实现"})
}

// handleCaptureNow 立即截图
func (s *Server) handleCaptureNow(c *gin.Context) {
	var req struct {
		ScreenIndex int `json:"screen_index"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	screenshot, err := s.captureEng.CaptureNow(req.ScreenIndex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, screenshot)
}

// handleGetSummaries 获取工作总结列表
func (s *Server) handleGetSummaries(c *gin.Context) {
	// 默认获取今天的
	date := time.Now()
	if d := c.Query("date"); d != "" {
		parsed, err := time.Parse("2006-01-02", d)
		if err == nil {
			date = parsed
		}
	}

	summaries, err := s.storageMgr.GetWorkSummaries(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// handleGetSummariesByDate 获取指定日期的总结
func (s *Server) handleGetSummariesByDate(c *gin.Context) {
	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日期格式"})
		return
	}

	summaries, err := s.storageMgr.GetWorkSummaries(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summaries)
}

// handleAnalyzeNow 立即触发 AI 分析（按整点分段，空段留空）
// 行为：
//   1. 获取当天截图的最早和最晚时间；
//   2. 第一段：从最早截图时间 -> 下一个整点；
//   3. 中间段：整点 -> 整点；
//   4. 最后一段：整点 -> 最后截图时间（或当前时间）；
//   5. 如果某段没有截图，则不调用 AI，直接写入空占位。
func (s *Server) handleAnalyzeNow(c *gin.Context) {
	var req struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}
	_ = c.ShouldBindJSON(&req)

	// 1. 获取当天截图
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	screenshots, err := s.storageMgr.GetScreenshots(startOfDay, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(screenshots) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "今天还没有可用的截图数据，请先开始截屏后再分析"})
		return
	}

	firstTs := screenshots[0].Timestamp
	lastTs := screenshots[len(screenshots)-1].Timestamp

	// 2. 计算整点边界的时间段
	segments := []struct {
		Start, End time.Time
		HasData    bool
	}{}

	// 计算第一个整点边界（向上取整到下一个整点）
	firstHourEnd := time.Date(
		firstTs.Year(), firstTs.Month(), firstTs.Day(),
		firstTs.Hour()+1, 0, 0, 0, firstTs.Location(),
	)

	// 第一段：从第一张截图到下一个整点
	currentStart := firstTs
	currentEnd := firstHourEnd

	// 如果 lastTs 在第一个整点之前，整个数据只有一段
	if lastTs.Before(firstHourEnd) || lastTs.Equal(firstHourEnd) {
		currentEnd = lastTs
	}

	for {
		// 检查该段是否有截图
		hasData := false
		for _, ss := range screenshots {
			if (ss.Timestamp.Equal(currentStart) || ss.Timestamp.After(currentStart)) &&
				ss.Timestamp.Before(currentEnd) {
				hasData = true
				break
			}
		}

		segments = append(segments, struct {
			Start, End time.Time
			HasData    bool
		}{
			Start:   currentStart,
			End:     currentEnd,
			HasData: hasData,
		})

		// 如果已达到或超过最后截图时间，结束
		if currentEnd.Equal(lastTs) || currentEnd.After(lastTs) {
			break
		}

		// 下一段：从当前结束时间（整点）开始
		currentStart = currentEnd
		// 下一个结束时间：下一个整点
		currentEnd = currentStart.Add(1 * time.Hour)

		// 如果下一个整点超过 lastTs，则用 lastTs 作为结束
		if currentEnd.After(lastTs) {
			currentEnd = lastTs
		}
	}

	// 3. 清空当天已有的总结
	if err := s.storageMgr.DeleteWorkSummariesForDate(now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("清空今日工作总结失败: %v", err)})
		return
	}

	// 4. 逐段分析或写空占位
	var results []*models.WorkSummary
	for _, seg := range segments {
		if !seg.HasData {
			// 没有截图数据，写入空占位记录
			emptySummary := &models.WorkSummary{
				StartTime:  seg.Start,
				EndTime:    seg.End,
				Summary:    "暂无截屏内容",
				Activities: []models.Activity{},
				AppUsage:   map[string]int{},
				CreatedAt:  time.Now(),
			}
			if err := s.storageMgr.SaveWorkSummary(emptySummary); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存空占位失败: %v", err)})
				return
			}
			results = append(results, emptySummary)
		} else {
			// 有截图，调用 AI 分析
			summary, err := s.aiAnalyzer.AnalyzePeriod(seg.Start, seg.End)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			results = append(results, summary)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "分析完成",
		"summaries": results,
	})
}

// handleGetTodayStats 获取今日统计
func (s *Server) handleGetTodayStats(c *gin.Context) {
	screenshots, summaries, err := s.storageMgr.GetTodayStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"today_captures":  screenshots,
		"today_summaries": summaries,
	})
}

// handleGetStorageStats 获取存储统计
func (s *Server) handleGetStorageStats(c *gin.Context) {
	stats, err := s.storageMgr.GetStorageStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleOpenStorageFolder 打开截图存储文件夹
func (s *Server) handleOpenStorageFolder(c *gin.Context) {
	storageCfg := s.configMgr.GetStorage()
	screenshotsDir := storageCfg.ScreenshotsDir

	// 转换为绝对路径
	absPath, err := filepath.Abs(screenshotsDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取绝对路径失败: " + err.Error()})
		return
	}

	// 获取目标参数：today（今日目录）或 root（根目录）
	target := c.Query("target")
	var targetDir string

	if target == "today" {
		// 强制打开今日目录
		today := time.Now().Format("2006-01-02")
		targetDir = filepath.Join(absPath, today)
	} else {
		// 打开截图根目录（data/screenshots）
		targetDir = absPath
	}

	// 确保目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败: " + err.Error()})
		return
	}

	fmt.Printf("📂 打开文件夹 (target=%s): %s\n", target, targetDir)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Windows 下直接打开目录（不使用 /select, 以避免仅选中目录而不打开）
		// 使用 explorer 直接打开目录会自动切换到前台
		cmd = exec.Command("explorer", targetDir)
	case "darwin":
		cmd = exec.Command("open", targetDir)
	case "linux":
		cmd = exec.Command("xdg-open", targetDir)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的操作系统"})
		return
	}

	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "打开文件夹失败: " + err.Error(),
			"path":  targetDir,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已打开截图存储文件夹",
		"path":    targetDir,
	})
}
// handleStartService 启动服务
func (s *Server) handleStartService(c *gin.Context) {
	// 自动启用截屏配置
	if err := s.configMgr.Update(func(cfg *models.AppConfig) {
		cfg.Capture.Enabled = true
	}); err != nil {
		fmt.Printf("⚠️ 启用截屏配置失败: %v\n", err)
	}

	if err := s.captureEng.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "截屏服务已启动"})
}

// handleStopService 停止服务
func (s *Server) handleStopService(c *gin.Context) {
	if err := s.captureEng.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "服务已停止"})
}

// handleGetStatus 获取服务状态
func (s *Server) handleGetStatus(c *gin.Context) {
	screenshots, summaries, _ := s.storageMgr.GetTodayStats()

	status := models.ServiceStatus{
		Running:        s.captureEng.IsRunning(),
		CaptureEnabled: s.configMgr.GetCapture().Enabled,
		LastCapture:    s.captureEng.GetLastCapture(),
		TodayCaptures:  screenshots,
		TodaySummaries: summaries,
	}

	c.JSON(http.StatusOK, status)
}

// handleTestAIConnection 测试 AI 连接并获取模型列表
func (s *Server) handleTestAIConnection(c *gin.Context) {
	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
		BaseURL  string `json:"base_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "API 密钥不能为空"})
		return
	}

	// 测试连接并获取模型列表
	models, err := s.aiAnalyzer.TestConnection(req.Provider, req.APIKey, req.BaseURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"models":  models,
	})
}

