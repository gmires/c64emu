# Documentazione di Riferimento - C64 Emulator

## Collegamenti Utili

### Opcode 6502/6510
- **Matrice completa opcode (documentati + undocumented)**: https://www.oxyron.de/html/opcodes02.html
- **Lista opcode con flags e cicli**: Vedi tabella sotto

### KERNAL C64
- **Funzioni KERNAL**: https://www.c64-wiki.com/wiki/Kernal
- **KERNAL ROM Listing dettagliato**: http://unusedino.de/ec64/technical/aay/c64/krnromma.htm
- **Jump table KERNAL**: $FF81-$FFF3

### CIA 6526
- **Registri CIA dettagliati**: https://www.c64-wiki.com/wiki/CIA
- **Data sheet MOS 6526 (PDF)**: http://www.6502.org/documents/datasheets/mos/mos_6526_cia.pdf

### VIC-II
- **Registri VIC-II**: https://www.c64-wiki.com/wiki/VIC
- **Memory mapping VIC**: $D000-$D3FF

### Memory Banking
- **CPU Port $01**: https://www.c64-wiki.com/wiki/6510
- **C64 Memory Map**: https://www.c64-wiki.com/wiki/Memory_Map

---

## Opcode Undocumented Implementati

| Opcode | Nome | Descrizione | Addressing | Cycles |
|--------|------|-------------|------------|--------|
| $03 | SLO | ASL + ORA | (zp,X) | 8 |
| $0F | SLO | ASL + ORA | abs | 6 |
| $3C | NOP | No Operation | abs,X | 4 |
| $E7 | ISC | INC + SBC | zp | 5 |
| $F4 | NOP | No Operation | zp,X | 4 |

### Opcode da implementare in futuro

Dalla matrice completa, opcode undocumented comuni nel KERNAL:
- $07, $17, $27, $37, $47, $57, $67, $77, $87, $97, $A7, $B7, $C7, $D7, $F7 (SLO, RLA, SRE, RRA, SAX, LAX, DCP, ISC varianti)
- $1B, $3B, $5B, $7B, $DB, $FB (ASS varianti)
- $93, $9F, $9C, $9E (SHX, SHY, AHX, TAS)

---

## Funzioni KERNAL Principali

| Nome | Indirizzo | Funzione |
|------|-----------|----------|
| CINT | $FF81 | Inizializza screen editor e VIC-II |
| IOINIT | $FF84 | Inizializza I/O devices |
| RAMTAS | $FF87 | RAM test |
| RESTOR | $FF8A | Set top of RAM |
| VECTOR | $FF8D | Manage RAM vectors |
| SETMSG | $FF90 | Set system message output |
| SECOND | $FF93 | Send secondary address for LISTEN |
| TKSA | $FF96 | Send secondary address to TALK |
| MEMTOP | $FF99 | Set top of RAM |
| MEMBOT | $FF9C | Set bottom of memory |
| SCNKEY | $FF9F | Scan keyboard |
| SETTMO | $FFA2 | Set timeout flag |
| ACPTR | $FFA5 | Input byte from serial port |
| CIOUT | $FFA8 | Transmit byte over serial bus |
| UNTLK | $FFAB | Send UNTALK command |
| UNLSN | $FFAE | Send UNLISTEN command |
| LISTEN | $FFB1 | Command device to listen |
| TALK | $FFB4 | Command device to talk |
| READST | $FFB7 | Read status word |
| SETLFS | $FFBA | Set up logical file |
| SETNAM | $FFBD | Set up file name |
| OPEN | $FFC0 | Open logical file |
| CLOSE | $FFC3 | Close logical file |
| CHKIN | $FFC6 | Open channel for input |
| CHKOUT | $FFC9 | Open channel for output |
| CLRCHN | $FFCC | Clear I/O channels |
| CHRIN | $FFCF | Get character from input |
| CHROUT | $FFD2 | Output character |
| LOAD | $FFD5 | Load RAM from device |
| SAVE | $FFD8 | Save memory to device |
| SETTIM | $FFDB | Set system clock |
| RDTIM | $FFDE | Read system clock |
| STOP | $FFE1 | Check STOP key |
| GETIN | $FFE4 | Get character |
| CLALL | $FFE7 | Close all files |
| UDTIM | $FFEA | Update system clock |
| SCREEN | $FFED | Return screen format |
| PLOT | $FFF0 | Set/retrieve cursor location |
| IOBASE | $FFF3 | Define I/O memory page |

