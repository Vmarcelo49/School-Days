package main

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 44100
)

// AudioFile represents an audio file that can be from disk or GPK
type AudioFile struct {
	Name      string // Display name
	Path      string // Full path for disk files or GPK package name
	IsFromGPK bool   // Whether this file is from a GPK package
	GPKEntry  string // Entry name within GPK (if IsFromGPK is true)
}

type Game struct {
	audioContext *audio.Context
	audioPlayer  *audio.Player
	audioFiles   []AudioFile
	currentFile  int
	isPlaying    bool
	gpkPackages  []*GPK // Loaded GPK packages
}

func NewGame() *Game {
	audioContext := audio.NewContext(sampleRate)

	game := &Game{
		audioContext: audioContext,
		audioFiles:   make([]AudioFile, 0),
		currentFile:  0,
		isPlaying:    false,
		gpkPackages:  make([]*GPK, 0),
	}

	// Find all audio files (both loose OGG files and from GPK packages)
	err := game.findAudioFiles(".")
	if err != nil {
		log.Printf("Error finding audio files: %v", err)
	}

	return game
}

// findAudioFiles scans for both loose OGG files and OGG files within GPK packages
func (g *Game) findAudioFiles(dir string) error {
	// First, find loose OGG files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".ogg" {
				// Add loose OGG file
				g.audioFiles = append(g.audioFiles, AudioFile{
					Name:      filepath.Base(path),
					Path:      path,
					IsFromGPK: false,
				})
			} else if ext == ".gpk" {
				// Load GPK package and scan for OGG files
				err := g.loadGPKPackage(path)
				if err != nil {
					log.Printf("Warning: Failed to load GPK package %s: %v", path, err)
				}
			}
		}

		return nil
	})

	log.Printf("Found %d audio files (%d GPK packages loaded)", len(g.audioFiles), len(g.gpkPackages))
	return err
}

// loadGPKPackage loads a GPK package and adds its OGG files to the audio files list
func (g *Game) loadGPKPackage(gpkPath string) error {
	gpk := NewGPK()
	err := gpk.Load(gpkPath)
	if err != nil {
		return fmt.Errorf("failed to load GPK: %w", err)
	}

	g.gpkPackages = append(g.gpkPackages, gpk)

	// Add OGG files from this GPK package
	entries := gpk.GetEntries()
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToUpper(entry.Name), ".OGG") {
			g.audioFiles = append(g.audioFiles, AudioFile{
				Name:      fmt.Sprintf("[%s] %s", gpk.GetName(), filepath.Base(entry.Name)),
				Path:      gpkPath,
				IsFromGPK: true,
				GPKEntry:  entry.Name,
			})
		}
	}

	log.Printf("Loaded GPK package: %s with %d OGG files", gpk.GetName(),
		len(gpk.List("*.ogg")))

	return nil
}

func (g *Game) loadAudioFile(audioFile AudioFile) error {
	// Stop current player if playing
	if g.audioPlayer != nil {
		g.audioPlayer.Close()
		g.audioPlayer = nil
	}

	var data []byte
	var err error

	if audioFile.IsFromGPK {
		// Load from GPK package
		data, err = g.loadFromGPK(audioFile.Path, audioFile.GPKEntry)
		if err != nil {
			return fmt.Errorf("failed to load from GPK %s: %v", audioFile.Name, err)
		}
	} else {
		// Load from disk
		file, err := os.Open(audioFile.Path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", audioFile.Path, err)
		}
		defer file.Close()

		data, err = io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", audioFile.Path, err)
		}
	} // Decode the OGG file using streaming approach for better compatibility
	var reader io.ReadSeeker

	if audioFile.IsFromGPK {
		// Use our custom streaming reader for GPK files that may have header issues
		reader = NewOGGStreamReader(data, audioFile.GPKEntry)
	} else {
		// For regular files, use the data directly
		fixedData := data // Here we could implement a proper header fix
		reader = bytes.NewReader(fixedData)
	} // Decode with the streaming interface
	stream, err := vorbis.DecodeWithSampleRate(sampleRate, reader)
	if err != nil {
		// If decoding fails, analyze the OGG structure for debugging
		log.Printf("OGG decoding failed for %s: %v", audioFile.Name, err)

		var analysisFilename string
		if audioFile.IsFromGPK {
			analysisFilename = audioFile.GPKEntry
			analyzeOGGStructure(data, audioFile.GPKEntry)
			analyzeOGGVorbisHeaders(data, audioFile.GPKEntry)
			analyzeMultipleOGGPages(data, audioFile.GPKEntry)
		} else {
			analysisFilename = audioFile.Name
			analyzeOGGStructure(data, audioFile.Name)
			analyzeOGGVorbisHeaders(data, audioFile.Name)
			analyzeMultipleOGGPages(data, audioFile.Name)
		}

		// Save the problematic OGG data for external analysis
		saveOGGForAnalysis(data, analysisFilename)

		return fmt.Errorf("failed to decode OGG file %s: %v", audioFile.Name, err)
	}

	// Create audio player
	g.audioPlayer, err = g.audioContext.NewPlayer(stream)
	if err != nil {
		return fmt.Errorf("failed to create audio player: %v", err)
	}

	return nil
}

