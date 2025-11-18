package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"worktracker/pkg/models"

	_ "modernc.org/sqlite"
)

// Manager 存储管理器
type Manager struct {
	db     *sql.DB
	dbPath string
}

// NewManager 创建存储管理器
func NewManager(dataDir string) (*Manager, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "worktracker.db")

	// 注意：modernc.org/sqlite 的驱动名称是 "sqlite" 而不是 "sqlite3"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	m := &Manager{
		db:     db,
		dbPath: dbPath,
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return m, nil
}

// initSchema 初始化数据库表结构
func (m *Manager) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS screenshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		screen_index INTEGER NOT NULL,
		file_path TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		resolution TEXT,
		analyzed BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_screenshots_timestamp ON screenshots(timestamp);
	CREATE INDEX IF NOT EXISTS idx_screenshots_analyzed ON screenshots(analyzed);

	CREATE TABLE IF NOT EXISTS work_summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		start_time DATETIME NOT NULL,
		end_time DATETIME NOT NULL,
		summary TEXT NOT NULL,
		activities_json TEXT,
		app_usage_json TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_summaries_date ON work_summaries(date(start_time));
	`

	_, err := m.db.Exec(schema)
	return err
}

// Close 关闭数据库
func (m *Manager) Close() error {
	return m.db.Close()
}

// SaveScreenshot 保存截图记录
func (m *Manager) SaveScreenshot(ss *models.Screenshot) error {
	query := `
		INSERT INTO screenshots (timestamp, screen_index, file_path, file_size, resolution, analyzed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := m.db.Exec(query,
		ss.Timestamp,
		ss.ScreenIndex,
		ss.FilePath,
		ss.FileSize,
		ss.Resolution,
		ss.Analyzed,
		ss.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert screenshot: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get insert id: %w", err)
	}

	ss.ID = id
	return nil
}

// GetScreenshots 获取指定时间范围的截图
func (m *Manager) GetScreenshots(start, end time.Time) ([]*models.Screenshot, error) {
	query := `
		SELECT id, timestamp, screen_index, file_path, file_size, resolution, analyzed, created_at
		FROM screenshots
		WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := m.db.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query screenshots: %w", err)
	}
	defer rows.Close()

	var screenshots []*models.Screenshot
	for rows.Next() {
		ss := &models.Screenshot{}
		err := rows.Scan(
			&ss.ID,
			&ss.Timestamp,
			&ss.ScreenIndex,
			&ss.FilePath,
			&ss.FileSize,
			&ss.Resolution,
			&ss.Analyzed,
			&ss.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan screenshot: %w", err)
		}
		screenshots = append(screenshots, ss)
	}

	return screenshots, nil
}

// GetRecentScreenshots 获取最近的 N 个截图
func (m *Manager) GetRecentScreenshots(limit int) ([]*models.Screenshot, error) {
	query := `
		SELECT id, timestamp, screen_index, file_path, file_size, resolution, analyzed, created_at
		FROM screenshots
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query screenshots: %w", err)
	}
	defer rows.Close()

	var screenshots []*models.Screenshot
	for rows.Next() {
		ss := &models.Screenshot{}
		err := rows.Scan(
			&ss.ID,
			&ss.Timestamp,
			&ss.ScreenIndex,
			&ss.FilePath,
			&ss.FileSize,
			&ss.Resolution,
			&ss.Analyzed,
			&ss.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan screenshot: %w", err)
		}
		screenshots = append(screenshots, ss)
	}

	return screenshots, nil
}

// MarkScreenshotAnalyzed 标记截图已分析
func (m *Manager) MarkScreenshotAnalyzed(id int64) error {
	query := `UPDATE screenshots SET analyzed = 1 WHERE id = ?`
	_, err := m.db.Exec(query, id)
	return err
}

// DeleteOldScreenshots 删除旧截图
func (m *Manager) DeleteOldScreenshots(retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	// 首先获取要删除的截图文件路径
	query := `SELECT file_path FROM screenshots WHERE timestamp < ?`
	rows, err := m.db.Query(query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to query old screenshots: %w", err)
	}

	var filePaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan file path: %w", err)
		}
		filePaths = append(filePaths, path)
	}
	rows.Close()

	// 删除文件
	for _, path := range filePaths {
		os.Remove(path) // 忽略错误
	}

	// 从数据库删除记录
	deleteQuery := `DELETE FROM screenshots WHERE timestamp < ?`
	result, err := m.db.Exec(deleteQuery, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old screenshots: %w", err)
	}

	return result.RowsAffected()
}

