# <img src="https://s2.loli.net/2025/11/18/mrxTlJA9DyckEK1.png" alt="WorkTraceAI" style="zoom:3%;" />WorkTracker AI - 智能工作追踪分析工具

> 通过 AI 自动分析你的工作内容，生成精美的时间轴和活动报告
>
> 该项目受到dayflow项目的启发，但因为dayflow只能在mac上使用，因此使用AI变成工具制作一个windows版本，希望你会喜欢~

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?style=flat&logo=windows)](https://www.microsoft.com/windows)

---

## ✨ 功能特性

| 功能 | 说明 | 状态 |
|------|------|------|
| 🖥️ **自动截屏** | 后台定时截取屏幕（2-5秒间隔可配置） | ✅ |
| 🎯 **多屏幕支持** | 支持选择特定屏幕进行截图 | ✅ |
| 🤖 **AI 智能分析** | 使用大模型分析截图，自动总结工作内容 |   🕐  |
| 📊 **数据统计** | 应用使用时长、活动类型统计 | ✅ |
| ⚙️ **灵活配置** | 自定义工作时间、截图间隔、AI 模型等 | ✅ |
| 🌐 **Web 界面** | 通过浏览器访问和配置 | ✅ |
| 📱 **系统托盘** | Windows 系统托盘集成 | ✅ |
| 🔒 **隐私保护** | 所有数据本地存储，不上传云端，仅指本地部署大模型 | ✅ |


---

## 🎯 快速开始

### 📋 第一步：检查环境

确保已安装 Go 1.21 或更高版本：

```bash
go version
```

如未安装，请访问 👉 [Go 官网下载](https://go.dev/dl/)

### 🔨 第二步：构建项目

双击运行 `build.bat` 或在命令行执行：

```bash
build.bat
```

**💡 遇到 screenshot 依赖问题？** 我们提供了多种解决方案：

```bash
# 方案一：使用修复脚本（推荐）
fix-deps.bat

# 方案二：使用手动构建（100%成功）
build-manual.bat

# 方案三：设置国内代理
go env -w GOPROXY=https://goproxy.cn,direct
```


### 🚀 第三步：启动程序

双击运行生成的 `worktracker.exe`

程序会自动：
- ✅ 启动后台服务
- ✅ 在系统托盘显示图标
- ✅ 打开浏览器访问 `http://localhost:9527`

### ⚙️ 第四步：配置 AI

在 Web 界面中：
1. 选择 AI 提供商（OpenAI/Claude/DeepSeek/通义千问/豆包）
2. 填写 API 密钥
3. 点击"保存配置"

### ▶️ 第五步：开始使用

点击"开始截屏"按钮，程序将自动工作！

---

## 🏗️ 技术栈

### 后端技术
- **语言**: Go 1.21+
- **Web 框架**: Gin
- **数据库**: SQLite
- **任务调度**: Cron
- **截图**: kbinani/screenshot
- **系统托盘**: getlantern/systray

### 前端技术
- **核心**: 原生 HTML5 + CSS3 + JavaScript
- **设计**: 响应式布局 + 现代渐变风格
- **特点**: 轻量级、无依赖、快速加载

### AI 支持
- ✅ OpenAI (GPT-4o, GPT-4-Vision)
- ✅ DeepSeek (deepseek-chat, deepseek-vl)
- ✅ 通义千问 (qwen-vl-plus, qwen-vl-max)
- ✅ 豆包 (doubao-vision-pro)
- ✅ 本地部署 (ollama，支持baseurl调用即可)

---

## 📁 项目结构

```
WorkTracker/
├── 📂 cmd/worktracker/          # 主程序入口
├── 📂 internal/                 # 内部模块
│   ├── ai/                      # AI 分析器
│   ├── capture/                 # 截屏引擎
│   ├── config/                  # 配置管理
│   ├── scheduler/               # 任务调度
│   ├── server/                  # Web 服务器
│   ├── storage/                 # 数据存储
│   └── tray/                    # 系统托盘
├── 📂 pkg/                      # 公共包
│   ├── models/                  # 数据模型
│   └── utils/                   # 工具函数
├── 📂 web/                      # Web 资源
│   ├── templates/               # HTML 模板
│   └── static/                  # 静态文件
├── 📂 data/                     # 数据目录（运行时生成）
├── 🔧 build.bat                 # 构建脚本
├── 🔧 run.bat                   # 开发运行
└── 📄 go.mod                    # Go 模块定义
```

---

## 🎮 使用示例

### 系统托盘操作

右键点击托盘图标：
- 🌐 打开控制面板
- 🚪 退出程序

### Web 界面功能

- **状态监控**: 实时显示运行状态、今日截图数、今日分析数
- **一键控制**: 开始/停止截屏、立即截图、立即分析（会清空当天已有分析并自动重新分析当天已有截图）
- **配置管理**: 修改所有配置参数
- **查看总结**: 浏览每日 AI 生成的工作总结

---

## 🔧 开发指南

### 开发模式运行

```bash
# 方式一：使用脚本
run.bat

# 方式二：Go 命令
go run cmd/worktracker/main.go
```

### 添加新的 AI 提供商

1. 在 `internal/ai/analyzer.go` 中添加新方法
2. 实现 API 调用逻辑
3. 在 `callLLM` 方法中添加路由

### 修改前端界面

编辑 `web/templates/index.html` 文件即可

---

## ❓ 常见问题


<summary><b>Q: 为什么截图不工作？</b></summary>
<details>
检查：
1. 是否点击了"开始截屏"
2. 当前时间是否在工作时间内
3. 查看控制台是否有错误信息
</details>

<summary><b>Q: AI 分析失败怎么办？</b></summary>
<details>
可能原因：
1. API 密钥未配置或错误
2. 网络连接问题
3. API 额度不足
4. 时间段内没有截图
</details>

<summary><b>Q: 如何节省存储空间？</b></summary>
<details>
调整配置：
- 提高截图间隔（5-10秒）
- 降低图片质量（30-45）
- 减少数据保留天数（7-14天）
</details>
---

## 🔒 隐私说明

- ✅ **本地存储**: 所有截图和数据保存在本地
- ✅ **API 调用**: 仅在分析时发送截图到 AI 服务
- ✅ **无追踪**: 不收集任何使用数据
- ✅ **可控制**: 可随时停止或删除数据

---

## 📊 性能指标

| 指标 | 数值 |
|------|------|
| 内存占用 | ~50-100 MB |
| CPU 占用（待机） | < 1% |
| CPU 占用（截图） | ~5-10% |
| 磁盘占用 | ~500 MB/天 (3秒间隔) |

---

## 🛠️ 故障排除

### 构建失败

```bash
# 确认 Go 版本
go version

# 清理并重新下载依赖
go clean -modcache
go mod download
go mod tidy
```

### 端口被占用

修改配置文件 `data/config.json` 中的端口号

---

## 📝 更新日志

### v0.7.0 (2025-11-13)
- ✨ 新增 DeepSeek AI 支持（国产，性价比高）
- ✨ 新增通义千问支持（阿里云）
- ✨ 新增豆包支持（字节跳动）
- 🔧 修复依赖版本问题，确保构建成功
- 🔧 默认端口改为 9527（原8080）
- 📚 更新所有文档和配置说明

### v0.8.0 (2024-11-13)
- ✨ 新增base url，给本地host使用
- ✅ 完整的截屏和 AI 分析功能
- ✅ Web 控制面板
- ✅ 系统托盘集成
- ✅ 完善的文档系统

### v0.9.9 (2024-11-14)
- ✨ 新增配置保存功能
- ✨ 新增测试连接获取AI模型功能
- ✅ 完成截屏和 AI 分析功能测试
- ✅ 图标修改完成

### v1.0.0 (2024-11-17)
- ✅ 修复系统托盘图标加载问题
- ✅ 删除不必要的过程文件
- ✅ 增加打包功能，方便非开发人员使用
### v1.0.0 (2024-11-18)

- ✅ 修复系统托盘图标加载问题
- ✅ 修复立即分析按钮对应的固定写死时区的问题
- ✅ 正在分析中的等待提示
- ✨ 新增今日小结的一天总览功能
- ✅ 今日小结改为一个工作内容一个点，修改AI提示词，输出格式为1.；2.；3.；

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

---

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

感谢以下开源项目：
- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [screenshot](https://github.com/kbinani/screenshot) - 跨平台截图
- [systray](https://github.com/getlantern/systray) - 系统托盘
- [cron](https://github.com/robfig/cron) - 任务调度

---

<div align="center">

**🎉 祝你使用愉快！**

[快速开始](START_HERE.md) · [使用指南](USAGE_GUIDE.md) · [项目详情](PROJECT_COMPLETE.md)

Made with ❤️ by AI-assisted development

</div>
