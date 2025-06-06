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

// GPKAudioReader mimics the C++ ov_callbacks approach for streaming OGG from GPK files
// This handles compression headers transparently, just like the C++ Stream class
type GPKAudioReader struct {
	gpk      *GPK
	entry    *GPKEntry
	data     []byte
	position int64
	size     int64
}

// ReadSeekCloser combines io.Reader, io.Seeker, and io.Closer for in-memory audio data
type ReadSeekCloser struct {
	*bytes.Reader
}

func (r *ReadSeekCloser) Close() error {
	return nil // No-op for in-memory data
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

// NewGPKAudioReader creates a new GPK audio reader
func NewGPKAudioReader(gpk *GPK, entry *GPKEntry) (*GPKAudioReader, error) {
	// Open the GPK file
	file, err := os.Open(gpk.fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPK file: %w", err)
	}
	defer file.Close()

	// Seek to the entry offset
	_, err = file.Seek(int64(entry.Header.Offset), 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to entry: %w", err)
	}

	// Read the compressed data
	compressedData := make([]byte, entry.Header.CompressedFileLen)
	_, err = file.Read(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to read entry data: %w", err)
	}
	oggData := compressedData // ogg files are already decompressed in GPK

	return &GPKAudioReader{
		gpk:      gpk,
		entry:    entry,
		data:     oggData,
		position: 0,
		size:     int64(len(oggData)),
	}, nil
}

// Read implements io.Reader interface for GPKAudioReader
func (r *GPKAudioReader) Read(p []byte) (n int, err error) {
	if r.position >= r.size {
		return 0, io.EOF
	}

	available := r.size - r.position
	toRead := int64(len(p))
	if toRead > available {
		toRead = available
	}

	copy(p, r.data[r.position:r.position+toRead])
	r.position += toRead
	return int(toRead), nil
}

// Seek implements io.Seeker interface for GPKAudioReader
func (r *GPKAudioReader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = r.position + offset
	case io.SeekEnd:
		newPos = r.size + offset
	default:
		return 0, fmt.Errorf("invalid whence value: %d", whence)
	}

	if newPos < 0 {
		newPos = 0
	}
	if newPos > r.size {
		newPos = r.size
	}

	r.position = newPos
	return newPos, nil
}

// Close implements io.Closer interface for GPKAudioReader
func (r *GPKAudioReader) Close() error {
	return nil // No resources to close for in-memory data
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
	}
	// Decode the OGG file using streaming approach for better compatibility
	var reader io.ReadSeeker

	if audioFile.IsFromGPK {
		// Use our custom streaming reader for GPK files that may have header issues
		reader = NewOGGStreamReader(data, audioFile.GPKEntry)
	} else {
		// For regular files, apply OGG header fix if needed
		fixedData, err := fixOggHeader(data)
		if err != nil {
			// If fixing fails, log the error and use original data
			log.Printf("Warning: Failed to fix OGG header for %s: %v", audioFile.Name, err)
			fixedData = data
		}
		reader = bytes.NewReader(fixedData)
	}
	// Decode with the streaming interface
	stream, err := vorbis.DecodeWithSampleRate(sampleRate, reader)
	if err != nil {
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
	ebitenutil.DebugPrintAt(screen, "LEFT/RIGHT: Previous/Next file", 10, 70)

	// Show current file
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

// NewOGGStreamReader creates a new stream reader for OGG data
func NewOGGStreamReader(data []byte, filename string) *OGGStreamReader {
	// Try to fix any header issues first
	fixedData, err := fixOggHeader(data)
	if err != nil {
		// If fixing fails, log the error and use original data
		log.Printf("Warning: Failed to fix OGG header for %s: %v", filename, err)
		fixedData = data
	}

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runGameWindow() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Simple OGG Audio Player")

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
