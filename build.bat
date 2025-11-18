@echo off
chcp 65001 >nul
echo ========================================
echo   WorkTracker AI - 构建脚本
echo ========================================
echo.

echo [1/5] 检查 Go 环境...
go version
if %errorlevel% neq 0 (
    echo ❌ Go 未安装或不在 PATH 中
    echo 请从 https://go.dev/dl/ 下载并安装 Go
    pause
    exit /b 1
)
echo ✅ Go 环境检查通过
echo.

echo [2/5] 清理旧的 vendor 目录...
if exist "vendor" (
    echo 发现旧的 vendor 目录，正在删除...
    rmdir /s /q vendor
    echo ✅ 已清理
) else (
    echo ✅ 无需清理
)
echo.

echo [3/5] 检查 Git 环境...
git --version >nul 2>&1
if %errorlevel% neq 0 (
    echo ⚠️ Git 未安装，将使用 go get 方式
    goto use_goget
)
echo ✅ Git 环境检查通过
echo.

echo [4/5] 克隆 screenshot 库到本地...
mkdir vendor 2>nul
cd vendor
git clone --depth=1 https://github.com/kbinani/screenshot.git screenshot
if %errorlevel% neq 0 (
    echo ⚠️ 克隆失败，尝试使用 go get
    cd ..
    rmdir /s /q vendor 2>nul
    goto use_goget
)
cd ..
echo ✅ screenshot 库克隆完成
echo.

echo 配置使用本地 screenshot...
findstr /C:"replace github.com/kbinani/screenshot" go.mod >nul
if %errorlevel% neq 0 (
    echo. >> go.mod
    echo replace github.com/kbinani/screenshot =^> ./vendor/screenshot >> go.mod
)
echo ✅ 已配置使用本地版本
echo.

goto build_app

:use_goget
echo.
echo [4/5] 使用 go get 获取 screenshot...
go get -u github.com/kbinani/screenshot@master
if %errorlevel% neq 0 (
    echo ❌ go get 失败
    echo.
    echo 💡 建议：
    echo 1. 安装 Git: https://git-scm.com/
    echo 2. 重新运行 build.bat
    pause
    exit /b 1
)
echo ✅ screenshot 库获取完成
echo.

:build_app
echo [5/5] 下载依赖并编译...
go mod download
go mod tidy

echo 生成 Windows 应用图标资源...
set "RSRC_OUT=cmd\worktracker\rsrc_windows.syso"
if exist "%RSRC_OUT%" del /f /q "%RSRC_OUT%" >nul 2>&1
go run github.com/akavel/rsrc@latest -ico asserts\WorkTraceAI.ico -o "%RSRC_OUT%"
if %errorlevel% neq 0 (
    echo ⚠️ 图标资源生成失败，将继续使用默认 exe 图标
) else (
    echo ✅ 已生成图标资源: %RSRC_OUT%
)

echo 编译程序（使用纯 Go SQLite 驱动，无需 CGO）...
set CGO_ENABLED=0
pushd cmd\worktracker >nul
go build -mod=mod -ldflags="-H windowsgui" -o ..\..\WorkTrackerAI.exe
if %errorlevel% neq 0 (
    popd >nul
    echo ❌ 编译失败
    pause
    exit /b 1
)
popd >nul
echo ✅ 编译完成
echo.

pause
