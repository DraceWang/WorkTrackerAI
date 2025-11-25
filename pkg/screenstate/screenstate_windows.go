//go:build windows
// +build windows

package screenstate

import (
	"syscall"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
)

const (
	SPI_GETSCREENSAVERRUNNING = 0x0072
)

// IsScreenLocked 检测屏幕是否被锁定
// 通过检查是否有前台窗口来判断（锁屏时没有前台窗口）
func IsScreenLocked() bool {
	hwnd, _, _ := procGetForegroundWindow.Call()
	// 如果没有前台窗口，可能是锁屏状态
	if hwnd == 0 {
		return true
	}
	return false
}

// IsScreensaverRunning 检测屏幕保护程序是否正在运行
func IsScreensaverRunning() bool {
	var running uint32
	ret, _, _ := procSystemParametersInfo.Call(
		uintptr(SPI_GETSCREENSAVERRUNNING),
		0,
		uintptr(unsafe.Pointer(&running)),
		0,
	)
	
	if ret == 0 {
		// API调用失败，假设屏保未运行
		return false
	}
	
	return running != 0
}

// IsScreenActive 检测屏幕是否处于活跃状态（未锁定、未运行屏保）
func IsScreenActive() bool {
	// 如果屏保正在运行，屏幕不活跃
	if IsScreensaverRunning() {
		return false
	}
	
	// 如果屏幕被锁定，屏幕不活跃
	if IsScreenLocked() {
		return false
	}
	
	return true
}
