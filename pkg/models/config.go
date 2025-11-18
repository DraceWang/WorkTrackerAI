package models

// AppConfig 应用程序配置
type AppConfig struct {
	// 截屏配置
	Capture CaptureConfig `json:"capture"`

	// 工作时间配置
	Schedule WorkSchedule `json:"schedule"`

	// AI 配置
	AI AIConfig `json:"ai"`

	// 存储配置
	Storage StorageConfig `json:"storage"`

	// 服务器配置
	Server ServerConfig `json:"server"`
}

// CaptureConfig 截屏配置
type CaptureConfig struct {
	Interval        int   `json:"interval"`         // 截屏间隔（秒）
	SelectedScreens []int `json:"selected_screens"` // 选中的屏幕索引
	Quality         int   `json:"quality"`          // JPEG 质量 (1-100)
	Enabled         bool  `json:"enabled"`          // 是否启用截屏
}

// WorkSchedule 工作时间配置
type WorkSchedule struct {
	StartTime        string   `json:"start_time"`        // 开始时间 "09:00"
	EndTime          string   `json:"end_time"`          // 结束时间 "18:00"
	WorkDays         []int    `json:"work_days"`         // 工作日 (0=周日, 1=周一, ...)
	AnalysisInterval int      `json:"analysis_interval"` // AI 分析间隔（分钟）
	Enabled          bool     `json:"enabled"`           // 是否启用时间限制
}

// AIConfig AI 配置
type AIConfig struct {
	Provider     string  `json:"provider"`      // openai, claude, gemini, azure
	APIKey       string  `json:"api_key"`       // API 密钥
	Model        string  `json:"model"`         // 模型名称
	BaseURL      string  `json:"base_url"`      // Base URL (如 https://api.openai.com/v1)
	Endpoint     string  `json:"endpoint"`      // 自定义端点（Azure 专用）
	MaxTokens    int     `json:"max_tokens"`    // 最大 token 数
	Temperature  float32 `json:"temperature"`   // 温度参数
	MaxImages    int     `json:"max_images"`    // 单次分析最大图片数
}

// StorageConfig 存储配置
type StorageConfig struct {
	DataDir         string `json:"data_dir"`          // 数据目录
	ScreenshotsDir  string `json:"screenshots_dir"`   // 截图存储目录
	LogsDir         string `json:"logs_dir"`          // 日志存储目录
	RetentionDays   int    `json:"retention_days"`    // 截图保留天数
	Compression     bool   `json:"compression"`       // 是否压缩
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int    `json:"port"`          // 端口号
	Host         string `json:"host"`          // 主机地址
	EnableCORS   bool   `json:"enable_cors"`   // 是否启用 CORS
	AutoOpenBrowser bool `json:"auto_open_browser"` // 启动时自动打开浏览器
}

// DefaultConfig 返回默认配置
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Capture: CaptureConfig{
			Interval:        3,
			SelectedScreens: []int{0},
			Quality:         75,
			Enabled:         false,
		},
		Schedule: WorkSchedule{
			StartTime:        "09:00",
			EndTime:          "18:00",
			WorkDays:         []int{1, 2, 3, 4, 5}, // 周一到周五
			AnalysisInterval: 60,
			Enabled:          true,
		},
		AI: AIConfig{
			Provider:    "openai",
			Model:       "gpt-4o",
			MaxTokens:   2000,
			Temperature: 0.3,
			MaxImages:   20,
		},
		Storage: StorageConfig{
			DataDir:         "./data",
			ScreenshotsDir:  "./data/screenshots",
			LogsDir:         "./data/logs",
			RetentionDays:   30,
			Compression:     true,
		},
		Server: ServerConfig{
			Port:            9527,
			Host:            "localhost",
			EnableCORS:      true,
			AutoOpenBrowser: true,
		},
	}
}
