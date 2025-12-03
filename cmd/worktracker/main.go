package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"WorkTrackerAI/internal/ai"
	"WorkTrackerAI/internal/capture"
	"WorkTrackerAI/internal/config"
	"WorkTrackerAI/internal/scheduler"
	"WorkTrackerAI/internal/server"
	"WorkTrackerAI/internal/singleton"
	"WorkTrackerAI/internal/storage"
	"WorkTrackerAI/internal/tray"
	"WorkTrackerAI/pkg/logger"
)

const (
	AppName    = "WorkTrackerAI"
	AppVersion = "1.49.3"
)

// getAppDataDir 获取应用数据目录
// Windows: %LOCALAPPDATA%\worktrackerAIAI
// 如果环境变量不存在，则使用当前工作目录
func getAppDataDir() string {
	// 优先使用 LOCALAPPDATA 环境变量（Windows）
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, AppName)
	}

	// 其他平台或环境变量不存在时，使用当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("❌ 无法获取工作目录: %v", err)
	}
	return workDir
}

func main() {
	// printBanner()

	// 单实例检测 - 防止程序重复启动
	mutex, err := singleton.EnsureSingleInstance(AppName)
	if err != nil {
		// 已有实例在运行，退出
		os.Exit(1)
	}
	// 确保程序退出时释放互斥锁
	defer mutex.Close()

	// 获取应用数据目录
	appDataDir := getAppDataDir()

	// 确保应用数据目录存在
	if err := os.MkdirAll(appDataDir, 0755); err != nil {
		log.Fatalf("❌ 创建应用数据目录失败 %s: %v", appDataDir, err)
	}

	// 初始化配置管理器
	configPath := filepath.Join(appDataDir, "data", "config.json")
	configMgr, err := config.NewManager(configPath)
	if err != nil {
		log.Fatalf("❌ 初始化配置管理器失败: %v", err)
	}
	fmt.Println("✅ 配置管理器初始化完成")

	// 确保必要的目录存在
	storageCfg := configMgr.GetStorage()
	requiredDirs := []string{
		storageCfg.DataDir,
		filepath.Join(storageCfg.DataDir, "screenshots"),
		filepath.Join(storageCfg.DataDir, "logs"),
		filepath.Join(storageCfg.DataDir, "summaries"),
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("❌ 创建目录失败 %s: %v", dir, err)
		}
	}
	fmt.Println("✅ 目录结构初始化完成")

	// 初始化日志系统
	logsDir := filepath.Join(storageCfg.DataDir, "logs")
	if err := logger.Init(logsDir, false); err != nil {
		log.Printf("⚠️ 日志系统初始化失败: %v, 使用控制台输出", err)
	} else {
		fmt.Println("✅ 日志系统初始化完成")
		logger.Info("==================== worktrackerAI %s 启动 ====================", AppVersion)
		logger.Info("应用数据目录: %s", appDataDir)
		logger.Info("数据目录: %s", storageCfg.DataDir)
	}

	// 初始化存储管理器
	storageMgr, err := storage.NewManager(storageCfg.DataDir)
	if err != nil {
		log.Fatalf("❌ 初始化存储管理器失败: %v", err)
	}
	fmt.Println("✅ 存储管理器初始化完成")

	// 初始化截屏引擎
	captureEng := capture.NewEngine(configMgr, storageMgr)
	fmt.Println("✅ 截屏引擎初始化完成")

	// 初始化 AI 分析器
	aiAnalyzer := ai.NewAnalyzer(configMgr, storageMgr)
	fmt.Println("✅ AI 分析器初始化完成")

	// 初始化任务调度器
	sched := scheduler.NewScheduler(configMgr, storageMgr, aiAnalyzer, captureEng)
	if err := sched.Start(); err != nil {
		log.Fatalf("❌ 启动任务调度器失败: %v", err)
	}

	// 初始化 Web 服务器
	webServer := server.NewServer(configMgr, storageMgr, captureEng, aiAnalyzer, AppVersion)

	// 启动 Web 服务器（在独立 goroutine 中）
	go func() {
		if err := webServer.Start(); err != nil {
			log.Printf("❌ Web 服务器错误: %v", err)
		}
	}()

	// 获取 Web 地址
	serverCfg := configMgr.GetServer()
	webURL := fmt.Sprintf("http://%s:%d", serverCfg.Host, serverCfg.Port)

	// 初始化系统托盘
	fmt.Println("🎯 启动系统托盘...")
	trayApp := tray.NewTrayApp(
		captureEng,
		sched,
		webURL,
		serverCfg.AutoOpenBrowser, // 传递自动打开浏览器配置
		func() {
			// 清理资源
			fmt.Println("📦 正在清理资源...")
			webServer.Shutdown()
			storageMgr.Close()
			fmt.Println("✅ 资源清理完成")
		},
	)

	// 运行托盘应用（阻塞）
	trayApp.Run()
}

// printBanner 打印欢迎信息
func printBanner() {
	banner := `
╔═══════════════════════════════════════════════╗
║                                               ║
║     🚀 worktrackerAI AI - 工作追踪工具          ║
║     版本: ` + AppVersion + `                               ║
║                                               ║
║     📸 自动截屏 + 🤖 AI 分析 + 📊 时间轴       ║
║                                               ║
╚═══════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
