package main

import (
	"flag"
	"fmt"
	"os"

	c64 "github.com/c64-emu/c64-emu"
)

func main() {
	var (
		testMode = flag.Bool("test", false, "Run test pattern and export screenshot")
		prgFile  = flag.String("prg", "", "Load PRG file")
		d64File  = flag.String("d64", "", "Load D64 disk image")
		d64Entry = flag.String("d64file", "", "File to load from D64")
		tapFile  = flag.String("tap", "", "Load TAP tape image")
		crtFile  = flag.String("crt", "", "Load CRT cartridge")
		autoRun  = flag.Bool("autorun", false, "Auto-run loaded PRG (RUN for BASIC, JMP for ML)")
		frames   = flag.Int("frames", 1, "Number of frames to run (0 = infinite)")
		pngOut   = flag.String("png", "screenshot.png", "Output PNG file")
		gui      = flag.Bool("gui", false, "Run in GUI mode with window")
		scale      = flag.Float64("scale", 2.0, "Display scale factor for GUI mode")
		fullscreen = flag.Bool("fullscreen", false, "Run in fullscreen mode")
		ntsc       = flag.Bool("ntsc", false, "Use NTSC timing (default: PAL)")
		fallback   = flag.Bool("fallback", false, "Force use built-in fallback ROMs")
		debug      = flag.Bool("debug", false, "Enable debug visualization overlay")
		saveFile   = flag.String("save", "", "Save state to file after running")
		loadFile   = flag.String("load", "", "Load state from file before running")
	)
	flag.Parse()

	cfg := c64.DefaultConfig()
	cfg.PAL = !*ntsc

	machine := c64.NewMachine(cfg)

	// Load ROMs from disk
	roms := []struct {
		slot string
		path string
	}{
		{"basic", "roms/basic"},
		{"kernal", "roms/kernal"},
		{"chargen", "roms/chargen"},
	}

	romsLoaded := false
	for _, r := range roms {
		if _, err := os.Stat(r.path); err == nil {
			if err := machine.LoadROM(r.path, r.slot); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}

	// Check if ROMs are valid C64 ROMs
	if !machine.HasValidROMs() {
		fmt.Println("WARNING: ROMs missing or invalid (not standard C64)")
		fmt.Println("Using built-in fallback boot screen...")
		*fallback = true
		// Clear potentially corrupt chargen so embedded font is used
		machine.LoadROM("", "chargen")
	} else {
		romsLoaded = true
		fmt.Println("Valid C64 ROMs detected!")
	}

	machine.Reset()

	// Load state if requested
	if *loadFile != "" {
		if err := machine.LoadStateFromFile(*loadFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded state from %s\n", *loadFile)
	}

	// Enable debug visualization
	if *debug {
		c64.DebugRenderOverlay = true
		fmt.Println("Debug visualization enabled")
	}

	// Test mode: inject pattern into screen RAM
	if *testMode {
		fmt.Println("Test mode: injecting pattern...")
		machine.InjectTestPattern()
	}

	// Fallback boot screen (when no valid ROMs)
	if *fallback && !*testMode {
		machine.ShowBootScreen()
	}

	// Load PRG
	var prgAddr uint16
	if *prgFile != "" {
		data, err := os.ReadFile(*prgFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading PRG: %v\n", err)
			os.Exit(1)
		}
		if len(data) >= 2 {
			prgAddr = uint16(data[0]) | uint16(data[1])<<8
		}
		if err := machine.LoadPRG(*prgFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading PRG: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded PRG: %s at $%04X\n", *prgFile, prgAddr)
	}

	// Load from D64
	if *d64File != "" {
		d64, err := c64.NewD64Reader(*d64File)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading D64: %v\n", err)
			os.Exit(1)
		}
		if *d64Entry == "" {
			// List files
			entries, err := d64.ListFiles()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading D64: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Files in %s:\n", *d64File)
			for _, e := range entries {
				fmt.Printf("  %-16s %s %3d blocks\n", e.Name, e.Type, e.Size)
			}
			os.Exit(0)
		}
		data, err := d64.ReadFile(*d64Entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file from D64: %v\n", err)
			os.Exit(1)
		}
		if len(data) >= 2 {
			prgAddr = uint16(data[0]) | uint16(data[1])<<8
		}
		for i := 2; i < len(data); i++ {
			machine.Bus().Write(prgAddr+uint16(i-2), data[i])
		}
		fmt.Printf("Loaded '%s' from D64 at $%04X (%d bytes)\n", *d64Entry, prgAddr, len(data)-2)
	}

	// Load from TAP
	if *tapFile != "" {
		tap, err := c64.NewTAPReader(*tapFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading TAP: %v\n", err)
			os.Exit(1)
		}
		data, err := tap.ExtractPRG()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting PRG from TAP: %v\n", err)
			os.Exit(1)
		}
		if len(data) >= 2 {
			prgAddr = uint16(data[0]) | uint16(data[1])<<8
		}
		for i := 2; i < len(data); i++ {
			machine.Bus().Write(prgAddr+uint16(i-2), data[i])
		}
		fmt.Printf("Loaded PRG from TAP at $%04X (%d bytes)\n", prgAddr, len(data)-2)
	}

	// Load CRT cartridge
	if *crtFile != "" {
		crt, err := c64.NewCRTReader(*crtFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading CRT: %v\n", err)
			os.Exit(1)
		}
		if err := crt.LoadIntoMachine(machine); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading CRT: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Loaded cartridge: %s (HW type %d, %d banks)\n", crt.Name(), crt.HWType(), len(crt.Banks()))
	}

	// Auto-run
	if *autoRun && *prgFile != "" && romsLoaded {
		fmt.Println("Auto-run enabled...")
		machine.AutoRun(prgAddr)
	}

	fmt.Println("C64 Emulator initialized")
	fmt.Printf("CPU: PC=%04X A=%02X X=%02X Y=%02X SP=%02X SR=%02X\n",
		machine.CPU().PC,
		machine.CPU().A,
		machine.CPU().X,
		machine.CPU().Y,
		machine.CPU().SP,
		machine.CPU().SR,
	)

	// GUI mode
	if *gui {
		display, err := c64.NewDisplay(machine, *scale, *fullscreen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "GUI error: %v\n", err)
			os.Exit(1)
		}
		if err := display.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "GUI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Headless mode
	if *frames > 0 {
		fmt.Printf("Running %d frame(s)...\n", *frames)
		machine.RunFrames(*frames)

		if err := machine.ExportBMP(*pngOut); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving PNG: %v\n", err)
		} else {
			fmt.Printf("Saved screenshot: %s\n", *pngOut)
		}
		if *saveFile != "" {
			if err := machine.SaveStateToFile(*saveFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
			} else {
				fmt.Printf("Saved state: %s\n", *saveFile)
			}
		}
	} else {
		fmt.Println("Running infinite loop (Ctrl+C to stop)...")
		machine.Run()
	}
}
