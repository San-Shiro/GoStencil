@echo off
REM GoStencil WASM Build Script
REM Builds the WASM binary and copies it to the web directory.

echo Building GoStencil WASM...
set GOOS=js
set GOARCH=wasm
go build -o clients\wasm\web\gostencil.wasm .\clients\wasm\
set GOOS=
set GOARCH=

if %ERRORLEVEL% NEQ 0 (
    echo Build failed!
    exit /b 1
)

echo Done! Output: clients\wasm\web\gostencil.wasm
echo.
echo To serve locally:
echo   cd clients\wasm\web
echo   python -m http.server 8080
echo   Open http://localhost:8080