// loadFromGPK loads an OGG file from a GPK package
func (g *Game) loadFromGPK(gpkPath, entryName string) ([]byte, error) {
	// Find the GPK package
	var targetGPK *GPK
	for _, gpk := range g.gpkPackages {
		if gpk.fileName == gpkPath {
			targetGPK = gpk
			break
		}
	}

	if targetGPK == nil {
		return nil, fmt.Errorf("GPK package not found: %s", gpkPath)
	}

	// Open the file from GPK
	gpkFile, err := targetGPK.Open(entryName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file from GPK: %w", err)
	}
	defer gpkFile.Close()

	// Read all data
	data, err := gpkFile.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read GPK file data: %w", err)
	}
	// Fix OGG , we need to implement a proper header fix
	data = data

	return data, nil
}

func (g *Game) Update() error {
	// Handle input
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		if len(g.audioFiles) > 0 {
			if g.audioPlayer == nil {
				// Load the first file
				if err := g.loadAudioFile(g.audioFiles[g.currentFile]); err != nil {
					log.Printf("Error loading audio file: %v", err)
					return nil
				}
			}

			if g.isPlaying {
				g.audioPlayer.Pause()
				g.isPlaying = false
			} else {
				g.audioPlayer.Play()
				g.isPlaying = true
			}
		}
	}

	// Next file
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) && len(g.audioFiles) > 0 {
		g.currentFile = (g.currentFile + 1) % len(g.audioFiles)
		if err := g.loadAudioFile(g.audioFiles[g.currentFile]); err != nil {
			log.Printf("Error loading next audio file: %v", err)
		}
		g.isPlaying = false
	}

	// Previous file
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && len(g.audioFiles) > 0 {
		g.currentFile = (g.currentFile - 1 + len(g.audioFiles)) % len(g.audioFiles)
		if err := g.loadAudioFile(g.audioFiles[g.currentFile]); err != nil {
			log.Printf("Error loading previous audio file: %v", err)
		}
		g.isPlaying = false
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0x20, 0x20, 0x40, 0xff})

	// Draw instructions
	ebitenutil.DebugPrint(screen, "Simple OGG Audio Player")
	ebitenutil.DebugPrintAt(screen, "Controls:", 10, 30)
	ebitenutil.DebugPrintAt(screen, "SPACE: Play/Pause", 10, 50)
	ebitenutil.DebugPrintAt(screen, "LEFT/RIGHT: Previous/Next file", 10, 70) // Show current file
	if len(g.audioFiles) > 0 {
		currentAudio := g.audioFiles[g.currentFile]
		var displayName string
		if currentAudio.IsFromGPK {
			displayName = fmt.Sprintf("%s (from %s)", currentAudio.GPKEntry, filepath.Base(currentAudio.Path))
		} else {
			displayName = filepath.Base(currentAudio.Path)
		}

		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Current file: %s", displayName), 10, 100)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("File %d of %d", g.currentFile+1, len(g.audioFiles)), 10, 120)

		status := "Stopped"
		if g.isPlaying {
			status = "Playing"
		} else if g.audioPlayer != nil {
			status = "Paused"
		}
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Status: %s", status), 10, 140)
	} else {
		ebitenutil.DebugPrintAt(screen, "No OGG files found in current directory or GPK packages", 10, 100)
		ebitenutil.DebugPrintAt(screen, "Place some .ogg files or .gpk packages in the same folder", 10, 120)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// OGGStreamReader provides a streaming interface for OGG files that may have corrupted headers
// This mimics the C++ ov_callbacks approach for better compatibility
type OGGStreamReader struct {
	data   []byte
	offset int64
	size   int64
}

// needs to be remade to fix OGG header issues
func NewOGGStreamReader(data []byte, filename string) *OGGStreamReader {
	// Try to fix any header issues first
	fixedData := data

	return &OGGStreamReader{
		data:   fixedData,
		offset: 0,
		size:   int64(len(fixedData)),
	}
}

// Read implements io.Reader interface
func (r *OGGStreamReader) Read(p []byte) (n int, err error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}

	n = copy(p, r.data[r.offset:])
	r.offset += int64(n)
	return n, nil
}

