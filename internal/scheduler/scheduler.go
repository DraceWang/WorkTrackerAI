package scheduler

import (
	"fmt"
	"sync"
	"time"

	"WorkTrackerAI/internal/ai"
	"WorkTrackerAI/internal/config"
	"WorkTrackerAI/internal/storage"

	"github.com/robfig/cron/v3"
)

// Scheduler 任务调度器
type Scheduler struct {
	cron       *cron.Cron
	configMgr  *config.Manager
	storageMgr *storage.Manager
	aiAnalyzer *ai.Analyzer
	mu         sync.Mutex
	running    bool
}

// NewScheduler 创建任务调度器
func NewScheduler(
	configMgr *config.Manager,
	storageMgr *storage.Manager,
	aiAnalyzer *ai.Analyzer,
) *Scheduler {
	return &Scheduler{
		cron:       cron.New(),
		configMgr:  configMgr,
		storageMgr: storageMgr,
		aiAnalyzer: aiAnalyzer,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	// 添加周期性 AI 分析任务
	schedule := s.configMgr.GetSchedule()
	analysisInterval := schedule.AnalysisInterval // 分钟

	// 每 N 分钟执行一次分析
	cronExpr := fmt.Sprintf("@every %dm", analysisInterval)
	_, err := s.cron.AddFunc(cronExpr, s.runAnalysis)
	if err != nil {
		return fmt.Errorf("failed to add analysis job: %w", err)
	}

	// 添加每日工作日报任务（工作结束前10分钟）
	if err := s.addDailyReportJob(); err != nil {
		fmt.Printf("⚠️ 添加每日日报任务失败: %v\n", err)
	}

	// 添加清理任务（每天凌晨 3 点）
	_, err = s.cron.AddFunc("0 3 * * *", s.runCleanup)
	if err != nil {
		return fmt.Errorf("failed to add cleanup job: %w", err)
	}

	// 每小时自动检查上一时间段是否需要分析（整点过5分钟执行，更稳妥）
	_, err = s.cron.AddFunc("5 * * * *", s.runHourlyPreviousSegmentAnalysis)
	if err != nil {
		return fmt.Errorf("failed to add hourly analysis job: %w", err)
	}

	s.cron.Start()
	s.running = true

	fmt.Printf("⏰ 任务调度器已启动 (AI分析间隔: %d分钟)\n", analysisInterval)
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cron.Stop()
	s.running = false
	fmt.Println("⏰ 任务调度器已停止")
}

// IsRunning 检查是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// runAnalysis 执行 AI 分析（使用整点边界）
func (s *Scheduler) runAnalysis() {
	fmt.Println("🤖 开始 AI 分析任务...")

	// 使用整点边界：从上一个整点到当前整点
	now := time.Now()
	currentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	prevHour := currentHour.Add(-1 * time.Hour)

	// 检查该时间段是否已存在总结，避免重复分析
	hasSummary, err := s.storageMgr.HasWorkSummaryForRange(prevHour, currentHour)
	if err != nil {
		fmt.Printf("⚠️ 检查历史总结失败: %v\n", err)
		return
	}
	if hasSummary {
		fmt.Printf("ℹ️ 时间段 %s - %s 已存在总结，跳过分析\n", prevHour.Format("15:04"), currentHour.Format("15:04"))
		return
	}

	summary, err := s.aiAnalyzer.AnalyzePeriod(prevHour, currentHour)
	if err != nil {
		fmt.Printf("❌ AI 分析失败: %v\n", err)
		return
	}

	fmt.Printf("✅ AI 分析完成: %s - %s: %s\n", prevHour.Format("15:04"), currentHour.Format("15:04"), summary.Summary)
}

// runCleanup 执行清理任务
func (s *Scheduler) runCleanup() {
	fmt.Println("🧹 开始清理旧数据...")

	storageCfg := s.configMgr.GetStorage()
	deleted, err := s.storageMgr.DeleteOldScreenshots(storageCfg.RetentionDays)
	if err != nil {
		fmt.Printf("❌ 清理失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 清理完成，删除了 %d 个旧截图\n", deleted)
}

// runHourlyPreviousSegmentAnalysis 每小时自动分析上一个整点时间段
// 行为：
//   - 每小时的第 5 分钟执行（例如 16:05）；
//   - 计算上一小时段 [H-1:00, H:00)；
//   - 如果该段结束时间在配置的工作结束时间内；
//   - 且该段尚无工作总结；
//   - 且该段内有截图；
//   - 则调用 AI 对该段进行一次分析，并保存结果。
func (s *Scheduler) runHourlyPreviousSegmentAnalysis() {
	fmt.Println("⏰ 每小时自动检查上一时间段是否需要分析...")

	schedule := s.configMgr.GetSchedule()
	if !schedule.Enabled {
		fmt.Println("ℹ️ 工作时间限制未启用，跳过自动整点分析")
		return
	}

	now := time.Now()

	// 解析工作时间配置
	startParts, err := time.Parse("15:04", schedule.StartTime)
	if err != nil {
		fmt.Printf("⚠️ 无效的开始时间配置: %v\n", err)
		return
	}
	endParts, err := time.Parse("15:04", schedule.EndTime)
	if err != nil {
		fmt.Printf("⚠️ 无效的结束时间配置: %v\n", err)
		return
	}

	workStart := time.Date(now.Year(), now.Month(), now.Day(), startParts.Hour(), startParts.Minute(), 0, 0, now.Location())
	workEnd := time.Date(now.Year(), now.Month(), now.Day(), endParts.Hour(), endParts.Minute(), 0, 0, now.Location())

	// 计算上一小时段 [prevStart, prevEnd)
	prevEnd := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	prevStart := prevEnd.Add(-1 * time.Hour)

	// 如果上一段结束时间超出工作结束时间，则不再自动分析
	if prevEnd.After(workEnd) {
		fmt.Println("ℹ️ 上一个整点已超过配置的工作结束时间，跳过自动分析")
		return
	}
	// 如果上一段开始时间早于工作开始时间，也不分析（例如早上还没到上班时间）
	if prevEnd.Before(workStart) || prevStart.Before(workStart) {
		fmt.Println("ℹ️ 上一时间段尚未进入工作时间范围，跳过自动分析")
		return
	}

	// 检查该时间段是否已存在总结，避免重复分析
	hasSummary, err := s.storageMgr.HasWorkSummaryForRange(prevStart, prevEnd)
	if err != nil {
		fmt.Printf("⚠️ 检查历史总结失败: %v\n", err)
		return
	}
	if hasSummary {
		fmt.Printf("ℹ️ 时间段 %s - %s 已存在总结，跳过自动分析\n", prevStart.Format("15:04"), prevEnd.Format("15:04"))
		return
	}

	// 检查该段内是否有截图
	screenshots, err := s.storageMgr.GetScreenshots(prevStart, prevEnd)
	if err != nil {
		fmt.Printf("⚠️ 获取截图失败: %v\n", err)
		return
	}
	if len(screenshots) == 0 {
		fmt.Printf("ℹ️ 时间段 %s - %s 内没有截图，跳过自动分析\n", prevStart.Format("15:04"), prevEnd.Format("15:04"))
		return
	}

	// 调用 AI 进行分析
	fmt.Printf("🤖 自动分析上一时间段: %s - %s...\n", prevStart.Format("15:04"), prevEnd.Format("15:04"))
	summary, err := s.aiAnalyzer.AnalyzePeriod(prevStart, prevEnd)
	if err != nil {
		fmt.Printf("❌ 自动整点分析失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 自动整点分析完成：%s - %s，摘要：%s\n", prevStart.Format("15:04"), prevEnd.Format("15:04"), summary.Summary)
}

// addDailyReportJob 添加每日工作日报任务
func (s *Scheduler) addDailyReportJob() error {
	schedule := s.configMgr.GetSchedule()

	// 解析工作结束时间
	endTime, err := time.Parse("15:04", schedule.EndTime)
	if err != nil {
		return fmt.Errorf("无效的结束时间格式: %w", err)
	}

	// 计算工作结束前10分钟的时间
	reportTime := endTime.Add(-10 * time.Minute)
	hour := reportTime.Hour()
	minute := reportTime.Minute()

	// 创建 cron 表达式：分 时 * * 1-5 (周一到周五)
	// 例如：17:50 -> "50 17 * * 1-5"
	cronExpr := fmt.Sprintf("%d %d * * 1-5", minute, hour)

	_, err = s.cron.AddFunc(cronExpr, s.runDailyReport)
	if err != nil {
		return fmt.Errorf("failed to add daily report job: %w", err)
	}

	fmt.Printf("📊 每日工作日报任务已添加 (工作日 %02d:%02d 生成)\n", hour, minute)
	return nil
}

// runDailyReport 生成每日工作日报
func (s *Scheduler) runDailyReport() {
	fmt.Println("📊 开始生成每日工作日报...")

	schedule := s.configMgr.GetSchedule()

	// 解析工作开始和结束时间
	now := time.Now()
	startTimeStr := schedule.StartTime
	endTimeStr := schedule.EndTime

	// 构造今天的工作开始和结束时间
	startParts, _ := time.Parse("15:04", startTimeStr)
	endParts, _ := time.Parse("15:04", endTimeStr)

	start := time.Date(now.Year(), now.Month(), now.Day(),
		startParts.Hour(), startParts.Minute(), 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(),
		endParts.Hour(), endParts.Minute(), 0, 0, now.Location())

	// 生成日报
	summary, err := s.aiAnalyzer.AnalyzePeriod(start, end)
	if err != nil {
		fmt.Printf("❌ 生成每日工作日报失败: %v\n", err)
		return
	}

	fmt.Println("✅ 每日工作日报生成完成！")
	fmt.Printf("📝 工作时间：%s - %s\n", start.Format("15:04"), end.Format("15:04"))
	fmt.Printf("📋 工作总结：%s\n", summary.Summary)

	// 统计工作时长
	totalMinutes := 0
	for _, act := range summary.Activities {
		totalMinutes += act.DurationMinutes
	}
	hours := totalMinutes / 60
	minutes := totalMinutes % 60
	fmt.Printf("⏱️  工作时长：%d小时%d分钟\n", hours, minutes)
}

