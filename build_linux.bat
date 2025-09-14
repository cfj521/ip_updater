@echo off
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
"C:\Program Files\Go\bin\go.exe" build -ldflags "-s -w" -o build/ip_updater ./cmd/ip_updater
echo Build completed!