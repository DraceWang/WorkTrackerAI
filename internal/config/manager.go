package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"worktracker/pkg/models"
)

// Manager 配置管理器
type Manager struct {
	config     *models.AppConfig
	configPath string
	mu         sync.RWMutex
}

// NewManager 创建配置管理器
func NewManager(configPath string) (*Manager, error) {
	m := &Manager{
		configPath: configPath,
	}

	if err := m.load(); err != nil {
		// 如果加载失败，使用默认配置
		m.config = models.DefaultConfig()
		if err := m.Save(); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	return m, nil
}

// load 加载配置
func (m *Manager) load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file not found")
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	var config models.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.config = &config
	return nil
}

// save 保存配置 (内部方法,不加锁)
func (m *Manager) save() error {
	// 确保目录存在
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("✅ 配置已保存到:", m.configPath)
	return nil
}

// Save 保存配置 (公共方法,加锁)
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.save()
}

// Get 获取配置（只读）
func (m *Manager) Get() *models.AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 返回副本，避免外部修改
	configCopy := *m.config
	return &configCopy
}

// Update 更新配置
func (m *Manager) Update(updater func(*models.AppConfig)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	updater(m.config)
	return m.save() // 使用内部 save() 方法,避免重复加锁
}

// GetCapture 获取截屏配置
func (m *Manager) GetCapture() models.CaptureConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Capture
}

// GetSchedule 获取工作时间配置
func (m *Manager) GetSchedule() models.WorkSchedule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Schedule
}

// GetAI 获取 AI 配置
func (m *Manager) GetAI() models.AIConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.AI
}

// GetStorage 获取存储配置
func (m *Manager) GetStorage() models.StorageConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Storage
}

// GetServer 获取服务器配置
func (m *Manager) GetServer() models.ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server
}
