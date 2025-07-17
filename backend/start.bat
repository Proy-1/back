@echo off
echo ========================================
echo   Go Backend - E-Commerce Dashboard
echo ========================================
echo.

echo 📦 Building Go application...
go build -o backend.exe .

if %errorlevel% neq 0 (
    echo ❌ Build failed! Please check for errors.
    pause
    exit /b %errorlevel%
)

echo ✅ Build successful!
echo.
echo 🚀 Starting backend server...
echo 💡 The server will run on http://localhost:5000
echo 💡 Press Ctrl+C to stop the server
echo.
echo 📋 Make sure MongoDB is running before starting!
echo.

backend.exe
