# C64 Emulator

Un emulatore del **Commodore 64** scritto in **Go**, con rendering grafico in tempo reale, audio synthesis, input da tastiera/joystick, caricamento programmi PRG/D64/TAP/CRT e **schermo di boot integrato** che funziona anche senza ROM originali.

---

## Panoramica

Questo progetto emula l'hardware del Commodore 64 includendo:
- **CPU MOS 6510/6502** con tutti i 256 opcode (documentati + undocumented)
- **VIC-II** (Video Interface Chip) con tutte le modalità video: text, bitmap, multicolor, ECM, sprites
- **CIA 6526** (Complex Interface Adapter) con timer, tastiera, joystick e interrupt
- **SID 6581** (Sound Interface Device) con sintesi audio e filtri LP/BP/HP
- **Bus di sistema** con memory banking LORAM/HIRAM/CHAREN
- **Caricamento multi-formato**: PRG, D64 (disk), TAP (tape), CRT (cartridge)
- **Save states** per salvare e ripristinare lo stato completo della macchina
- **GUI** basata su Ebiten per l'esecuzione interattiva
- **Fallback boot screen** per test senza ROM

### Caratteristiche principali

| Feature | Stato | Note |
|---------|-------|------|
| CPU 6502 completo | ✅ | Tutti i 256 opcode (doc + undocumented) |
| VIC-II Text Mode + Sprites | ✅ | 384×272 PAL, border, scrolling |
| VIC-II Bitmap Mode | ✅ | Standard + Multicolor |
| VIC-II Multicolor Text | ✅ | MCM text rendering |
| VIC-II ECM | ✅ | Extended Color Mode |
| Sprite Multicolor + X/Y | ✅ | MC1/MC2 colors, expansion |
| SID Audio synthesis | ✅ | Waveform + filtri LP/BP/HP base |
| Memory Banking | ✅ | LORAM/HIRAM/CHAREN completo |
| Keyboard input | ✅ | Buffer KERNAL ($0277) + hardware matrix |
| Joystick input | ✅ | Emulato da tastiera (freccie + Ctrl destro) |
| GUI Window (Ebiten) | ✅ | 60 FPS, scala configurabile, fullscreen |
| PRG Loader | ✅ | Auto-run con RUN (BASIC) o JMP (ML) |
| D64 Loader | ✅ | List, extract, partial match, bulk load, KERNAL I/O hook |
| TAP Loader | ⚠️ | Basic PRG extraction |
| CRT Loader | ✅ | Cartridge ROM mapping |
| Save States | ✅ | Salvataggio/caricamento completo |
| Boot screen fallback | ✅ | Funziona senza ROM |
| ROM Validation | ✅ | Checksum + reset vector + chargen pattern |
| Debug Visualization Overlay | ✅ | Griglia, bordi, marker centro |
| Embedded Font Fallback | ✅ | Font PETSCII completo |
| KERNAL reale | ⚠️ | Boot completo ma main loop genera spurious errors (vedi Known Issues) |
| CIA Timer | ✅ | Cycle-accurate PAL/NTSC |
| NTSC Mode | ✅ | 262 linee, 65 cicli/linea |

---

## Requisiti

