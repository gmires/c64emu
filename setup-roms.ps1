# Script per scaricare le ROM C64 da VICE (distribuite legalmente sotto licenza GPL)
# VICE ha il permesso di distribuire le ROM originali Commodore 64

param(
    [switch]$Force
)

$RomsDir = "roms"
$BaseUrl = "https://raw.githubusercontent.com/libretro/vice-libretro/master/vice/data/C64"

$Roms = @(
    @{ File = "basic"; Url = "$BaseUrl/basic-901226-01.bin" },
    @{ File = "kernal"; Url = "$BaseUrl/kernal-901227-03.bin" },
    @{ File = "chargen"; Url = "$BaseUrl/chargen-901225-01.bin" }
)

Write-Host "=== C64 ROM Setup ===" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $RomsDir)) {
    New-Item -ItemType Directory -Path $RomsDir | Out-Null
    Write-Host "Creata cartella: $RomsDir" -ForegroundColor Green
}

foreach ($rom in $Roms) {
    $dest = Join-Path $RomsDir $rom.File
    
    if ((Test-Path $dest) -and -not $Force) {
        Write-Host "[SKIP] $($rom.File) - già presente" -ForegroundColor Yellow
        continue
    }
    
    Write-Host "[DOWNLOAD] $($rom.File) da VICE..." -ForegroundColor Cyan -NoNewline
    try {
        Invoke-WebRequest -Uri $rom.Url -OutFile $dest -UseBasicParsing -ErrorAction Stop
        $size = (Get-Item $dest).Length
        Write-Host " OK ($size bytes)" -ForegroundColor Green
    } catch {
        Write-Host " ERRORE: $_" -ForegroundColor Red
        Write-Host ""
        Write-Host "Non è stato possibile scaricare automaticamente le ROM." -ForegroundColor Yellow
        Write-Host "Puoi ottenerle manualmente da:" -ForegroundColor Yellow
        Write-Host "  1. VICE Emulator: https://sourceforge.net/projects/vice-emu/files/" -ForegroundColor White
        Write-Host "  2. Clonando il repo VICE: git clone https://github.com/libretro/vice-libretro.git" -ForegroundColor White
        Write-Host "     e copiando i file da vice/data/C64/ nella cartella roms/" -ForegroundColor White
        exit 1
    }
}

Write-Host ""
Write-Host "ROM C64 pronte! Puoi avviare l'emulatore:" -ForegroundColor Green
Write-Host "  .\c64emu.exe -gui" -ForegroundColor White
