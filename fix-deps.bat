@echo off
chcp 65001 >nul
echo ========================================
echo   快速修复 screenshot 依赖问题
echo ========================================
echo.

echo 正在设置 Go 代理为国内镜像...
go env -w GOPROXY=https://goproxy.cn,direct
echo ✅ Go 代理已设置
echo.

echo 正在清理缓存...
go clean -modcache
echo ✅ 缓存已清理
echo.

echo 正在获取 screenshot 库...
go get -v github.com/kbinani/screenshot@master
echo ✅ 获取完成
echo.

echo 正在更新依赖...
go mod tidy
echo ✅ 依赖已更新
echo.

echo ========================================
echo   修复完成! 现在可以运行 build.bat
echo ========================================
echo.
pause
