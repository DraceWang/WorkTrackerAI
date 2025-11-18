package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

// TimeInRange 检查当前时间是否在指定范围内
func TimeInRange(startTime, endTime string) (bool, error) {
	now := time.Now()

	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return false, fmt.Errorf("invalid start time format: %w", err)
	}

	end, err := time.Parse("15:04", endTime)
	if err != nil {
		return false, fmt.Errorf("invalid end time format: %w", err)
	}

	// 将时间应用到今天
	startToday := time.Date(now.Year(), now.Month(), now.Day(),
		start.Hour(), start.Minute(), 0, 0, now.Location())
	endToday := time.Date(now.Year(), now.Month(), now.Day(),
		end.Hour(), end.Minute(), 0, 0, now.Location())

	// 处理跨天的情况
	if endToday.Before(startToday) {
		endToday = endToday.Add(24 * time.Hour)
	}

	return now.After(startToday) && now.Before(endToday), nil
}

// IsDayInList 检查星期几是否在列表中
func IsDayInList(day time.Weekday, days []int) bool {
	dayInt := int(day)
	for _, d := range days {
		if d == dayInt {
			return true
		}
	}
	return false
}

// HashBytes 计算字节数组的 MD5 哈希
func HashBytes(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// TruncateString 截断字符串
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
