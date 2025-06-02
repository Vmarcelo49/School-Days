package audio

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Sound effect IDs based on the original C++ engine
const (
	SndTitle  = 0
	SndSystem = 1
	SndCancel = 2
	SndSelect = 3
	SndClick  = 4
	SndUp     = 5
	SndDown   = 6
	SndView   = 7
	SndOpen   = 8
	SndSize   = 9
)

// Manager handles all audio playback
type Manager struct {
	context      *audio.Context
	bgmPlayer    *audio.Player
	soundPlayers [SndSize]*audio.Player

	bgmVolume   float64
	seVolume    float64
	voiceVolume float64
	muted       bool
}

// NewManager creates a new audio manager
func NewManager() *Manager {
	return &Manager{
		bgmVolume:   1.0,
		seVolume:    1.0,
		voiceVolume: 1.0,
		muted:       false,
	}
}

// Init initializes the audio system
func (m *Manager) Init() error {
	var err error

	// Create audio context with standard sample rate
	m.context = audio.NewContext(44100)

	log.Println("Audio manager initialized")
	return err
}

// LoadBGM loads background music from file
func (m *Manager) LoadBGM(filename string) error {
	// TODO: Implement actual file loading from filesystem
	// For now, this is a placeholder
	log.Printf("Loading BGM: %s", filename)
	return nil
}

// PlayBGM plays background music
func (m *Manager) PlayBGM(loops int) error {
	if m.muted {
		return nil
	}

	if m.bgmPlayer != nil {
		// Set volume
		m.bgmPlayer.SetVolume(m.bgmVolume)

		// Play the music
		m.bgmPlayer.Rewind()
		m.bgmPlayer.Play()

		log.Println("Playing BGM")
	}

	return nil
}

// StopBGM stops background music
func (m *Manager) StopBGM() {
	if m.bgmPlayer != nil {
		m.bgmPlayer.Pause()
		log.Println("Stopped BGM")
	}
}

// LoadSoundEffect loads a sound effect
func (m *Manager) LoadSoundEffect(filename string, id int) error {
	if id < 0 || id >= SndSize {
		return nil
	}

	// TODO: Implement actual file loading from filesystem
	// For now, this is a placeholder
	log.Printf("Loading sound effect: %s (ID: %d)", filename, id)
	return nil
}

// PlaySoundEffect plays a sound effect
func (m *Manager) PlaySoundEffect(id int) {
	if m.muted || id < 0 || id >= SndSize {
		return
	}

	if m.soundPlayers[id] != nil {
		m.soundPlayers[id].SetVolume(m.seVolume)
		m.soundPlayers[id].Rewind()
		m.soundPlayers[id].Play()

		log.Printf("Playing sound effect ID: %d", id)
	}
}

// PlaySystemSound plays a system sound (like menu navigation sounds)
func (m *Manager) PlaySystemSound(soundType int) {
	switch soundType {
	case SndSelect:
		m.PlaySoundEffect(SndSelect)
	case SndCancel:
		m.PlaySoundEffect(SndCancel)
	case SndClick:
		m.PlaySoundEffect(SndClick)
	case SndUp:
		m.PlaySoundEffect(SndUp)
	case SndDown:
		m.PlaySoundEffect(SndDown)
	default:
		m.PlaySoundEffect(SndSystem)
	}
}

// SetBGMVolume sets the background music volume (0.0 to 1.0)
func (m *Manager) SetBGMVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}

	m.bgmVolume = volume

	if m.bgmPlayer != nil {
		m.bgmPlayer.SetVolume(volume)
	}
}

// SetSEVolume sets the sound effects volume (0.0 to 1.0)
func (m *Manager) SetSEVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}

	m.seVolume = volume
}

// SetVoiceVolume sets the voice volume (0.0 to 1.0)
func (m *Manager) SetVoiceVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}

	m.voiceVolume = volume
}

// SetMuted sets the mute state
func (m *Manager) SetMuted(muted bool) {
	m.muted = muted

	if muted {
		m.StopBGM()
	}
}

// IsMuted returns whether audio is muted
func (m *Manager) IsMuted() bool {
	return m.muted
}

// Update updates the audio system (called every frame)
func (m *Manager) Update() {
	// Update audio state if needed
	// This is where we could handle audio streaming, fading, etc.
}

// Cleanup cleans up audio resources
func (m *Manager) Cleanup() {
	if m.bgmPlayer != nil {
		m.bgmPlayer.Close()
	}

	for i := range m.soundPlayers {
		if m.soundPlayers[i] != nil {
			m.soundPlayers[i].Close()
		}
	}

	log.Println("Audio manager cleaned up")
}