---

## Registri CIA 6526

### CIA #1 ($DC00-$DC0F)

| Reg | Indirizzo | Nome | Funzione |
|-----|-----------|------|----------|
| 0 | $DC00 | PRA | Data Port A (keyboard cols, joystick 2) |
| 1 | $DC01 | PRB | Data Port B (keyboard rows, joystick 1) |
| 2 | $DC02 | DDRA | Data Direction Port A |
| 3 | $DC03 | DDRB | Data Direction Port B |
| 4 | $DC04 | TA LO | Timer A Low Byte |
| 5 | $DC05 | TA HI | Timer A High Byte |
| 6 | $DC06 | TB LO | Timer B Low Byte |
| 7 | $DC07 | TB HI | Timer B High Byte |
| 8 | $DC08 | TOD 10THS | Time of Day 1/10s |
| 9 | $DC09 | TOD SEC | Time of Day Seconds |
| 10 | $DC0A | TOD MIN | Time of Day Minutes |
| 11 | $DC0B | TOD HR | Time of Day Hours |
| 12 | $DC0C | SDR | Serial Shift Register |
| 13 | $DC0D | ICR | Interrupt Control and Status |
| 14 | $DC0E | CRA | Control Timer A |
| 15 | $DC0F | CRB | Control Timer B |

### Control Timer A ($DC0E)

| Bit | Funzione |
|-----|----------|
| 0 | Start/Stop timer (1=start) |
| 1 | Timer output to PB6 |
| 2 | Toggle/Pulse mode for PB6 |
| 3 | One-shot mode (1=stop after underflow) |
| 4 | Force load latch into timer |
| 5 | Count source (0=system cycles, 1=CNT pin) |
| 6 | Serial shift register direction |
| 7 | TOD frequency (0=60Hz, 1=50Hz) |

### Interrupt Control ($DC0D)

**Lettura:**
- Bit 0: Timer A underflow
- Bit 1: Timer B underflow
- Bit 2: TOD alarm
- Bit 3: Serial shift register complete
- Bit 4: FLAG pin (cassette/serial)
- Bit 7: IRQ occurred

**Scrittura:**
- Bit 7 = 1: Set mask bits
- Bit 7 = 0: Clear mask bits

---

## Problemi Noti e Soluzioni

### 1. Opcode Undocumented
Il KERNAL C64 usa opcode non documentati del 6510. Senza questi, il codice non funziona.
**Soluzione:** Implementare gli opcode undocumented nella CPU.

### 2. CIA Interrupt
Il timer CIA genera interrupt all'underflow indipendentemente dal bit 3 di $DC0E.
**Bug precedente:** Il codice controllava erroneamente `ctrlA & 0x08`.
**Fix:** Generare sempre interrupt all'underflow.

### 3. Color RAM
La Color RAM ($D800-$DBFF) deve essere sempre accessibile al VIC-II.
**Bug precedente:** Veniva mappata alla ROM chargen quando `charon() = false`.
**Fix:** Color RAM sempre accessibile indipendentemente da CHAREN.

### 4. Reset Vector
Il reset vector del KERNAL C64 è a offset $1FFC nella ROM, non $1FFA.
**Bug precedente:** Lettura all'offset sbagliato causava rifiuto delle ROM valide.

---

## Prossimi Passi

1. Implementare tutti gli opcode undocumented della matrice
2. Verificare che il KERNAL chiami correttamente CINT ($FF81)
3. Controllare il memory banking durante l'inizializzazione
4. Testare con PRG semplici prima del KERNAL completo
