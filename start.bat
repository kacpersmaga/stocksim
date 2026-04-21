@echo off
setlocal enabledelayedexpansion

if "%~1"=="" (
    echo Usage: start.bat ^<port^>
    exit /b 1
)

set PORT=%~1

echo Starting Stock Market Simulation on port %PORT%...

docker compose build --parallel
docker compose up -d

echo.
echo Service running at http://localhost:%PORT%
echo   Health: http://localhost:%PORT%/healthz
echo   Stocks: http://localhost:%PORT%/stocks
echo   Log:    http://localhost:%PORT%/log
echo.
echo To stop: docker compose down
