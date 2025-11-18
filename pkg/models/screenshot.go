package models

import "time"

// Screenshot 截图数据模型
type Screenshot struct {
	ID          int64     `json:"id" db:"id"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	ScreenIndex int       `json:"screen_index" db:"screen_index"`
	FilePath    string    `json:"file_path" db:"file_path"`
	FileSize    int64     `json:"file_size" db:"file_size"`
	Resolution  string    `json:"resolution" db:"resolution"`
	Analyzed    bool      `json:"analyzed" db:"analyzed"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// WorkSummary 工作总结
type WorkSummary struct {
	ID         int64      `json:"id" db:"id"`
	StartTime  time.Time  `json:"start_time" db:"start_time"`
	EndTime    time.Time  `json:"end_time" db:"end_time"`
	Summary    string     `json:"summary" db:"summary"`
	Activities []Activity `json:"activities" db:"-"`
	AppUsage   map[string]int `json:"app_usage" db:"-"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// Activity 活动
type Activity struct {
	Name            string   `json:"name"`
	DurationMinutes int      `json:"duration_minutes"`
	Apps            []string `json:"apps"`
	Category        string   `json:"category"`
}

// ScreenInfo 屏幕信息
type ScreenInfo struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	IsPrimary  bool   `json:"is_primary"`
}

// StorageStats 存储统计
type StorageStats struct {
	TotalScreenshots int64  `json:"total_screenshots"`
	TotalSize        int64  `json:"total_size"`
	OldestDate       string `json:"oldest_date"`
	NewestDate       string `json:"newest_date"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Running         bool      `json:"running"`
	CaptureEnabled  bool      `json:"capture_enabled"`
	LastCapture     time.Time `json:"last_capture,omitempty"`
	LastAnalysis    time.Time `json:"last_analysis,omitempty"`
	TodayCaptures   int       `json:"today_captures"`
	TodaySummaries  int       `json:"today_summaries"`
}
