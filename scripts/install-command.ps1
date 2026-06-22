$ErrorActionPreference = 'Stop'

$projectDir = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$binaryName = 'radio.exe'
$userBin = Join-Path $HOME 'bin'
$destPath = Join-Path $userBin $binaryName

# Go kontrolü
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Hata: 'go' bulunamadi. Lutfen Go'yu yukleyin: https://go.dev/dl/"
    exit 1
}

# ffplay kontrolü
if (-not (Get-Command ffplay -ErrorAction SilentlyContinue)) {
    Write-Warning "Uyari: 'ffplay' bulunamadi. Oynatma icin ffmpeg gereklidir."
}

Write-Host "Binary derleniyor..."
Push-Location $projectDir
try {
    go build -o $binaryName ./cmd/radio
} finally {
    Pop-Location
}
$builtBinary = Join-Path $projectDir $binaryName
Write-Host "✅ Derleme tamamlandi: $builtBinary"

# Kurulum dizinini oluştur ve kopyala
New-Item -ItemType Directory -Force -Path $userBin | Out-Null
Copy-Item -Path $builtBinary -Destination $destPath -Force
Write-Host "✅ Komut kuruldu: $destPath"

# PATH kontrolü ve otomatik ekleme
$userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
$pathParts = $userPath -split ';' | Where-Object { $_ -ne '' }
if ($pathParts -notcontains $userBin) {
    $newPath = ($pathParts + $userBin) -join ';'
    [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
    Write-Host "✅ PATH'e eklendi: $userBin"
    Write-Host "Degisikligin gecerli olması icin yeni bir terminal acin."
} else {
    Write-Host "PATH zaten ayarli: $userBin"
}

Write-Host ""
Write-Host "Kurulum tamamlandi. Yeni terminalde 'radio' komutuyla calistirabilisiniz."
