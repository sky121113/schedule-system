@echo off
echo ====================================
echo Schedule System - Frontend Launcher
echo ====================================
echo.
echo Checking if node_modules exists...

cd frontend

if not exist "node_modules" (
    echo node_modules not found. Installing dependencies...
    echo This may take a few minutes...
    echo.
    call npm install
    echo.
    echo Dependencies installed successfully!
    echo.
)

echo Starting frontend development server...
echo Frontend will run at http://localhost:3000
echo.
echo Press Ctrl+C to stop the server
echo ====================================
echo.

call npm run dev
