package audio

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// LoadBGM loads background music from a file
func (m *Manager) LoadBGM(filename string) error {
	log.Printf("Loading BGM: %s", filename)

	var data []byte
	var err error

	if strings.Contains(filename, ".gpk") || strings.Contains(filename, "::") {
		// GPK format: "package.gpk::entry/path.ogg"
		parts := strings.Split(filename, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid GPK path format: %s", filename)
		}
		data, err = m.loadAudioFromGPK(parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to load BGM from GPK: %w", err)
		}
	} else {
		// Regular file
		data, err = m.loadAudioFromFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load BGM from file: %w", err)
		}
	}

	// Cache the audio file
	m.loadedBGM = &AudioFile{
		Name:      filepath.Base(filename),
		Path:      filename,
		Data:      data,
		IsFromGPK: strings.Contains(filename, ".gpk") || strings.Contains(filename, "::"),
	}

	// Close existing BGM player
	if m.bgmPlayer != nil {
		m.bgmPlayer.Close()
		m.bgmPlayer = nil
	}

	// Create new BGM player
	player, err := m.createPlayerFromData(data)
	if err != nil {
		return fmt.Errorf("failed to create BGM player: %w", err)
	}

	m.bgmPlayer = player
	m.bgmPlayer.SetVolume(m.bgmVolume)

	log.Printf("BGM loaded successfully: %s", filename)
	return nil
}

// PlayBGM starts background music playback
func (m *Manager) PlayBGM(loops int) error {
	if m.muted || m.bgmPlayer == nil {
		return nil
	}

	// Set volume
	m.bgmPlayer.SetVolume(m.bgmVolume)

	// Rewind and play
	m.bgmPlayer.Rewind()
	m.bgmPlayer.Play()

	log.Printf("Playing BGM (loops: %d)", loops)
	return nil
}

// StopBGM stops background music
func (m *Manager) StopBGM() {
	if m.bgmPlayer != nil {
		m.bgmPlayer.Pause()
		log.Println("BGM stopped")
	}
}

// IsBGMPlaying returns true if BGM is currently playing
func (m *Manager) IsBGMPlaying() bool {
	return m.bgmPlayer != nil && m.bgmPlayer.IsPlaying()
}

// GetCurrentBGMFile returns the currently loaded BGM file name
func (m *Manager) GetCurrentBGMFile() string {
	if m.loadedBGM != nil {
		return m.loadedBGM.Name
	}
	return ""
}
