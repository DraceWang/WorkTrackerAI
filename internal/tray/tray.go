package tray

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"WorkTrackerAI/internal/capture"
	"WorkTrackerAI/internal/scheduler"

	"github.com/getlantern/systray"
)

// TrayApp 托盘应用
type TrayApp struct {
	captureEng      *capture.Engine
	scheduler       *scheduler.Scheduler
	webURL          string
	autoOpenBrowser bool
	onExit          func()
}

// NewTrayApp 创建托盘应用
func NewTrayApp(
	captureEng *capture.Engine,
	scheduler *scheduler.Scheduler,
	webURL string,
	autoOpenBrowser bool,
	onExit func(),
) *TrayApp {
	return &TrayApp{
		captureEng:      captureEng,
		scheduler:       scheduler,
		webURL:          webURL,
		autoOpenBrowser: autoOpenBrowser,
		onExit:          onExit,
	}
}

// Run 运行托盘应用
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onQuit)
}

// onReady 托盘准备就绪
func (t *TrayApp) onReady() {
	// 设置托盘图标和提示
	systray.SetIcon(getIcon())
	systray.SetTitle("WorkTracker")
	systray.SetTooltip("WorkTracker AI - 工作追踪工具\n点击右键查看选项")

	// 打开 Web 管理界面
	mOpen := systray.AddMenuItem("🌐 打开管理界面", "在浏览器中打开 Web 管理页面")

	systray.AddSeparator()

	// 退出程序
	mQuit := systray.AddMenuItem("❌ 退出程序", "退出 WorkTracker")

	// 事件循环
	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				fmt.Println("📱 打开浏览器...")
				t.openBrowser()

			case <-mQuit.ClickedCh:
				fmt.Println("🛑 用户请求退出...")
				systray.Quit()
				return
			}
		}
	}()

	// 自动启动截屏功能
	go func() {
		if err := t.captureEng.Start(); err != nil {
			fmt.Printf("⚠️ 自动启动截屏失败: %v\n", err)
		} else {
			fmt.Println("✅ 截屏功能已自动启动")
		}
	}()

	// 自动打开浏览器（延迟1秒确保Web服务器已完全启动）
	if t.autoOpenBrowser {
		go func() {
			time.Sleep(1 * time.Second)
			fmt.Printf("🌐 自动打开浏览器: %s\n", t.webURL)
			t.openBrowser()
		}()
	}
}

// onQuit 托盘退出
func (t *TrayApp) onQuit() {
	// 清理资源
	if t.captureEng.IsRunning() {
		t.captureEng.Stop()
	}
	if t.scheduler.IsRunning() {
		t.scheduler.Stop()
	}

	if t.onExit != nil {
		t.onExit()
	}

	fmt.Println("👋 WorkTracker 已退出")
}

// openBrowser 打开浏览器
func (t *TrayApp) openBrowser() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", t.webURL)
	case "darwin":
		cmd = exec.Command("open", t.webURL)
	default: // linux
		cmd = exec.Command("xdg-open", t.webURL)
	}

	if err := cmd.Start(); err != nil {
		fmt.Printf("无法打开浏览器: %v\n", err)
	}
}

// Quit 退出托盘
func (t *TrayApp) Quit() {
	systray.Quit()
	os.Exit(0)
}

// getIcon 获取托盘图标
//
// 注意：
//   - Windows 托盘推荐使用 .ico 格式；
//   - macOS / Linux 可使用 .png。
//
// 为了兼容性，这里会：
//   1. 以程序所在目录为基准查找 asserts 目录；
//   2. Windows 优先使用 WorkTraceAI_16x16.ico；
//   3. 其他系统优先使用 PNG 图标；
//   4. 找不到文件时回退到内置的简单 PNG 图标。
func getIcon() []byte {
	// 程序所在目录（而不是当前工作目录）
	exePath, err := os.Executable()
	baseDir := "."
	if err == nil {
		baseDir = filepath.Dir(exePath)
	}

	// 图标候选列表（按优先级）
	var candidates []string
	if runtime.GOOS == "windows" {
		// Windows 托盘图标优先使用 .ico
		candidates = []string{
			filepath.Join(baseDir, "asserts", "WorkTraceAI.ico"),
		}
	} else {
		// 其他平台优先用 PNG
		candidates = []string{
			filepath.Join(baseDir, "asserts", "WorkTraceAI.png"),
			filepath.Join(baseDir, "asserts", "WorkTraceAI_16x16.png"),
			filepath.Join(baseDir, "asserts", "WorkTraceAI.ico"),
		}
	}

	for _, iconPath := range candidates {
		if data, err := os.ReadFile(iconPath); err == nil && len(data) > 0 {
			fmt.Printf("✅ 使用托盘图标: %s (%.2f KB)\n", iconPath, float64(len(data))/1024)
			return data
		}
	}

	// 最后备选：内置默认图标
	fmt.Println("⚠️  未找到自定义图标文件，使用内置默认图标")
	fmt.Println("   提示：请确认 asserts 目录与可执行文件在同一目录")
	// 返回简单的备用图标（PNG 格式），这是一个 16x16 的简单蓝色方块 PNG
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x91, 0x68,
		0x36, 0x00, 0x00, 0x00, 0x19, 0x49, 0x44, 0x41,
		0x54, 0x28, 0x91, 0x63, 0x64, 0x60, 0xF8, 0x0F,
		0x04, 0x0C, 0x0C, 0x8C, 0x40, 0x06, 0x06, 0x46,
		0x20, 0x03, 0x03, 0x23, 0x00, 0x00, 0x0F, 0x70,
		0x01, 0x18, 0xE5, 0xD4, 0x8F, 0x4F, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42,
		0x60, 0x82,
	}
}
