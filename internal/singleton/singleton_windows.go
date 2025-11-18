package singleton

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	user32          = syscall.NewLazyDLL("user32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
	procMessageBox  = user32.NewProc("MessageBoxW")
)

// Mutex 持有互斥锁句柄
type Mutex struct {
	handle syscall.Handle
}

// CreateMutex 创建命名互斥锁，用于单实例检测
// mutexName: 互斥锁名称
// 返回: 互斥锁对象，是否是首次创建
func CreateMutex(mutexName string) (*Mutex, bool, error) {
	mutexNamePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, false, err
	}

	ret, _, err := procCreateMutex.Call(
		0,                            // 默认安全属性
		0,                            // 不初始拥有
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥锁名称
	)

	if ret == 0 {
		return nil, false, err
	}

	// ERROR_ALREADY_EXISTS = 183
	isFirst := err != syscall.ERROR_ALREADY_EXISTS

	return &Mutex{handle: syscall.Handle(ret)}, isFirst, nil
}

// Close 释放互斥锁
func (m *Mutex) Close() error {
	if m.handle != 0 {
		return syscall.CloseHandle(m.handle)
	}
	return nil
}

// ShowMessageBox 显示 Windows 消息框
func ShowMessageBox(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	messagePtr, _ := syscall.UTF16PtrFromString(message)

	// MB_OK = 0, MB_ICONWARNING = 0x30
	procMessageBox.Call(
		0, // 无父窗口
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		0x30, // MB_ICONWARNING | MB_OK
	)
}

// EnsureSingleInstance 确保只有一个实例运行
// appName: 应用名称，用于互斥锁名称和提示消息
// 返回: 互斥锁对象（需要在程序退出时调用 Close）
func EnsureSingleInstance(appName string) (*Mutex, error) {
	mutexName := fmt.Sprintf("Global\\%s_SingleInstance", appName)

	mutex, isFirst, err := CreateMutex(mutexName)
	if err != nil {
		return nil, fmt.Errorf("创建互斥锁失败: %w", err)
	}

	if !isFirst {
		// 已经有实例在运行
		ShowMessageBox(
			appName+" - 警告",
			appName+" 已经在运行中！\n\n请在系统托盘查找图标，右键可打开管理界面。\n\n如需退出程序，请右键托盘图标选择退出。",
		)
		mutex.Close()
		return nil, fmt.Errorf("应用已在运行")
	}

	return mutex, nil
}
