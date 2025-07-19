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

REM Check if using Atlas or local MongoDB
findstr /C:"mongodb+srv://" .env >nul
if %errorlevel% equ 0 (
    echo 🌐 Using MongoDB Atlas (Cloud Database)
    echo ✅ No need to start local MongoDB!
) else (
    echo 🏠 Using Local MongoDB
    echo ⚠️  Make sure MongoDB Compass is running or MongoDB service is started!
)

echo.
backend.exe
