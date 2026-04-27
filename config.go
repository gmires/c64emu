package c64

const (
	CLK_PAL  = 985248   // Hz - PAL clock
	CLK_NTSC = 1022737  // Hz - NTSC clock

	VIC_MEM  = 0xD000   // VIC-II I/O base
	CIA_MEM  = 0xDC00   // CIA I/O base
	SID_MEM  = 0xD400   // SID I/O base

	RAM_SIZE = 65536    // 64KB
	ROM_KERN = 8192     // 8KB BASIC
	ROM_BASIC = 8192    // 8KB KERNAL
	ROM_CHAR = 4096     // 4KB CHARGEN

	COLOR_MEM = 1024    // 1KB color RAM
)

type SystemConfig struct {
	PAL      bool
	RAM      []uint8
	ROMBasic []uint8
	ROMKernal []uint8
	ROMChar  []uint8
}

func DefaultConfig() *SystemConfig {
	return &SystemConfig{
		PAL: true,
		RAM: make([]uint8, RAM_SIZE),
	}
}