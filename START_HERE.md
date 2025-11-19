# ✅ WorkTracker AI - 快速启动清单

## 🎯 第一次运行前的准备

### ☑️ 步骤 1: 验证环境

打开命令提示符，运行以下命令：

```bash
go version
```

✅ 如果显示 Go 版本（如 go1.21.x），表示环境正常
❌ 如果提示 "命令不存在"，请先安装 Go: https://go.dev/dl/

### ☑️ 步骤 2: 构建项目

双击运行 **`build.bat`** 文件

**🚨 如果遇到 screenshot 依赖下载失败：**

#### 快速解决方案（三选一）：

1. **使用修复脚本**（推荐）
   ```bash
   # 双击运行，会自动设置代理和修复依赖
   fix-deps.bat

   # 然后再运行
   build.bat
   ```

2. **使用手动构建脚本**
   ```bash
   # 会自动克隆 screenshot 库到本地
   build-manual.bat
   ```

3. **设置国内代理**（中国用户推荐）
   ```bash
   go env -w GOPROXY=https://goproxy.cn,direct
   build.bat
   ```

**📖 详细说明**: 查看 [BUILD_HELP.md](BUILD_HELP.md)

---

预期成功输出：
```
[1/4] 检查 Go 环境...
✅ Go 环境检查通过

[2/4] 获取 screenshot 库最新版本...
✅ screenshot 库获取完成

[3/4] 下载其他依赖...
✅ 依赖下载完成

[4/4] 编译程序...
✅ 编译完成

构建成功! 🎉
```

### ☑️ 步骤 3: 准备 API 密钥

根据你想使用的 AI 服务，准备好 API 密钥：

#### OpenAI
1. 访问: https://platform.openai.com/api-keys
2. 登录或注册账号
3. 创建新的 API 密钥
4. 复制密钥（格式: sk-...）
5. 推荐模型: `gpt-4o` 或 `gpt-4-vision-preview`

#### DeepSeek (国产，性价比高)
1. 访问: https://platform.deepseek.com/
2. 注册并获取 API 密钥
3. 推荐模型: `deepseek-chat` 或 `deepseek-vl`

#### 通义千问 (阿里云)
1. 访问: https://dashscope.aliyun.com/
2. 开通服务并获取 API Key
3. 推荐模型: `qwen-vl-plus` 或 `qwen-vl-max`

#### 豆包 (字节跳动)
1. 访问: https://www.volcengine.com/product/doubao
2. 申请并获取 API 密钥
3. 推荐模型: `doubao-vision-pro`


### ☑️ 步骤 4: 启动程序

双击运行 **`worktracker.exe`**

预期结果：
- ✅ 命令行窗口会显示启动信息
- ✅ 系统托盘出现 WorkTrackerAI 图标
- ✅ 浏览器自动打开 http://localhost:9527

### ☑️ 步骤 5: 配置 AI

在打开的 Web 界面中：

1. 滚动到 **"⚙️ 配置设置"** 部分
2. 填写以下关键配置：
   - **AI 提供商**: 选择你要使用的服务
     - `OpenAI` - 国际版，需科学上网
     - `DeepSeek` - 国产，性价比高，直连
     - `通义千问` - 阿里云，稳定可靠
     - `豆包` - 字节跳动，响应快速
   - **AI 模型**: 根据提供商填写模型名
     - OpenAI: `gpt-4o` 或 `gpt-4-vision-preview`
     - DeepSeek: `deepseek-chat` 或 `deepseek-vl`
     - 通义千问: `qwen-vl-plus` 或 `qwen-vl-max`
     - 豆包: `doubao-vision-pro`
   - **API 密钥**: 粘贴你的 API 密钥
3. 点击 **"💾 保存配置"** 按钮

### ☑️ 步骤 6: 开始监控

1. 点击 **"▶️ 开始截屏"** 按钮
2. 状态应该变为 **"运行中"**
3. 程序开始按照配置的间隔自动截图

---

## 🎉 恭喜! 设置完成!

### 接下来会发生什么？

1. **自动截图**: 程序每 3 秒（默认）截取一次屏幕
2. **AI 分析**: 每 60 分钟（默认）自动分析截图并生成工作总结
3. **查看结果**: 在 **"📝 今日工作总结"** 部分查看 AI 生成的分析

### 系统托盘操作

右键点击托盘图标可以：
- 🌐 打开控制面板
- 🚪 退出程序

---

## ⚠️ 常见启动问题

### 问题 1: 构建失败

**错误**: "go: github.com/kbinani/screenshot... invalid version"

**快速解决**:
```bash
# 方式 1: 使用修复脚本
fix-deps.bat

# 方式 2: 使用手动构建
build-manual.bat

# 方式 3: 设置代理后重试
go env -w GOPROXY=https://goproxy.cn,direct
build.bat
```

**完整说明**: 查看 [BUILD_HELP.md](BUILD_HELP.md)

### 问题 2: 端口已被占用

**错误**: "bind: address already in use"

**解决方案**:
1. 检查是否已有 WorkTracker 实例在运行
2. 或者修改配置文件中的端口号（默认 8080）

### 问题 3: AI 分析失败

**错误**: "API error: 401 Unauthorized"

**解决方案**:
1. 检查 API 密钥是否正确
2. 确认 API 密钥有效且有余额
3. 检查网络连接

### 问题 4: 截图没有保存

**检查清单**:
- ✅ 是否点击了 "开始截屏" 按钮？
- ✅ 当前时间是否在工作时间内？
- ✅ 查看 `data/screenshots/` 目录是否有文件

---

## 📞 需要帮助？

1. 查看详细文档: `USAGE_GUIDE.md`
2. 检查控制台输出是否有错误信息
3. 确认所有配置项都已正确填写

---

## 🚀 开发模式（可选）

如果你是开发者，想要修改代码并实时测试：

```bash
# 直接运行（无需编译）
run.bat

# 或者使用 Go 命令
go run cmd/worktracker/main.go
```

---

## 🎯 配置优化建议

### 日常使用
- 截图间隔: **3-5 秒**
- 分析间隔: **60 分钟**
- 图片质量: **75**

### 节省存储空间
- 截图间隔: **10 秒**
- 数据保留: **7-14 天**
- 图片质量: **60**

### 更精确分析
- 截图间隔: **2 秒**
- 分析间隔: **30 分钟**
- 图片质量: **85**

---

**🎉 祝你使用愉快!**
