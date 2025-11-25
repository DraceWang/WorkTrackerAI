//go:build !windows
// +build !windows

package screenstate

// IsScreenLocked 检测屏幕是否被锁定（非Windows平台暂不支持）
func IsScreenLocked() bool {
	// 非Windows平台暂不检测，假设未锁定
	return false
}

// IsScreensaverRunning 检测屏幕保护程序是否正在运行（非Windows平台暂不支持）
func IsScreensaverRunning() bool {
	// 非Windows平台暂不检测，假设未运行
	return false
}

// IsScreenActive 检测屏幕是否处于活跃状态（非Windows平台默认为活跃）
func IsScreenActive() bool {
	// 非Windows平台默认为活跃
	return true
}