- **Go 1.21+** (consigliato 1.24+)
- **Ebiten v2** (installato automaticamente via `go mod tidy`)
- **ROM C64** (opzionali - l'emulatore funziona anche senza!)

---

## Compilazione

```bash
# Clona o naviga nel progetto
cd c64-emu

# Installa dipendenze
go mod tidy

# Compila l'emulatore
go build -o c64emu.exe ./cmd/emu    # Windows
go build -o c64emu ./cmd/emu        # Linux/macOS
```

---

## Utilizzo

### Modalità GUI (Finestra Interattiva)

Avvia l'emulatore con una finestra grafica in tempo reale:

```bash
# GUI base (scala 2x = 768x544, framebuffer PAL 384x272)
./c64emu -gui

# Scala maggiore
./c64emu -gui -scale 3      # 1152x816
./c64emu -gui -scale 4      # 1536x1088

# Fullscreen
./c64emu -gui -fullscreen

# Overlay debug per visualizzare griglia, bordi e area testo
./c64emu -gui -debug

# Carica un PRG e avvialo
./c64emu -gui -prg gioco.prg -autorun

# NTSC mode
./c64emu -gui -ntsc
```

**Mapping Tastiera PC → C64**

| Tasto PC | C64 | Note |
|----------|-----|------|
| `A-Z` | Lettere A-Z | Minuscole → maiuscole (C64 boota in uppercase mode) |
| `0-9` | Numeri 0-9 | |
| `Spazio` | SPACE | |
| `Invio` | RETURN | |
| `Backspace` | DELETE (BACKSPACE) | |
| `Shift` | SHIFT | Attiva caratteri secondari su alcuni tasti |
| `Esc` | RUN/STOP | |
| `F1-F8` | F1-F8 | Tasti funzione C64 |
| `,` | , | |
| `.` | . | |
| `/` | / | |
| `;` | ; | |
| `=` | = | |
| `-` | - | |
| `+` | + | Shift + tasto numerico |
| `*` | * | Shift + tasto numerico |
| `$` | $ | Shift + 4 |
| `%` | % | Shift + 5 |
| `&` | & | Shift + 6 |
| `(` | ( | Shift + 8 |
| `)` | ) | Shift + 9 |
| `@` | @ | |
| `:` | : | |

**Mapping Joystick 2 (emulato da tastiera)**

| Tasto PC | Azione Joystick |
|----------|----------------|
| `↑ Freccia su` | UP |
| `↓ Freccia giù` | DOWN |
| `← Freccia sinistra` | LEFT |
| `→ Freccia destra` | RIGHT |
| `Ctrl destro` | FIRE |

> **Nota:** In GUI mode la tastiera funziona tramite inserimento diretto nel buffer KERNAL (`$0277`), bypassando il problema degli interrupt CIA. Il joystick usa direttamente la porta hardware CIA ($DC00).

### Modalità Headless (Screenshot)

Esegui N frame e salva uno screenshot PNG:

```bash
# Pattern di test (senza ROM)
./c64emu -test -png screenshot.png

# Schermo di boot C64 (senza ROM valide)
./c64emu -frames 1 -png boot.png

# Debug overlay su screenshot
./c64emu -debug -frames 1 -png debug.png

# Carica PRG, esegui 60 frame, salva screenshot
./c64emu -prg demo.prg -autorun -frames 60 -png output.png

# Salva stato dopo esecuzione
./c64emu -frames 60 -save state.bin

# Carica stato e riparti
./c64emu -load state.bin -frames 1 -png loaded.png

# Loop infinito (Ctrl+C per terminare)
./c64emu -frames 0
```

### Caricamento ROM

Le ROM originali Commodore 64 sono **protette da copyright** e non sono incluse in questo repository. Tuttavia, possono essere ottenute legalmente attraverso le seguenti fonti:

#### Opzione 1: Script automatico (consigliata)

Lo script scarica le ROM dal repository di VICE (distribuite legalmente sotto licenza GPL):

**Windows (PowerShell):**
```powershell
.\setup-roms.ps1
```

**Linux/macOS (Bash):**
```bash
chmod +x setup-roms.sh
./setup-roms.sh
```

#### Opzione 2: VICE Emulator

Scarica VICE da [sourceforge.net/projects/vice-emu](https://sourceforge.net/projects/vice-emu/files/) e copia i file dalla cartella `data/C64/`:

```bash
mkdir -p roms
cp /path/to/vice/data/C64/basic-901226-01.bin  roms/basic
cp /path/to/vice/data/C64/kernal-901227-03.bin roms/kernal
cp /path/to/vice/data/C64/chargen-901225-01.bin roms/chargen
```

#### Opzione 3: Clona VICE da GitHub

```bash
git clone https://github.com/libretro/vice-libretro.git
cp vice-libretro/vice/data/C64/*.bin roms/
# Rinomina i file:
mv roms/basic-901226-01.bin roms/basic
mv roms/kernal-901227-03.bin roms/kernal
mv roms/chargen-901225-01.bin roms/chargen
```

#### Struttura attesa

```
roms/
  basic      → ROM BASIC (8KB, $A000-$BFFF)
  kernal     → ROM KERNAL (8KB, $E000-$FFFF)
  chargen    → ROM Character (4KB, $D000-$DFFF)
```

**Validazione automatica:** L'emulatore verifica se le ROM sono valide (checksum + reset vector). Se invalide, mostra automaticamente lo **schermo di boot fallback**.

```bash
# Forza boot screen fallback anche con ROM valide
./c64emu -fallback -gui
```

### Caricamento programmi

```bash
# Carica PRG diretto
./c64emu -prg gioco.prg -autorun -gui

# Lista file in un D64
./c64emu -d64 disco.d64

# Carica file da D64 (nome parziale, case-insensitive)
./c64emu -d64 disco.d64 -d64file "GIOCO" -autorun -gui
# "Pyramid" trova automaticamente "100.000 PYRAMID"

# Carica TUTTI i file PRG da D64 in memoria (utile per giochi multi-file)
./c64emu -d64 disco.d64 -d64bulk -autorun -gui

# Giochi multi-file che caricano durante l'esecuzione (hook KERNAL I/O)
# Il D64 viene montato e le chiamate KERNAL LOAD/OPEN/CHRIN vengono
# intercettate per caricare file dal disco automaticamente.
./c64emu -d64 disco.d64 -d64file "GIOCO" -autorun -gui

# Carica da TAP
./c64emu -tap gioco.tap -autorun -gui

# Carica cartuccia CRT
./c64emu -crt gioco.crt -gui
```

---

## Schermo di Boot Fallback

Quando le ROM C64 non sono presenti o sono invalide, l'emulatore mostra automaticamente uno schermo di boot centrato con testo leggibile:

```
**** COMMODORE 64 BASIC V2 ****

64K RAM SYSTEM  38911 BASIC BYTES FREE

READY.
```

Con colori corretti C64 (bordo azzurro, sfondo blu, testo bianco) e font embedded completo.

---

## Architettura

### Componenti Principali

```
+--------------------------------------------------+
|                  Machine (main.go)                |
+--------------------------------------------------+
|  +--------+  +--------+  +--------+  +--------+  |
|  | CPU    |  | VIC-II |  | CIA #1 |  | CIA #2 |  |
|  | 6510   |  | Video  |  | I/O    |  | Serial |  |
|  +--------+  +--------+  +--------+  +--------+  |
|  +--------+  +--------------------------------+  |
|  | SID    |  | Bus (Memory + I/O Mapping)     |  |
|  | Audio  |  +--------------------------------+  |
|  +--------+                                      |
+--------------------------------------------------+
```

### File del Progetto

| File | Descrizione |
|------|-------------|
| `cpu.go` | CPU MOS 6510/6502 - 256 opcode completi |
| `bus.go` | Bus di sistema con memory banking e I/O routing |
| `vic2.go` | VIC-II - controller video, sprites, interrupt |
| `raster.go` | Renderer grafico raster (tutte le modalità video) |
| `cia.go` | CIA 6526 - timer, keyboard matrix 8x8, joystick |
| `sid.go` | SID 6581 - audio synthesis, waveform, filtri |
| `audio.go` | Sistema audio Ebiten per output continuo |
| `display.go` | GUI Ebiten - finestra e rendering |
| `input.go` | Mappatura tastiera e joystick |
| `config.go` | Costanti di sistema e configurazione |
| `main.go` | Machine coordinator |
| `savestate.go` | Serializzazione/deserializzazione stato |
| `embedded_font.go` | Font PETSCII embedded per fallback |
| `chargen_validator.go` | Validatore ROM chargen con pattern esatti |
| `debug_overlay.go` | Overlay visivo per debug del rendering |
| `d64.go` | Lettore immagini disco D64 |
| `tap.go` | Lettore immagini nastro TAP |
| `crt.go` | Lettore cartucce CRT |
| `cmd/emu/main.go` | Entry point CLI |

---

## Componenti Implementati

### CPU 6510 ✅
- [x] Tutti gli opcodes 6502 documentati (151 opcode)
- [x] Tutti gli opcode undocumented (105 opcode): SLO, RLA, SRE, RRA, SAX, LAX, DCP, ISC, NOP variants, JAM
- [x] Addressing modes: imm, zp, zpX, zpY, abs, absX, absY, (ind,X), (ind),Y
- [x] Flag N, Z, C, V, I, D, B correttamente gestiti
- [x] ADC/SBC con modalità decimale (BCD)
- [x] Branch con page-crossing cycles + fix fetch operand quando non preso
- [x] Stack operations (PHA, PLA, PHP, PLP, JSR, RTS, RTI)
- [x] Memory banking via porta $01 (LORAM/HIRAM/CHAREN)

### VIC-II ✅
- [x] Framebuffer PAL completo 384×272 (con border 32+36 pixel)
- [x] Framebuffer NTSC 384×272
- [x] Raster line tracking (312 linee PAL, 262 NTSC)
- [x] Standard Text Mode (40x25 caratteri, 8x8 pixel)
- [x] Multicolor Text Mode
- [x] Standard Bitmap Mode
- [x] Multicolor Bitmap Mode
- [x] Extended Color Mode (ECM)
- [x] Sprite rendering (8 sprite hardware)
- [x] Sprite Multicolor + X/Y expansion
- [x] Collision detection (sprite-sprite, sprite-background)
- [x] Border color e background colors
- [x] Smooth scrolling (X/Y offset)
- [x] Memory pointers ($D018) — screen e char base
- [x] Character ROM mapping ($1000-$1FFF, $9000-$9FFF)
- [x] Interrupt raster (IRQ)
- [x] Read registers: $D011, $D012, $D019, $D01A
- [x] Color RAM support ($D800-$DBFF) — sempre accessibile al VIC-II
- [x] Embedded font fallback quando chargen è invalido

### CIA 6526 ✅
- [x] Timer A e Timer B
- [x] One-shot mode
- [x] Keyboard matrix 8x8 completa
- [x] Joystick emulation (PORT A, bit=0 = pressed)
- [x] Interrupt handling
- [x] Port A/Port B read con pull-up e merge joystick
- [x] ICR read clears flags (real 6526 behavior)
- [x] **Fix cycle-accurate**: runFrame conta cicli totali invece di istruzioni

### SID 6581 ✅
- [x] Waveform: triangle, sawtooth, pulse, noise
- [x] Envelope generator (attack, decay, sustain, release)
- [x] Gate control
- [x] Volume master
- [x] Filtri LP, BP, HP base
- [x] Resonance
- [x] Voice routing attraverso filtro

### Bus di Sistema ✅
- [x] 64KB RAM
- [x] Memory banking LORAM/HIRAM/CHAREN
- [x] ROM mapping: BASIC ($A000), KERNAL ($E000), CHARGEN ($D000)
- [x] I/O mapping: VIC-II, SID, CIA#1, CIA#2
- [x] Color RAM ($D800-$DBFF) — sempre accessibile al VIC-II
- [x] CPU Port $01 (memory configuration)
- [x] Write-through RAM per area I/O ($D000-$DFFF)

### Save States ✅
- [x] Serializzazione completa stato (CPU, RAM, I/O, VIC-II, CIAs, SID)
- [x] Flag CLI `-save FILE` e `-load FILE`
- [x] Versioning del formato

---

## Stato Emulazione KERNAL Reale

L'emulatore riconosce, carica ed esegue correttamente le ROM C64 (BASIC, KERNAL, CHARGEN):

- ✅ ROM rilevate e validate (checksum + reset vector $FCE2)
- ✅ CPU esegue codice KERNAL dalla ROM reale
- ✅ RAM inizializzata con pattern VICE (deterministico)
- ✅ CPU port $01 con pull-up corretti (DDR=0 → input=1)
- ✅ CIA timer e interrupt funzionanti (cycle-accurate)
- ✅ Opcode undocumented implementati (SLO, RLA, SRE, RRA, SAX, LAX, DCP, ISC, NOP variants)
- ✅ Branch bug fix (fetch operand anche quando branch non preso)
- ✅ CIA interrupt fix (timer underflow always sets IRQ flag)
- ✅ VIC-II `$D018` memory pointers (screen/char base)
- ✅ Character ROM mapping per VIC-II ($1000-$1FFF / $9000-$9FFF)
- ✅ Color RAM sempre accessibile al VIC-II indipendentemente da CHAREN
- ⚠️ **KERNAL boot completo ma con bug post-boot** (vedi Known Issues sotto)
- ✅ Schermo di boot fallback disponibile per test senza ROM

### Known Issues

**Cold start bug — `?OUT OF DATA ERROR IN 0` loop**
Il KERNAL ROM completa correttamente RAMTAS → CINT → IOINIT → BASIC startup, ma il main loop del BASIC stampa ripetutamente `?OUT OF DATA ERROR IN 0` invece di `READY.`.

**Causa identificata:** Il vettore `$0300`/`$0301` (usato dall'IRQ handler e dal warm start) non viene inizializzato correttamente durante il cold start. Il BASIC warm start a `$E394` chiama `$E453` che *dovrebbe* copiare i vettori a `$0300-$030B`, ma il valore letto da `$0300`/`$0301` rimane `$0000`. Quando l'IRQ handler del KERNAL salta a `($0300)`, il CPU finisce a `$0000` (BRK → interrupt loop → warm start → JMP `$0000`...). Questo corrompe lo zero-page, incluso il flag errore `$79` che viene sovrascritto con `$AD` (opcode LDA assoluto copiato da una tabella del KERNAL). Da quel momento il BASIC main loop vede un errore pendente e stampa il messaggio ripetutamente.

**Impatto:**
- Giochi/carichi da PRG/D64/TAP **non funzionano** con le ROM reali (il cold start fallisce silenziosamente)
- Lo schermo di boot **fallback** (senza ROM) funziona perfettamente
- I save states caricati da una macchina già avviata funzionerebbero, ma non possiamo generarli con questo bug

**Workaround attuale:** Nessuno. Serve un fix nel percorso di cold start o nell'inizializzazione dei vettori `$0300-$030B`.

**Per usare l'emulatore:**
```bash
# Con ROM reali (consigliato)
./c64emu -gui

# Boot screen fallback (senza ROM)
./c64emu -fallback -gui
```

---

## Formati Supportati

| Formato | Estensione | Supporto | Note |
|---------|-----------|----------|------|
| PRG | `.prg` | ✅ | Byte 0-1: indirizzo, poi codice |
| ROM | `.bin` | ✅ | RAW binary per basic/kernal/chargen |
| PNG | `.png` | ✅ Esporta | Screenshot framebuffer |
| D64 | `.d64` | ✅ | Disk image: list files, extract PRG |
| TAP | `.tap` | ⚠️ | Datasette: basic PRG extraction |
| CRT | `.crt` | ✅ | Cartridge: ROM mapping into memory |
| State | `.bin` | ✅ | Save state completo |

---

## ROM C64 - Informazioni

### Checksum ROM Standard

| ROM | CRC32 Originale | CRC32 VICE |
|-----|----------------|------------|
| BASIC | `9A0B6A3A` | `8D5FB9E6` |
| KERNAL | `3B41D573` | `6B1D8E93` |
| CHARGEN | `66C90C0C` | `8C6D0065` |

### Reset Vector KERNAL
- **Valido**: `$FCE2` (standard C64)
- **Invalido**: qualsiasi altro valore (es. `$FE43` = ROM di altro computer)

### Validazione CHARGEN
Il validatore verifica pattern esatti dei caratteri nel ROM chargen:
- **Space (0x20)**: deve essere tutto zero
- **'A' (0x41)**: pattern classico C64 `{0x18, 0x3C, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00}`
- **'@' (0x00)**: pattern classico C64 `{0x3C, 0x66, 0x6E, 0x76, 0x66, 0x66, 0x3C, 0x00}`

Se la ROM chargen non supera la validazione, l'emulatore usa automaticamente un **font embedded** completo per il rendering del testo.

### Fonti ROM Legal

1. **Script `setup-roms.ps1` / `setup-roms.sh`** — Scarica automaticamente le ROM dal repository VICE (GPL)
2. **VICE Emulator** ([vice-emu.sourceforge.io](https://vice-emu.sourceforge.io)) — ROM incluse nelle release ufficiali
3. **GitHub libretro/vice-libretro** — [github.com/libretro/vice-libretro](https://github.com/libretro/vice-libretro) — ROM nel branch `master` sotto `vice/data/C64/`
4. **Estrazione da C64 reale** che possiedi personalmente

> **Nota legale:** Le ROM originali Commodore 64 sono protette da copyright. Questo progetto non le distribuisce. Gli script forniti puntano al repository VICE che ha il permesso legale di distribuirle sotto licenza GPL.

---

## Esempi d'Uso

### Test Rapido (senza ROM)
```bash
# Schermo di boot C64
./c64emu -frames 1 -png boot.png

# Pattern di test
./c64emu -test -png test.png
```

### GUI Interattiva
```bash
# Boot screen
./c64emu -gui

# Scala 3x
./c64emu -gui -scale 3

# Fullscreen
./c64emu -gui -fullscreen

# Con PRG
./c64emu -gui -prg gioco.prg -autorun
```

### Con ROM Valide
```bash
# Crea cartella roms/ e copia i file
mkdir roms
cp basic.rom roms/basic
cp kernal.rom roms/kernal
cp chargen.rom roms/chargen

# Avvia (validazione automatica)
./c64emu -gui
```

### Caricare un Gioco
```bash
# Carica PRG e avvia in GUI
./c64emu -gui -prm giochi/pacman.prg -autorun

# Carica da D64
./c64emu -gui -d64 giochi/disco.d64 -d64file "PACMAN" -autorun

# Carica e fai screenshot
./c64emu -prg giochi/pacman.prg -autorun -frames 120 -png pacman.png

# Salva stato durante il gioco
./c64emu -gui -prg giochi/rpg.prg -autorun -save rpg_save.bin
```

---

## Palette Colori PAL

L'emulatore utilizza la palette standard C64 PAL:

| Idx | Colore | Hex | Nome |
|-----|--------|-----|------|
| 0 | Nero | #000000 | Black |
| 1 | Bianco | #FFFFFF | White |
| 2 | Rosso | #880000 | Red |
| 3 | Cyan | #AAFF66 | Cyan |
| 4 | Viola | #CC44CC | Purple |
| 5 | Verde | #44AA44 | Green |
| 6 | Blu | #0000AA | Blue |
| 7 | Giallo | #EEEE77 | Yellow |
| 8 | Arancione | #DD8855 | Orange |
| 9 | Marrone | #0066DD | Brown |
| 10 | Rosa | #CC6688 | Light Red |
| 11 | Grigio Sc. | #333333 | Dark Grey |
| 12 | Grigio Med | #777777 | Grey |
| 13 | Verde Chi. | #AAFF00 | Light Green |
| 14 | Azzurro | #0088FF | Light Blue |
| 15 | Grigio Chi | #BBBBBB | Light Grey |

---

## Note Tecniche

### Clock System
- **PAL**: 985248 Hz, 63 cicli/linea, 312 linee/frame
- **NTSC**: 1022737 Hz, 65 cicli/linea, 262 linee/frame

### Framebuffer PAL
- **Dimensione totale**: 384×272 pixel
- **Area testo visibile**: 320×200 pixel (centrata)
- **Border sinistro**: 32 pixel
- **Border destro**: 32 pixel
- **Border superiore**: 36 pixel
- **Border inferiore**: 36 pixel

### Memory Map
```
$0000-$9FFF  RAM (40KB)
$A000-$BFFF  ROM BASIC (8KB, se LORAM=1)
$C000-$CFFF  RAM (4KB)
$D000-$DFFF  I/O / CHARGEN / RAM (banking)
$E000-$FFFF  ROM KERNAL (8KB, se HIRAM=1)
```

### CPU Port $01
| Bit | Funzione |
|-----|----------|
| 0 | LORAM (BASIC ROM) |
| 1 | HIRAM (KERNAL ROM) |
| 2 | CHAREN (I/O vs CHARGEN) |
| 3-5 | Datasette motor / LED |
| 6-7 | Non usati (sempre 1) |

### Color RAM
- **Indirizzo**: $D800-$DBFF
- **Dimensione**: 1024 bytes (1 nibble effettivo, 4 bit colore)
- **Uso**: Determina il colore dei caratteri in text mode
- **Accesso**: Sempre accessibile al VIC-II indipendentemente dalla configurazione CHAREN

---

## CLI Flags

```
-gui              Esegui in modalità finestra
-scale N          Fattore di scala per GUI (default: 2.0)
-fullscreen       Modalità schermo intero
-ntsc             Usa timing NTSC (default: PAL)
-test             Inietta pattern di test
-prg FILE         Carica file PRG
-d64 FILE         Carica immagine disco D64
-d64file NAME     File da caricare da D64 (supporta partial match)
-d64bulk          Carica tutti i file PRG dal D64 in memoria
-tap FILE         Carica immagine nastro TAP
-crt FILE         Carica cartuccia CRT
-autorun          Auto-esegui PRG caricato
-frames N         Numero frame da eseguire (0 = infinito)
-png FILE         File output screenshot (default: screenshot.png)
-save FILE        Salva stato a file dopo esecuzione
-load FILE        Carica stato da file prima esecuzione
-fallback         Forza schermo di boot fallback
-debug            Abilita overlay visivo (griglia, bordi, marker)
```

---

## Componenti Mancanti / Futuri

### Priorità Alta
- [ ] **Fix cold start bug** — Il main loop del BASIC stampa `?OUT OF DATA ERROR IN 0` dopo il boot. Causa: vettore `$0300` non inizializzato correttamente (vedi Known Issues)
- [x] **Fix CIA timer decrement** — ✅ Risolto: `runFrame()` contava istruzioni invece di cicli totali
- [x] **SID Filtri** (Low-pass, Band-pass, High-pass) — ✅ Base implementata
- [x] **Proper IRQ timing** — ✅ Timer underflow a ~60Hz stabili

### Priorità Media
- [ ] **Raster interrupts** avanzati — timing preciso per split-screen
- [x] **Audio output** su device audio reale — ✅ Base implementata via Ebiten stream
- [x] **Fullscreen mode** — ✅ Supportato
- [x] **KERNAL I/O hook** — ✅ Intercetta LOAD/OPEN/CHKIN/CHRIN per caricare file dal D64 montato. **Non utilizzabile** finché il cold start bug non è risolto (il main loop BASIC non raggiunge i programmi caricati)

### Priorità Bassa
- [ ] **REU** (RAM Expansion Unit)
- [ ] **Userport** emulation
- [ ] **Serial bus** completo
- [x] **Save states** (salvataggio/caricamento stato) — ✅ Implementato
- [ ] **Debugger** (step-by-step, breakpoints, memory dump)
- [ ] **Assembler integrato**
- [x] **NTSC mode** — ✅ Supportato (262 linee, 65 cicli/linea)

---

## Licenza

Progetto educativo. Le ROM originali Commodore 64 sono protette da copyright e devono essere ottenute legalmente (estrattore da hardware proprio, o alternative open source come OpenC64ROMs).

---

## Contributi

Questo è un progetto in evoluzione. Le aree in cui si può contribuire:
1. Implementazione Bitmap Mode (completata)
2. Emulazione 1541 Disk Drive completa
3. Supporto formato TAP/D64 avanzato
4. Debugger interattivo
5. Ottimizzazione performance
6. Test suite automatizzata

---

## Contatti

Per bug report, feature request o contributi, fare riferimento al repository GitHub.

---

**Enjoy your C64!** 🕹️