// Seek implements io.Seeker interface
func (r *OGGStreamReader) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64

	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.size + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}

	if newOffset < 0 {
		newOffset = 0
	} else if newOffset > r.size {
		newOffset = r.size
	}

	r.offset = newOffset
	return newOffset, nil
}

// Size returns the total size of the data
func (r *OGGStreamReader) Size() int64 {
	return r.size
}

// analyzeOGGStructure provides debugging information about OGG file structure
func analyzeOGGStructure(data []byte, filename string) {
	log.Printf("Analyzing OGG structure for: %s", filename)
	log.Printf("Data size: %d bytes", len(data))

	if len(data) < 4 {
		log.Printf("  ERROR: File too small for OGG")
		return
	}

	// Look for OggS pattern
	oggStartIndex := -1
	for i := 0; i < len(data)-3; i++ {
		if data[i] == 'O' && data[i+1] == 'g' && data[i+2] == 'g' && data[i+3] == 'S' {
			oggStartIndex = i
			break
		}
	}

	if oggStartIndex == -1 {
		log.Printf("  ERROR: No OggS signature found")
		log.Printf("  First 20 bytes: %v", data[:min(20, len(data))])
		return
	}

	log.Printf("  OggS found at offset: %d", oggStartIndex)

	if oggStartIndex > 0 {
		log.Printf("  Header prefix size: %d bytes", oggStartIndex)
		log.Printf("  Header prefix: %v", data[:min(oggStartIndex, 20)])
	}

	if len(data) >= oggStartIndex+27 {
		oggData := data[oggStartIndex:]
		version := oggData[4]
		headerType := oggData[5]
		pageSegments := oggData[26]

		log.Printf("  OGG version: %d", version)
		log.Printf("  Header type: %d", headerType)
		log.Printf("  Page segments: %d", pageSegments)

		if version != 0 {
			log.Printf("  WARNING: Non-standard OGG version")
		}

		headerSize := 27 + int(pageSegments)
		if len(oggData) >= headerSize {
			log.Printf("  Header appears complete (size: %d)", headerSize)
		} else {
			log.Printf("  WARNING: Incomplete header (need %d, have %d)", headerSize, len(oggData))
		}
	} else {
		log.Printf("  ERROR: OGG header incomplete")
	}
}

// analyzeOGGVorbisHeaders provides detailed analysis of OGG Vorbis header packets
func analyzeOGGVorbisHeaders(data []byte, filename string) {
	log.Printf("Detailed Vorbis analysis for: %s", filename)

	if len(data) < 45 {
		log.Printf("  ERROR: Data too small for complete OGG page")
		return
	}

	// Parse first OGG page header
	pageSegments := data[26]
	headerSize := 27 + int(pageSegments)

	if len(data) < headerSize {
		log.Printf("  ERROR: Incomplete page header")
		return
	}

	// Calculate packet size from segment table
	var packetSize int
	for i := 0; i < int(pageSegments); i++ {
		segmentLen := int(data[27+i])
		packetSize += segmentLen
		log.Printf("  Segment %d size: %d", i, segmentLen)
	}

	log.Printf("  Total packet size: %d bytes", packetSize)

	if len(data) < headerSize+packetSize {
		log.Printf("  ERROR: Incomplete packet data (need %d, have %d)", headerSize+packetSize, len(data))
		return
	}

	// Analyze Vorbis identification header
	packetData := data[headerSize : headerSize+packetSize]
	if len(packetData) < 30 {
		log.Printf("  ERROR: Vorbis packet too small")
		return
	}

	// Check Vorbis identification header
	if packetData[0] != 0x01 {
		log.Printf("  ERROR: Expected Vorbis ID header (0x01), got 0x%02x", packetData[0])
		return
	}

	if string(packetData[1:7]) != "vorbis" {
		log.Printf("  ERROR: Expected 'vorbis' identifier, got: %s", string(packetData[1:7]))
		return
	}

	// Parse Vorbis stream info
	version := uint32(packetData[7]) | uint32(packetData[8])<<8 | uint32(packetData[9])<<16 | uint32(packetData[10])<<24
	channels := packetData[11]
	sampleRate := uint32(packetData[12]) | uint32(packetData[13])<<8 | uint32(packetData[14])<<16 | uint32(packetData[15])<<24

	log.Printf("  Vorbis version: %d", version)
	log.Printf("  Channels: %d", channels)
	log.Printf("  Sample rate: %d Hz", sampleRate)

	if version != 0 {
		log.Printf("  WARNING: Non-standard Vorbis version")
	}

	if channels == 0 || channels > 8 {
		log.Printf("  ERROR: Invalid channel count")
	}

	if sampleRate < 8000 || sampleRate > 192000 {
		log.Printf("  WARNING: Unusual sample rate")
	}

	// Check framing bit
	if packetData[len(packetData)-1]&0x01 == 0 {
		log.Printf("  ERROR: Framing bit not set in identification header")
	}

	log.Printf("  Vorbis identification header appears valid")
}

