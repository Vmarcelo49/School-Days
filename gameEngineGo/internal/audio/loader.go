package audio

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

// loadAudioFromFile loads audio data from a file path
func (m *Manager) loadAudioFromFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return data, nil
}

// loadAudioFromGPK loads audio data from a GPK package
func (m *Manager) loadAudioFromGPK(gpkFile, entryPath string) ([]byte, error) {
	// TODO: Connect with GPK filesystem manager
	// For now, return an error
	return nil, fmt.Errorf("GPK loading not yet implemented - need to connect with filesystem manager")
}

// createPlayerFromData creates an audio player from raw data
func (m *Manager) createPlayerFromData(data []byte) (*audio.Player, error) {
	// Try to fix OGG header if needed (for GPK files)
	fixedData, err := m.fixOggHeader(data)
	if err != nil {
		// If fixing fails, try with original data
		fixedData = data
	}

	stream, err := vorbis.DecodeWithoutResampling(bytes.NewReader(fixedData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode OGG file: %w", err)
	}

	player, err := m.context.NewPlayer(stream)
	if err != nil {
		return nil, fmt.Errorf("failed to create OGG player: %w", err)
	}

	return player, nil
}

// PreloadAudioFiles preloads common audio files from a directory
func (m *Manager) PreloadAudioFiles(audioDir string) error {
	// This method would scan the audio directory and preload common files
	// For now, just log that it was called
	log.Printf("Preloading audio files from directory: %s", audioDir)

	// Common system sounds mapping
	systemSounds := map[int]string{
		SndTitle:  "title.ogg",
		SndSystem: "system.ogg",
		SndCancel: "cancel.ogg",
		SndSelect: "select.ogg",
		SndClick:  "click.ogg",
		SndUp:     "up.ogg",
		SndDown:   "down.ogg",
		SndView:   "view.ogg",
		SndOpen:   "open.ogg",
	}

	// Try to load system sounds
	for id, filename := range systemSounds {
		fullPath := filepath.Join(audioDir, filename)
		if _, err := os.Stat(fullPath); err == nil {
			if err := m.LoadSoundEffect(fullPath, id); err != nil {
				log.Printf("Warning: Failed to preload %s: %v", filename, err)
			}
		}
	}

	log.Println("Audio preloading completed")
	return nil
}

// GetMemoryUsage returns the current memory usage of loaded audio files
func (m *Manager) GetMemoryUsage() int {
	var totalBytes int

	if m.loadedBGM != nil {
		totalBytes += len(m.loadedBGM.Data)
	}

	for _, sound := range m.loadedSounds {
		if sound != nil {
			totalBytes += len(sound.Data)
		}
	}

	return totalBytes
}

// GetAudioInfo returns information about loaded audio files
func (m *Manager) GetAudioInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// BGM info
	if m.loadedBGM != nil {
		info["bgm"] = map[string]interface{}{
			"name":    m.loadedBGM.Name,
			"path":    m.loadedBGM.Path,
			"size":    len(m.loadedBGM.Data),
			"fromGPK": m.loadedBGM.IsFromGPK,
			"playing": m.IsBGMPlaying(),
		}
	}

	// Sound effects info
	soundEffects := make(map[string]interface{})
	for i, sound := range m.loadedSounds {
		if sound != nil {
			soundEffects[GetSystemSoundName(i)] = map[string]interface{}{
				"name":    sound.Name,
				"path":    sound.Path,
				"size":    len(sound.Data),
				"fromGPK": sound.IsFromGPK,
				"id":      i,
			}
		}
	}
	info["soundEffects"] = soundEffects

	// Volume info
	info["volumes"] = map[string]interface{}{
		"bgm":   m.bgmVolume,
		"se":    m.seVolume,
		"voice": m.voiceVolume,
		"muted": m.muted,
	}

	// Memory usage
	info["memoryUsage"] = m.GetMemoryUsage()

	return info
}

// LoadConfig loads audio configuration
func (m *Manager) LoadConfig(config AudioLoadConfig) error {
	log.Println("Loading audio configuration")

	// Apply volume settings
	m.SetBGMVolume(config.VolumeSettings.BGMVolume)
	m.SetSEVolume(config.VolumeSettings.SEVolume)
	m.SetVoiceVolume(config.VolumeSettings.VoiceVolume)
	m.SetMuted(config.VolumeSettings.Muted)

	// Preload system sounds if requested
	if config.PreloadSystemSounds && config.AudioDirectory != "" {
		if err := m.PreloadAudioFiles(config.AudioDirectory); err != nil {
			log.Printf("Warning: Failed to preload audio files: %v", err)
		}
	}

	log.Println("Audio configuration loaded successfully")
	return nil
}

// GetConfig returns current audio configuration
func (m *Manager) GetConfig() AudioLoadConfig {
	return AudioLoadConfig{
		PreloadSystemSounds: false,      // This is a load-time setting
		AudioDirectory:      "",         // This is a load-time setting
		GPKPackages:         []string{}, // This is a load-time setting
		VolumeSettings: VolumeSettings{
			BGMVolume:   m.bgmVolume,
			SEVolume:    m.seVolume,
			VoiceVolume: m.voiceVolume,
			Muted:       m.muted,
		},
	}
}

// LoadFromScript loads audio specified in a script command
// This integrates with the script engine for "play bgm", "play se" commands
func (m *Manager) LoadFromScript(command, filename string, options map[string]interface{}) error {
	if err := ValidateAudioPath(filename); err != nil {
		return fmt.Errorf("invalid audio path in script: %w", err)
	}

	switch strings.ToLower(command) {
	case "bgm", "play_bgm", "load_bgm":
		if err := m.LoadBGM(filename); err != nil {
			return fmt.Errorf("failed to load BGM from script: %w", err)
		}

		// Check for auto-play option
		if autoPlay, ok := options["play"]; ok && autoPlay.(bool) {
			loops := -1 // Default to infinite loop
			if loopCount, ok := options["loops"]; ok {
				loops = loopCount.(int)
			}
			return m.PlayBGM(loops)
		}

	case "se", "play_se", "sound_effect":
		// Check if this is a system sound by ID
		if soundID, ok := options["id"]; ok {
			id := soundID.(int)
			if err := m.LoadSoundEffect(filename, id); err != nil {
				return fmt.Errorf("failed to load sound effect from script: %w", err)
			}

			// Auto-play if requested
			if autoPlay, ok := options["play"]; ok && autoPlay.(bool) {
				m.PlaySoundEffect(id)
			}
		} else {
			// Play as one-shot sound
			return m.PlaySoundEffectByName(filename)
		}

	case "voice", "play_voice":
		// TODO: Implement voice playback (similar to SE but with voice volume)
		log.Printf("Voice playback not yet implemented: %s", filename)
		return fmt.Errorf("voice playback not yet implemented")

	default:
		return fmt.Errorf("unknown audio command: %s", command)
	}

	return nil
}
