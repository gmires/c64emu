#!/bin/bash
# Script per scaricare le ROM C64 da VICE (distribuite legalmente sotto licenza GPL)
# VICE ha il permesso di distribuire le ROM originali Commodore 64

set -e

ROMS_DIR="roms"
BASE_URL="https://raw.githubusercontent.com/libretro/vice-libretro/master/vice/data/C64"

BASIC_URL="$BASE_URL/basic-901226-01.bin"
KERNAL_URL="$BASE_URL/kernal-901227-03.bin"
CHARGEN_URL="$BASE_URL/chargen-901225-01.bin"

echo -e "\033[36m=== C64 ROM Setup ===\033[0m"
echo ""

mkdir -p "$ROMS_DIR"

download_rom() {
    local name="$1"
    local url="$2"
    local dest="$ROMS_DIR/$name"
    
    if [ -f "$dest" ] && [ "$1" != "--force" ]; then
        echo -e "\033[33m[SKIP] $name - già presente\033[0m"
        return
    fi
    
    echo -en "\033[36m[DOWNLOAD] $name da VICE...\033[0m "
    if curl -sfL "$url" -o "$dest"; then
        local size=$(stat -f%z "$dest" 2>/dev/null || stat -c%s "$dest" 2>/dev/null || echo "?")
        echo -e "\033[32mOK ($size bytes)\033[0m"
    else
        echo -e "\033[31mERRORE\033[0m"
        echo ""
        echo -e "\033[33mNon è stato possibile scaricare automaticamente le ROM.\033[0m"
        echo -e "\033[33mPuoi ottenerle manualmente da:\033[0m"
        echo -e "  1. VICE Emulator: https://sourceforge.net/projects/vice-emu/files/"
        echo -e "  2. Clonando il repo VICE: git clone https://github.com/libretro/vice-libretro.git"
        echo -e "     e copiando i file da vice/data/C64/ nella cartella roms/"
        exit 1
    fi
}

# Parse arguments
FORCE=false
for arg in "$@"; do
    case $arg in
        --force|-f)
            FORCE=true
            shift
            ;;
    esac
done

download_rom "basic" "$BASIC_URL"
download_rom "kernal" "$KERNAL_URL"
download_rom "chargen" "$CHARGEN_URL"

echo ""
echo -e "\033[32mROM C64 pronte! Puoi avviare l'emulatore:\033[0m"
echo -e "  ./c64emu -gui"