// analyzeMultipleOGGPages analyzes all OGG pages in the data to find header packets
func analyzeMultipleOGGPages(data []byte, filename string) {
	log.Printf("Multi-page OGG analysis for: %s", filename)

	offset := 0
	pageNum := 0

	for offset < len(data)-27 {
		// Check for OggS signature
		if offset+4 <= len(data) && string(data[offset:offset+4]) == "OggS" {
			log.Printf("  Page %d at offset %d:", pageNum, offset)

			if offset+27 <= len(data) {
				version := data[offset+4]
				headerType := data[offset+5]
				pageSegments := data[offset+26]

				log.Printf("    Version: %d, Header type: %d, Segments: %d", version, headerType, pageSegments)

				headerSize := 27 + int(pageSegments)
				if offset+headerSize <= len(data) {
					// Calculate packet sizes
					var totalPacketSize int
					for i := range int(pageSegments) {
						segmentLen := int(data[offset+27+i])
						totalPacketSize += segmentLen
					}

					log.Printf("    Packet data size: %d bytes", totalPacketSize)

					// Check packet content if available
					packetStart := offset + headerSize
					if packetStart < len(data) && totalPacketSize > 0 && packetStart+totalPacketSize <= len(data) {
						if totalPacketSize > 0 {
							packetType := data[packetStart]
							log.Printf("    First packet type: 0x%02x", packetType)

							if totalPacketSize >= 7 && packetType <= 0x05 {
								vorbisId := string(data[packetStart+1 : packetStart+7])
								if vorbisId == "vorbis" {
									log.Printf("    Vorbis packet detected: %s", getVorbisPacketTypeName(packetType))
								} else {
									log.Printf("    Non-vorbis packet or corrupted data")
								}
							}
						}
					}

					// Move to next page
					nextOffset := offset + headerSize + totalPacketSize
					// Align to next OggS boundary if needed
					found := false
					for i := nextOffset; i < len(data)-3; i++ {
						if string(data[i:i+4]) == "OggS" {
							offset = i
							found = true
							break
						}
					}
					if !found {
						break
					}
				} else {
					break
				}
			} else {
				break
			}
			pageNum++
		} else {
			offset++
		}
	}

	log.Printf("  Total pages found: %d", pageNum)
}

// getVorbisPacketTypeName returns human readable packet type name
func getVorbisPacketTypeName(packetType byte) string {
	switch packetType {
	case 0x01:
		return "Identification Header"
	case 0x03:
		return "Comment Header"
	case 0x05:
		return "Setup Header"
	default:
		return "Audio Data"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// saveOGGForAnalysis saves the extracted OGG data to a file for external analysis
func saveOGGForAnalysis(data []byte, filename string) {
	// Create a safe filename
	safeFilename := strings.ReplaceAll(filename, "/", "_")
	safeFilename = strings.ReplaceAll(safeFilename, "\\", "_")
	safeFilename = "debug_" + safeFilename

	err := os.WriteFile(safeFilename, data, 0644)
	if err != nil {
		log.Printf("Failed to save OGG data for analysis: %v", err)
	} else {
		log.Printf("Saved OGG data to %s for external analysis", safeFilename)
	}
}

func runGameWindow() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Simple OGG Audio Player")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
