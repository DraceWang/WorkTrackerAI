package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"WorkTrackerAI/internal/config"
	"WorkTrackerAI/internal/storage"
	"WorkTrackerAI/pkg/logger"
	"WorkTrackerAI/pkg/models"
)

// Analyzer AI 分析器
type Analyzer struct {
	configMgr *config.Manager
	storage   *storage.Manager
	client    *http.Client
}

// NewAnalyzer 创建 AI 分析器
func NewAnalyzer(configMgr *config.Manager, storageMgr *storage.Manager) *Analyzer {
	return &Analyzer{
		configMgr: configMgr,
		storage:   storageMgr,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// AnalyzePeriod 分析指定时间段
func (a *Analyzer) AnalyzePeriod(start, end time.Time) (*models.WorkSummary, error) {
	logger.Info("==================== 开始AI分析 ====================")
	logger.Info("分析时段: %s - %s", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

	// 1. 获取时间段内的截图
	logger.Info("步骤1: 获取截图数据...")
	screenshots, err := a.storage.GetScreenshots(start, end)
	if err != nil {
		logger.Error("获取截图失败: %v", err)
		return nil, fmt.Errorf("failed to get screenshots: %w", err)
	}
	logger.Info("找到截图数量: %d", len(screenshots))

	if len(screenshots) == 0 {
		logger.Warn("时间段内没有截图数据")
		return nil, fmt.Errorf("未找到截图数据，请先点击'开始截屏'采集数据后再进行分析")
	}

	// 2. 智能采样
	logger.Info("步骤2: 智能采样...")
	maxImages := a.configMgr.GetAI().MaxImages
	sampled := a.sampleScreenshots(screenshots, maxImages)
	logger.Info("采样后数量: %d (最大: %d)", len(sampled), maxImages)

	// 3. 调用 LLM 分析
	logger.Info("步骤3: 调用AI分析 (提供商: %s, 模型: %s)...",
		a.configMgr.GetAI().Provider, a.configMgr.GetAI().Model)
	aiResponse, err := a.callLLM(sampled, start, end)
	if err != nil {
		logger.Error("AI分析失败: %v", err)
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}
	logger.Info("AI返回成功，响应长度: %d 字符", len(aiResponse))
	logger.Info("========== AI原始返回 ==========")
	logger.Info("%s", aiResponse)
	logger.Info("================================")

	// 4. 解析响应
	logger.Info("步骤4: 解析AI响应...")
	summary, err := a.parseResponse(aiResponse, start, end)
	if err != nil {
		logger.Error("解析响应失败: %v", err)
		logger.Error("原始响应内容: %s", aiResponse)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	logger.Info("解析成功: 活动数=%d, 应用数=%d", len(summary.Activities), len(summary.AppUsage))

	// 5. 保存总结到数据库
	logger.Info("步骤5: 保存到数据库...")
	if err := a.storage.SaveWorkSummary(summary); err != nil {
		logger.Error("保存到数据库失败: %v", err)
		return nil, fmt.Errorf("failed to save summary: %w", err)
	}
	logger.Info("数据库保存成功")

	// 6. 保存总结到本地Markdown文件
	logger.Info("步骤6: 保存到Markdown文件...")
	if err := a.saveSummaryToFile(summary); err != nil {
		logger.Error("保存分析结果到文件失败: %v", err)
		// 不中断流程，继续执行
	}

	// 7. 标记截图已分析
	logger.Info("步骤7: 标记截图已分析...")
	for _, ss := range screenshots {
		a.storage.MarkScreenshotAnalyzed(ss.ID)
	}

	logger.Info("==================== 分析完成 ====================")
	logger.Info("总结: %s", summary.Summary)
	logger.Info("时段: %s - %s, 截图数: %d, 采样数: %d",
		start.Format("15:04"), end.Format("15:04"), len(screenshots), len(sampled))

	return summary, nil
}

// sampleScreenshots 智能采样截图
func (a *Analyzer) sampleScreenshots(all []*models.Screenshot, maxCount int) []*models.Screenshot {
	if len(all) <= maxCount {
		return all
	}

	// 均匀采样
	sampled := make([]*models.Screenshot, 0, maxCount)
	step := len(all) / maxCount

	for i := 0; i < maxCount; i++ {
		idx := i * step
		if idx < len(all) {
			sampled = append(sampled, all[idx])
		}
	}

	return sampled
}

// callLLM 调用大语言模型
func (a *Analyzer) callLLM(screenshots []*models.Screenshot, start, end time.Time) (string, error) {
	cfg := a.configMgr.GetAI()

	switch cfg.Provider {
	case "openai":
		return a.callOpenAI(screenshots, start, end, cfg)
	case "claude":
		return a.callClaude(screenshots, start, end, cfg)
	case "deepseek":
		return a.callDeepSeek(screenshots, start, end, cfg)
	case "qwen", "tongyi":
		return a.callQwen(screenshots, start, end, cfg)
	case "doubao":
		return a.callDoubao(screenshots, start, end, cfg)
	default:
		return "", fmt.Errorf("unsupported AI provider: %s", cfg.Provider)
	}
}

// OpenAI 请求结构
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float32         `json:"temperature"`
}

type openAIMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type openAITextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIImageContent struct {
	Type     string           `json:"type"`
	ImageURL openAIImageURL   `json:"image_url"`
}

type openAIImageURL struct {
	URL string `json:"url"`
}

// OpenAI 响应结构
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// callOpenAI 调用 OpenAI API
func (a *Analyzer) callOpenAI(screenshots []*models.Screenshot, start, end time.Time, cfg models.AIConfig) (string, error) {
	// 构建消息内容
	content := []interface{}{
		openAITextContent{
			Type: "text",
			Text: a.buildPrompt(start, end),
		},
	}

	// 添加图片
	for _, ss := range screenshots {
		imageData, err := os.ReadFile(ss.FilePath)
		if err != nil {
			continue
		}

		base64Image := base64.StdEncoding.EncodeToString(imageData)
		content = append(content, openAIImageContent{
			Type: "image_url",
			ImageURL: openAIImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
			},
		})
	}

	// 构建请求
	reqBody := openAIRequest{
		Model: cfg.Model,
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: []interface{}{
					openAITextContent{
						Type: "text",
						Text: "你是一个工作分析助手，根据屏幕截图总结用户的工作内容。",
					},
				},
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	endpoint := "https://api.openai.com/v1/chat/completions"
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// 解析响应
	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// callClaude 调用 Claude API
func (a *Analyzer) callClaude(screenshots []*models.Screenshot, start, end time.Time, cfg models.AIConfig) (string, error) {
	// Claude API 实现（类似 OpenAI，但结构略有不同）
	return "", fmt.Errorf("Claude API not implemented yet")
}

// callDeepSeek 调用 DeepSeek API
// DeepSeek API 兼容 OpenAI 格式
func (a *Analyzer) callDeepSeek(screenshots []*models.Screenshot, start, end time.Time, cfg models.AIConfig) (string, error) {
	// DeepSeek 使用与 OpenAI 相同的 API 格式
	// 构建消息内容
	content := []interface{}{
		openAITextContent{
			Type: "text",
			Text: a.buildPrompt(start, end),
		},
	}

	// 添加图片
	for _, ss := range screenshots {
		imageData, err := os.ReadFile(ss.FilePath)
		if err != nil {
			continue
		}

		base64Image := base64.StdEncoding.EncodeToString(imageData)
		content = append(content, openAIImageContent{
			Type: "image_url",
			ImageURL: openAIImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
			},
		})
	}

	// 构建请求
	reqBody := openAIRequest{
		Model: cfg.Model, // 如 deepseek-chat
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: []interface{}{
					openAITextContent{
						Type: "text",
						Text: "你是一个工作分析助手，根据屏幕截图总结用户的工作内容。",
					},
				},
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// DeepSeek API 端点
	endpoint := "https://api.deepseek.com/v1/chat/completions"
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// 解析响应
	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// callQwen 调用通义千问 API
func (a *Analyzer) callQwen(screenshots []*models.Screenshot, start, end time.Time, cfg models.AIConfig) (string, error) {
	// 通义千问（阿里云）API 实现
	// 也兼容 OpenAI 格式
	content := []interface{}{
		openAITextContent{
			Type: "text",
			Text: a.buildPrompt(start, end),
		},
	}

	// 添加图片
	for _, ss := range screenshots {
		imageData, err := os.ReadFile(ss.FilePath)
		if err != nil {
			continue
		}

		base64Image := base64.StdEncoding.EncodeToString(imageData)
		content = append(content, openAIImageContent{
			Type: "image_url",
			ImageURL: openAIImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
			},
		})
	}

	// 构建请求
	reqBody := openAIRequest{
		Model: cfg.Model, // 如 qwen-vl-plus, qwen-vl-max
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: []interface{}{
					openAITextContent{
						Type: "text",
						Text: "你是一个工作分析助手，根据屏幕截图总结用户的工作内容。",
					},
				},
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 通义千问 API 端点
	endpoint := "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// 解析响应
	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// callDoubao 调用豆包 API
func (a *Analyzer) callDoubao(screenshots []*models.Screenshot, start, end time.Time, cfg models.AIConfig) (string, error) {
	// 豆包（字节跳动）API 实现
	// 也兼容 OpenAI 格式
	content := []interface{}{
		openAITextContent{
			Type: "text",
			Text: a.buildPrompt(start, end),
		},
	}

	// 添加图片
	for _, ss := range screenshots {
		imageData, err := os.ReadFile(ss.FilePath)
		if err != nil {
			continue
		}

		base64Image := base64.StdEncoding.EncodeToString(imageData)
		content = append(content, openAIImageContent{
			Type: "image_url",
			ImageURL: openAIImageURL{
				URL: fmt.Sprintf("data:image/jpeg;base64,%s", base64Image),
			},
		})
	}

	// 构建请求
	reqBody := openAIRequest{
		Model: cfg.Model, // 如 doubao-vision-pro
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: []interface{}{
					openAITextContent{
						Type: "text",
						Text: "你是一个工作分析助手，根据屏幕截图总结用户的工作内容。",
					},
				},
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 豆包 API 端点
	endpoint := "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.APIKey))

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// 解析响应
	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	return apiResp.Choices[0].Message.Content, nil
}

// buildPrompt 构建提示词
// 注意：
//   - summary 字段将用于“今日小结”，需要是若干条工作要点，格式严格为：
//     "1.xxx;2.xxx;3.xxx;" 这样的编号列表；
//   - activities 和 app_usage 依然用于前端详细信息展示。
func (a *Analyzer) buildPrompt(start, end time.Time) string {
	return fmt.Sprintf(`请分析 %s 至 %s 期间的工作内容。

**重要判断规则**：
- 如果提供的截图全部是黑屏、锁屏、空白屏幕，或者所有截图几乎完全相同（内容无明显变化），说明这段时间没有实际工作内容
- 此时请返回：{"summary": "暂无截屏内容", "activities": [], "app_usage": {}}

**正常分析要求**（仅当有明确工作内容时）：
1. 识别主要使用的应用程序（如 VS Code、浏览器、Office、微信等）。
2. 总结不同时间段的主要工作内容和活动类别（如：编程、文档编写、沟通、浏览等）。
3. 估算每个活动的大致时间占比（分钟）。
4. 特别要求：请将这段时间内的"工作要点总结"写入 summary 字段，并严格使用以下格式：
   - 用阿拉伯数字编号的条目，格式为："1.第一条内容;2.第二条内容;3.第三条内容;"。
   - 注意每一条后面必须以分号 ";" 结束，中间不要出现"本时段/该时段/在本时间段"等措辞，不要换行，不要使用中文括号或其他符号。

请严格按照以下 JSON 格式返回（不要包含任何其他文本）：
{
  "summary": "1.进行xx开发;2.解决了xx问题;3.查阅了xx文档;",
  "activities": [
    {
      "name": "编程开发",
      "duration_minutes": 45,
      "apps": ["VS Code", "Chrome"],
      "category": "开发"
    },
    {
      "name": "文档查阅",
      "duration_minutes": 15,
      "apps": ["Chrome"],
      "category": "学习"
    }
  ],
  "app_usage": {
    "VS Code": 40,
    "Chrome": 20
  }
}`,
		start.Format("15:04"),
		end.Format("15:04"),
	)
}

// AI 响应结构
type aiResponseData struct {
	Summary    string              `json:"summary"`
	Activities []activityData      `json:"activities"`
	AppUsage   map[string]int      `json:"app_usage"`
}

type activityData struct {
	Name            string   `json:"name"`
	DurationMinutes int      `json:"duration_minutes"`
	Apps            []string `json:"apps"`
	Category        string   `json:"category"`
}

// parseResponse 解析 AI 响应
func (a *Analyzer) parseResponse(response string, start, end time.Time) (*models.WorkSummary, error) {
	var data aiResponseData

	// 尝试提取 JSON（有些模型可能会在前后添加文本）
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		// 尝试提取 JSON 片段
		start := bytes.Index([]byte(response), []byte("{"))
		end := bytes.LastIndex([]byte(response), []byte("}"))
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				return nil, fmt.Errorf("failed to parse JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	// 转换为模型
	summary := &models.WorkSummary{
		StartTime:  start,
		EndTime:    end,
		Summary:    data.Summary,
		Activities: make([]models.Activity, len(data.Activities)),
		AppUsage:   data.AppUsage,
		CreatedAt:  time.Now(),
	}

	for i, act := range data.Activities {
		summary.Activities[i] = models.Activity{
			Name:            act.Name,
			DurationMinutes: act.DurationMinutes,
			Apps:            act.Apps,
			Category:        act.Category,
		}
	}

	return summary, nil
}

// TestConnection 测试 AI 连接并获取模型列表
func (a *Analyzer) TestConnection(provider, apiKey, baseURL string) ([]map[string]string, error) {
	var endpoint string

	// 确定端点
	if baseURL != "" {
		// 自定义 baseURL
		endpoint = baseURL
		if endpoint[len(endpoint)-1] != '/' {
			endpoint += "/"
		}
		endpoint += "models"
	} else {
		// 默认端点
		switch provider {
		case "openai":
			endpoint = "https://api.openai.com/v1/models"
		case "deepseek":
			endpoint = "https://api.deepseek.com/v1/models"
		case "qwen", "tongyi":
			endpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1/models"
		case "doubao":
			endpoint = "https://ark.cn-beijing.volces.com/api/v3/models"
		case "claude":
			// Claude 不提供标准的 models API，返回常用模型
			return []map[string]string{
				{"id": "claude-3-5-sonnet-20241022", "name": "Claude 3.5 Sonnet"},
				{"id": "claude-3-opus-20240229", "name": "Claude 3 Opus"},
				{"id": "claude-3-sonnet-20240229", "name": "Claude 3 Sonnet"},
				{"id": "claude-3-haiku-20240307", "name": "Claude 3 Haiku"},
			}, nil
		case "custom":
			if baseURL == "" {
				return nil, fmt.Errorf("自定义提供商需要指定 Base URL")
			}
			endpoint = baseURL
			if endpoint[len(endpoint)-1] != '/' {
				endpoint += "/"
			}
			endpoint += "models"
		default:
			return nil, fmt.Errorf("不支持的 AI 提供商: %s", provider)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(context.Background(), "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
		Object string `json:"object"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 转换为简化格式
	models := make([]map[string]string, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, map[string]string{
			"id":   m.ID,
			"name": m.ID, // 使用 ID 作为显示名称
		})
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("未找到可用模型")
	}

	return models, nil
}

// saveSummaryToFile 保存分析结果到Markdown文件
func (a *Analyzer) saveSummaryToFile(summary *models.WorkSummary) error {
	// 获取存储配置
	storageCfg := a.configMgr.GetStorage()

	// 创建summaries目录
	summariesDir := filepath.Join(storageCfg.DataDir, "summaries")
	if err := os.MkdirAll(summariesDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成文件名：summary_20250114_093045.md
	filename := fmt.Sprintf("summary_%s.md", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(summariesDir, filename)

	// 格式化为Markdown
	content := a.formatSummaryToMarkdown(summary)

	// 写入文件
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	logger.Info("分析结果已保存到: %s", filePath)
	return nil
}

// formatSummaryToMarkdown 格式化为Markdown
func (a *Analyzer) formatSummaryToMarkdown(summary *models.WorkSummary) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("# 工作分析报告\n\n")
	sb.WriteString(fmt.Sprintf("**分析时间**: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**工作时段**: %s - %s\n\n",
		summary.StartTime.Format("15:04"),
		summary.EndTime.Format("15:04")))

	// 总时长
	duration := summary.EndTime.Sub(summary.StartTime)
	sb.WriteString(fmt.Sprintf("**总时长**: %.0f 分钟\n\n", duration.Minutes()))

	// 分隔线
	sb.WriteString("---\n\n")

	// 工作总结
	sb.WriteString("## 📝 工作总结\n\n")
	sb.WriteString(summary.Summary)
	sb.WriteString("\n\n")

	// 活动详情
	if len(summary.Activities) > 0 {
		sb.WriteString("## 📊 活动详情\n\n")
		for i, activity := range summary.Activities {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, activity.Name))
			sb.WriteString(fmt.Sprintf("- **类别**: %s\n", activity.Category))
			sb.WriteString(fmt.Sprintf("- **时长**: %d 分钟\n", activity.DurationMinutes))
			if len(activity.Apps) > 0 {
				sb.WriteString(fmt.Sprintf("- **使用应用**: %s\n", strings.Join(activity.Apps, ", ")))
			}
			sb.WriteString("\n")
		}
	}

	// 应用使用统计
	if len(summary.AppUsage) > 0 {
		sb.WriteString("## 💻 应用使用统计\n\n")
		sb.WriteString("| 应用名称 | 使用时长 |\n")
		sb.WriteString("|---------|--------|\n")
		for app, minutes := range summary.AppUsage {
			sb.WriteString(fmt.Sprintf("| %s | %d 分钟 |\n", app, minutes))
		}
		sb.WriteString("\n")
	}

	// 底部信息
	sb.WriteString("---\n\n")
	sb.WriteString("*由 WorkTracker AI 自动生成*\n")

	return sb.String()
}

