//go:build windows
// +build windows

package screenstate

import (
	"syscall"
	"unsafe"
)

var (
	user32                   = syscall.NewLazyDLL("user32.dll")
	wtsapi32                 = syscall.NewLazyDLL("wtsapi32.dll")
	procSystemParametersInfo = user32.NewProc("SystemParametersInfoW")
	procGetForegroundWindow  = user32.NewProc("GetForegroundWindow")
	procGetClassNameW        = user32.NewProc("GetClassNameW")
	procWTSQuerySessionInfo  = wtsapi32.NewProc("WTSQuerySessionInformationW")
	procWTSFreeMemory        = wtsapi32.NewProc("WTSFreeMemory")
)

const (
	SPI_GETSCREENSAVERRUNNING = 0x0072
	WTS_CURRENT_SERVER_HANDLE = 0
	WTS_CURRENT_SESSION       = 0xFFFFFFFF
	WTSSessionInfoEx          = 25
)

// IsScreenLocked 检测屏幕是否被锁定
// 使用多种方法综合判断以提高准确性
func IsScreenLocked() bool {
	// 方法1：检查前台窗口的类名
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd != 0 {
		className := make([]uint16, 256)
		procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&className[0])), 256)
		clsName := syscall.UTF16ToString(className)
		
		// Windows 10/11 锁屏界面的类名
		if clsName == "Windows.UI.Core.CoreWindow" || 
		   clsName == "LockScreenBackstopFrame" ||
		   clsName == "SessionSwitchWindow" {
			return true
		}
	} else {
		// 如果没有前台窗口，也可能是锁屏状态
		return true
	}
	
	// 方法2：尝试使用 WTS API 检查会话状态
	// 这个方法在某些情况下更可靠
	var pBuffer uintptr
	var bytesReturned uint32
	
	ret, _, _ := procWTSQuerySessionInfo.Call(
		WTS_CURRENT_SERVER_HANDLE,
		WTS_CURRENT_SESSION,
		WTSSessionInfoEx,
		uintptr(unsafe.Pointer(&pBuffer)),
		uintptr(unsafe.Pointer(&bytesReturned)),
	)
	
	if ret != 0 && pBuffer != 0 {
		// 成功获取会话信息
		defer procWTSFreeMemory.Call(pBuffer)
		// 注意：WTSSessionInfoEx 返回的结构体较复杂，这里简化处理
		// 如果能成功调用，说明会话处于活动状态
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
	
	isRunning := running != 0
	return isRunning
}

// IsScreenActive 检测屏幕是否处于活跃状态（未锁定、未运行屏保）
func IsScreenActive() bool {
	screensaverRunning := IsScreensaverRunning()
	screenLocked := IsScreenLocked()
	
	// 如果屏保正在运行，屏幕不活跃
	if screensaverRunning {
		return false
	}
	
	// 如果屏幕被锁定，屏幕不活跃
	if screenLocked {
		return false
	}
	
	return true
}

// GetScreenStateInfo 获取屏幕状态详细信息（用于日志记录）
func GetScreenStateInfo() (active bool, screensaverRunning bool, screenLocked bool) {
	screensaverRunning = IsScreensaverRunning()
	screenLocked = IsScreenLocked()
	active = !screensaverRunning && !screenLocked
	return
}
