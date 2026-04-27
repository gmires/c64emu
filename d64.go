package c64

import (
	"fmt"
	"os"
	"strings"
)

// D64Reader reads Commodore 1541 disk images
type D64Reader struct {
	data []byte
}

// Track 18 Sector 0 is the BAM and directory header
// Directory entries start at track 18 sector 1

func NewD64Reader(path string) (*D64Reader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) != 174848 && len(data) != 175531 {
		return nil, fmt.Errorf("invalid D64 size: %d (expected 174848 or 175531)", len(data))
	}
	return &D64Reader{data: data}, nil
}

// Sector size is always 256 bytes in D64
func (d *D64Reader) sectorOffset(track, sector int) int {
	// Tracks 1-35, sectors per track vary
	sectorsPerTrack := []int{
		0,  // dummy for 1-based
		21, 21, 21, 21, 21, 21, 21, 21, 21, 21, // 1-10
		21, 21, 21, 21, 21, 21, 21, // 11-17
		19, 19, 19, 19, 19, 19, 19, // 18-24
		18, 18, 18, 18, 18, 18, // 25-30
		17, 17, 17, 17, 17, // 31-35
	}
	offset := 0
	for t := 1; t < track; t++ {
		offset += sectorsPerTrack[t] * 256
	}
	offset += sector * 256
	return offset
}

func (d *D64Reader) readSector(track, sector int) []byte {
	off := d.sectorOffset(track, sector)
	if off+256 > len(d.data) {
		return nil
	}
	return d.data[off : off+256]
}

// DirEntry represents a directory entry
type DirEntry struct {
	Name     string
	Size     int // in blocks
	Type     string
	Track    int
	Sector   int
	LoadAddr uint16
}

func (d *D64Reader) ListFiles() ([]DirEntry, error) {
	var entries []DirEntry
	// Directory starts at track 18, sector 1
	track, sector := 18, 1
	for track != 0 {
		sec := d.readSector(track, sector)
		if sec == nil {
			break
		}
		for i := 0; i < 8; i++ {
			off := i * 32
			fileType := sec[off+2]
			if fileType == 0 {
				continue // deleted
			}
			entryTrack := int(sec[off+3])
			entrySector := int(sec[off+4])
			nameBytes := sec[off+5 : off+21]
			name := ""
			for _, b := range nameBytes {
				if b == 0xA0 {
					break
				}
				name += string(b)
			}
			size := int(sec[off+30]) | int(sec[off+31])<<8
			typeStr := d64FileType(fileType)
			entries = append(entries, DirEntry{
				Name:   name,
				Size:   size,
				Type:   typeStr,
				Track:  entryTrack,
				Sector: entrySector,
			})
		}
		track = int(sec[0])
		sector = int(sec[1])
	}
	return entries, nil
}

func d64FileType(t uint8) string {
	switch t & 0x07 {
	case 0:
		return "DEL"
	case 1:
		return "SEQ"
	case 2:
		return "PRG"
	case 3:
		return "USR"
	case 4:
		return "REL"
	}
	return "???"
}

// ReadFile reads a PRG file from the D64 image and returns the data.
// Name matching is case-insensitive and supports partial matches.
func (d *D64Reader) ReadFile(name string) ([]byte, error) {
	entries, err := d.ListFiles()
	if err != nil {
		return nil, err
	}
	// Normalize: uppercase and trim spaces (C64 uses PETSCII uppercase)
	searchName := strings.ToUpper(strings.TrimSpace(name))

	// First try exact match
	for _, e := range entries {
		entryName := strings.ToUpper(strings.TrimSpace(e.Name))
		if entryName == searchName && e.Type == "PRG" {
			return d.readChain(e.Track, e.Sector)
		}
	}
	// Then try partial match
	for _, e := range entries {
		entryName := strings.ToUpper(strings.TrimSpace(e.Name))
		if strings.Contains(entryName, searchName) && e.Type == "PRG" {
			return d.readChain(e.Track, e.Sector)
		}
	}
	// Build helpful error message with available files
	var available []string
	for _, e := range entries {
		if e.Type == "PRG" {
			available = append(available, fmt.Sprintf("'%s'", strings.TrimSpace(e.Name)))
		}
	}
	return nil, fmt.Errorf("file '%s' not found. Available PRG files: %s", name, strings.Join(available, ", "))
}

// LoadAllPRG loads all PRG files from the D64 into machine memory at their
// respective load addresses. This is useful for multi-file games that load
// additional data during execution.
func (d *D64Reader) LoadAllPRG(machine *Machine) (int, error) {
	entries, err := d.ListFiles()
	if err != nil {
		return 0, err
	}
	loaded := 0
	for _, e := range entries {
		if e.Type != "PRG" {
			continue
		}
		data, err := d.readChain(e.Track, e.Sector)
		if err != nil {
			continue
		}
		if len(data) < 2 {
			continue
		}
		addr := uint16(data[0]) | uint16(data[1])<<8
		for i := 2; i < len(data); i++ {
			machine.Bus().Write(addr+uint16(i-2), data[i])
		}
		loaded++
	}
	return loaded, nil
}

func (d *D64Reader) readChain(startTrack, startSector int) ([]byte, error) {
	var data []byte
	track, sector := startTrack, startSector
	for track != 0 {
		sec := d.readSector(track, sector)
		if sec == nil {
			return nil, fmt.Errorf("invalid sector: track %d sector %d", track, sector)
		}
		nextTrack := int(sec[0])
		nextSector := int(sec[1])
		if nextTrack == 0 {
			// Last sector: sec[1] contains the number of valid bytes
			validBytes := nextSector - 1
			if validBytes < 0 {
				validBytes = 0
			}
			if validBytes > 254 {
				validBytes = 254
			}
			data = append(data, sec[2:2+validBytes]...)
			break
		}
		data = append(data, sec[2:]...)
		track = nextTrack
		sector = nextSector
	}
	return data, nil
}
