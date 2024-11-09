@echo off

:: Get current GOOS and GOARCH and store it to a variable
set GOOS_CURRENT=%GOOS%
set GOARCH_CURRENT=%GOARCH%

:: Build for Linux
set GOOS=linux
set GOARCH=amd64
go build -o outputLinux main.go

:: Reset
set GOOS=%GOOS_CURRENT%
set GOARCH=%GOARCH_CURRENT%