// SaveWorkSummary 保存工作总结
func (m *Manager) SaveWorkSummary(summary *models.WorkSummary) error {
	activitiesJSON, err := json.Marshal(summary.Activities)
	if err != nil {
		return fmt.Errorf("failed to marshal activities: %w", err)
	}

	appUsageJSON, err := json.Marshal(summary.AppUsage)
	if err != nil {
		return fmt.Errorf("failed to marshal app usage: %w", err)
	}

	query := `
		INSERT INTO work_summaries (start_time, end_time, summary, activities_json, app_usage_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := m.db.Exec(query,
		summary.StartTime,
		summary.EndTime,
		summary.Summary,
		string(activitiesJSON),
		string(appUsageJSON),
		summary.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert work summary: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get insert id: %w", err)
	}

	summary.ID = id
	return nil
}

// GetWorkSummaries 获取指定日期的工作总结
func (m *Manager) GetWorkSummaries(date time.Time) ([]*models.WorkSummary, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, start_time, end_time, summary, activities_json, app_usage_json, created_at
		FROM work_summaries
		WHERE start_time >= ? AND start_time < ?
		ORDER BY start_time ASC
	`

	rows, err := m.db.Query(query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query work summaries: %w", err)
	}
	defer rows.Close()

	var summaries []*models.WorkSummary
	for rows.Next() {
		ws := &models.WorkSummary{}
		var activitiesJSON, appUsageJSON string

		err := rows.Scan(
			&ws.ID,
			&ws.StartTime,
			&ws.EndTime,
			&ws.Summary,
			&activitiesJSON,
			&appUsageJSON,
			&ws.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan work summary: %w", err)
		}

		// 反序列化 JSON
		if activitiesJSON != "" {
			if err := json.Unmarshal([]byte(activitiesJSON), &ws.Activities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal activities: %w", err)
			}
		}

		if appUsageJSON != "" {
			if err := json.Unmarshal([]byte(appUsageJSON), &ws.AppUsage); err != nil {
				return nil, fmt.Errorf("failed to unmarshal app usage: %w", err)
			}
		}

		summaries = append(summaries, ws)
	}

	return summaries, nil
}

// DeleteWorkSummariesForDate 删除指定日期的所有工作总结（用于“立即分析”重新生成）
func (m *Manager) DeleteWorkSummariesForDate(date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	_, err := m.db.Exec(`DELETE FROM work_summaries WHERE start_time >= ? AND start_time < ?`, startOfDay, endOfDay)
	if err != nil {
		return fmt.Errorf("failed to delete work summaries: %w", err)
	}
	return nil
}

// GetStorageStats 获取存储统计信息
func (m *Manager) GetStorageStats() (*models.StorageStats, error) {
	stats := &models.StorageStats{}

	// 获取截图总数和总大小
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(file_size), 0) as total_size,
			MIN(date(timestamp)) as oldest,
			MAX(date(timestamp)) as newest
		FROM screenshots
	`

	var oldest, newest sql.NullString
	err := m.db.QueryRow(query).Scan(
		&stats.TotalScreenshots,
		&stats.TotalSize,
		&oldest,
		&newest,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}

	if oldest.Valid {
		stats.OldestDate = oldest.String
	}
	if newest.Valid {
		stats.NewestDate = newest.String
	}

	return stats, nil
}

// GetTodayStats 获取今日统计
func (m *Manager) GetTodayStats() (screenshots int, summaries int, err error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 今日截图数
	err = m.db.QueryRow(`SELECT COUNT(*) FROM screenshots WHERE timestamp >= ?`, startOfDay).Scan(&screenshots)
	if err != nil {
		return 0, 0, err
	}

	// 今日总结数
	err = m.db.QueryRow(`SELECT COUNT(*) FROM work_summaries WHERE start_time >= ?`, startOfDay).Scan(&summaries)
	if err != nil {
		return 0, 0, err
	}

	return screenshots, summaries, nil
}

// HasWorkSummaryForRange 判断指定时间段内是否已经存在工作总结
// 用于避免重复分析同一时间段（例如每个整点自动分析上一时间段）
func (m *Manager) HasWorkSummaryForRange(start, end time.Time) (bool, error) {
	var count int
	err := m.db.QueryRow(
		`SELECT COUNT(*) FROM work_summaries WHERE start_time >= ? AND start_time < ?`,
		start,
		end,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query work summaries for range: %w", err)
	}
	return count > 0, nil
}
