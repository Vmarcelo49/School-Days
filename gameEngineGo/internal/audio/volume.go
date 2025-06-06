package audio

import (
	"log"
)

// SetBGMVolume sets the background music volume (0.0 to 1.0)
func (m *Manager) SetBGMVolume(volume float64) {
	if volume < 0.0 {
		volume = 0.0
	} else if volume > 1.0 {
		volume = 1.0
	}

	m.bgmVolume = volume
	if m.bgmPlayer != nil && !m.muted {
		m.bgmPlayer.SetVolume(volume)
	}

	log.Printf("BGM volume set to: %.2f", volume)
}

// SetSEVolume sets the sound effects volume (0.0 to 1.0)
func (m *Manager) SetSEVolume(volume float64) {
	if volume < 0.0 {
		volume = 0.0
	} else if volume > 1.0 {
		volume = 1.0
	}

	m.seVolume = volume
	if !m.muted {
		for i := range m.soundPlayers {
			if m.soundPlayers[i] != nil {
				m.soundPlayers[i].SetVolume(volume)
			}
		}
	}

	log.Printf("SE volume set to: %.2f", volume)
}

// SetVoiceVolume sets the voice volume (0.0 to 1.0)
func (m *Manager) SetVoiceVolume(volume float64) {
	if volume < 0.0 {
		volume = 0.0
	} else if volume > 1.0 {
		volume = 1.0
	}

	m.voiceVolume = volume
	// TODO: Apply to voice players when implemented

	log.Printf("Voice volume set to: %.2f", volume)
}

// SetMuted sets the mute state for all audio
func (m *Manager) SetMuted(muted bool) {
	m.muted = muted

	if muted {
		// Mute all players
		if m.bgmPlayer != nil {
			m.bgmPlayer.SetVolume(0.0)
		}
		for i := range m.soundPlayers {
			if m.soundPlayers[i] != nil {
				m.soundPlayers[i].SetVolume(0.0)
			}
		}
		log.Println("Audio muted")
	} else {
		// Restore volumes
		if m.bgmPlayer != nil {
			m.bgmPlayer.SetVolume(m.bgmVolume)
		}
		for i := range m.soundPlayers {
			if m.soundPlayers[i] != nil {
				m.soundPlayers[i].SetVolume(m.seVolume)
			}
		}
		log.Println("Audio unmuted")
	}
}

// IsMuted returns the current mute state
func (m *Manager) IsMuted() bool {
	return m.muted
}
