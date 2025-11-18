@echo off
chcp 65001 >nul
echo ========================================
echo   WorkTracker AI - 手动构建脚本
echo   (适用于依赖下载失败的情况)
echo ========================================
echo.

echo 此脚本将手动克隆 screenshot 库并使用本地版本
echo.
pause

echo [1/6] 检查 Go 环境...
go version
if %errorlevel% neq 0 (
    echo ❌ Go 未安装或不在 PATH 中
    pause
    exit /b 1
)
echo ✅ Go 环境检查通过
echo.

echo [2/6] 检查 Git 环境...
git --version
if %errorlevel% neq 0 (
    echo ❌ Git 未安装
    echo 请从 https://git-scm.com/ 下载并安装 Git
    pause
    exit /b 1
)
echo ✅ Git 环境检查通过
echo.

echo [3/6] 创建 vendor 目录并克隆 screenshot 库...
if not exist "vendor" mkdir vendor
cd vendor

if exist "screenshot" (
    echo 目录已存在，删除旧版本...
    rmdir /s /q screenshot
)

echo 正在克隆 kbinani/screenshot...
git clone https://github.com/kbinani/screenshot.git screenshot
if %errorlevel% neq 0 (
    echo ❌ 克隆失败
    cd ..
    pause
    exit /b 1
)
cd ..
echo ✅ screenshot 库克隆完成
echo.

echo [4/6] 修改 go.mod 使用本地版本...
echo. >> go.mod
echo replace github.com/kbinani/screenshot =^> ./vendor/screenshot >> go.mod
echo ✅ go.mod 已更新
echo.

echo [5/6] 下载其他依赖...
go mod download
go mod tidy
echo ✅ 依赖下载完成
echo.

echo [6/6] 编译程序...
go build -ldflags="-H windowsgui" -o worktracker.exe cmd/worktracker/main.go
if %errorlevel% neq 0 (
    echo ❌ 编译失败
    pause
    exit /b 1
)
echo ✅ 编译完成
echo.

echo ========================================
echo   构建成功! 🎉
echo ========================================
echo.
echo 可执行文件: worktracker.exe
echo 运行方式: 双击 worktracker.exe 启动
echo.
echo 注意: 已使用本地克隆的 screenshot 库
echo.
pause
