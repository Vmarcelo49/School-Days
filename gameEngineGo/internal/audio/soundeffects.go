package audio

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

// LoadSoundEffect loads a sound effect with the given ID
func (m *Manager) LoadSoundEffect(filename string, id int) error {
	if id < 0 || id >= SndSize {
		return fmt.Errorf("invalid sound effect ID: %d", id)
	}

	log.Printf("Loading sound effect: %s (ID: %d)", filename, id)

	var data []byte
	var err error

	if strings.Contains(filename, ".gpk") || strings.Contains(filename, "::") {
		// GPK format
		parts := strings.Split(filename, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid GPK path format: %s", filename)
		}
		data, err = m.loadAudioFromGPK(parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to load sound effect from GPK: %w", err)
		}
	} else {
		// Regular file
		data, err = m.loadAudioFromFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load sound effect from file: %w", err)
		}
	}

	// Cache the audio file
	m.loadedSounds[id] = &AudioFile{
		Name:      filepath.Base(filename),
		Path:      filename,
		Data:      data,
		IsFromGPK: strings.Contains(filename, ".gpk") || strings.Contains(filename, "::"),
	}

	// Create and cache the player
	if m.soundPlayers[id] != nil {
		m.soundPlayers[id].Close()
		m.soundPlayers[id] = nil
	}

	player, err := m.createPlayerFromData(data)
	if err != nil {
		return fmt.Errorf("failed to create sound effect player: %w", err)
	}

	m.soundPlayers[id] = player
	m.soundPlayers[id].SetVolume(m.seVolume)

	log.Printf("Sound effect loaded successfully: %s (ID: %d)", filename, id)
	return nil
}

// PlaySoundEffect plays a sound effect
func (m *Manager) PlaySoundEffect(id int) {
	if m.muted || id < 0 || id >= SndSize {
		return
	}

	if m.soundPlayers[id] != nil {
		// Set volume
		m.soundPlayers[id].SetVolume(m.seVolume)

		// Rewind and play
		m.soundPlayers[id].Rewind()
		m.soundPlayers[id].Play()

		log.Printf("Playing sound effect ID: %d", id)
	} else {
		log.Printf("Warning: Sound effect ID %d not loaded", id)
	}
}

// PlaySoundEffectByName loads and plays a sound effect by filename
func (m *Manager) PlaySoundEffectByName(filename string) error {
	// Create a temporary player for one-shot sounds
	var data []byte
	var err error

	if strings.Contains(filename, ".gpk") || strings.Contains(filename, "::") {
		// GPK format
		parts := strings.Split(filename, "::")
		if len(parts) != 2 {
			return fmt.Errorf("invalid GPK path format: %s", filename)
		}
		data, err = m.loadAudioFromGPK(parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to load sound effect from GPK: %w", err)
		}
	} else {
		// Regular file
		data, err = m.loadAudioFromFile(filename)
		if err != nil {
			return fmt.Errorf("failed to load sound effect from file: %w", err)
		}
	}

	if m.muted {
		return nil
	}

	player, err := m.createPlayerFromData(data)
	if err != nil {
		return fmt.Errorf("failed to create temporary sound player: %w", err)
	}

	player.SetVolume(m.seVolume)
	player.Play()

	log.Printf("Playing sound effect: %s", filename)
	return nil
}

// PlaySystemSound plays a system sound by name
func (m *Manager) PlaySystemSound(soundName string) {
	if id, found := GetSystemSoundID(soundName); found {
		m.PlaySoundEffect(id)
	} else {
		log.Printf("Warning: Unknown system sound: %s", soundName)
	}
}

// PlaySystemSoundByID plays a system sound by ID (for backward compatibility)
func (m *Manager) PlaySystemSoundByID(soundID int) {
	m.PlaySoundEffect(soundID)
}

// GetLoadedSoundEffects returns a list of loaded sound effect names
func (m *Manager) GetLoadedSoundEffects() []string {
	var loaded []string
	for i, sound := range m.loadedSounds {
		if sound != nil {
			loaded = append(loaded, fmt.Sprintf("%s (ID: %d)", GetSystemSoundName(i), i))
		}
	}
	return loaded
}
