@echo off
REM 版本号管理构建脚本
REM 格式: 1.1.x (x自动递增)

SET MAJOR=1
SET MINOR=1

REM 读取当前构建号
IF NOT EXIST version.txt echo 1 > version.txt
SET /p BUILD=<version.txt

REM 显示当前版本
SET VERSION=%MAJOR%.%MINOR%.%BUILD%
echo Building IP-Updater v%VERSION%

REM 编译Linux版本
echo Compiling for Linux...
SET GOOS=linux
SET GOARCH=amd64
"C:\Program Files\Go\bin\go.exe" build -ldflags "-X main.Version=%VERSION%" -o build/ip_updater ./cmd/ip_updater

REM 检查编译是否成功
IF %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    exit /b 1
)

REM 编译成功，递增构建号
SET /a NEW_BUILD=%BUILD%+1
echo %NEW_BUILD% > version.txt

echo Build completed successfully!
echo Version: %VERSION%
echo Next build will be: %MAJOR%.%MINOR%.%NEW_BUILD%