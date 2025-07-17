@echo off
echo ========================================
echo   Go Backend - Development Setup
echo ========================================
echo.

echo 📦 Installing Go dependencies...
go mod tidy

if %errorlevel% neq 0 (
    echo ❌ Go mod tidy failed! Please check for errors.
    pause
    exit /b %errorlevel%
)

echo ✅ Dependencies installed successfully!
echo.
echo 🧪 Running tests...
go test -v

if %errorlevel% neq 0 (
    echo ⚠️  Some tests failed. Please check the output above.
) else (
    echo ✅ All tests passed!
)

echo.
echo 🚀 Development setup complete!
echo 💡 You can now run 'start.bat' to start the server
echo 💡 Make sure MongoDB is running first!
echo.
pause
