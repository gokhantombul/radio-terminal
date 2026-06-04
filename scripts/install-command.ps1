$ErrorActionPreference = 'Stop'

$projectDir = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$userBin = Join-Path $HOME 'bin'
$cmdPath = Join-Path $userBin 'radio.cmd'

New-Item -ItemType Directory -Force -Path $userBin | Out-Null

$cmdContent = @"
@echo off
setlocal
set "PROJECT_DIR=$projectDir"

cd /d "%PROJECT_DIR%"

if not exist "venv" (
  echo Python sanal ortami (venv) bulunamadi. Proje klasorunde su komutu calistirin:
  echo   python -m venv venv
  echo   .\venv\Scripts\pip install -r requirements.txt
  exit /b 1
)

where ffplay >nul 2>nul
if errorlevel 1 (
  echo ffplay bulunamadi. Lutfen ffmpeg yukleyin ve PATH'e ekleyin.
  exit /b 1
)

set "PYTHONPATH=%PROJECT_DIR%"
.\venv\Scripts\python.exe -m src.radio.main
"@

Set-Content -Path $cmdPath -Value $cmdContent -Encoding ascii

Write-Host "✅ Komut olusturuldu: $cmdPath"
Write-Host "PATH'e su klasoru ekleyin (Windows Environment Variables): $userBin"
Write-Host "Sonrasinda terminalde dogrudan 'radio' calisacaktir."
