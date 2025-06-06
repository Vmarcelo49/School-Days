package audio

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Manager handles all audio playback - supports the same interface as C++ version
type Manager struct {
	context      *audio.Context
	bgmPlayer    *audio.Player
	soundPlayers [SndSize]*audio.Player

	// Volume controls (0.0 to 1.0)
	bgmVolume   float64
	seVolume    float64
	voiceVolume float64
	muted       bool

	// Audio file cache for quick access
	loadedBGM    *AudioFile
	loadedSounds [SndSize]*AudioFile

	// GPK integration - will be set by the filesystem manager
	gpkManager interface{} // Will be set to filesystem.Manager
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
	m.context = audio.NewContext(SampleRate)

	log.Println("Audio manager initialized")
	return err
}

// SetGPKManager sets the GPK manager for loading audio from packages
func (m *Manager) SetGPKManager(gpkManager interface{}) {
	m.gpkManager = gpkManager
}

// Update updates the audio system (called each frame)
func (m *Manager) Update() error {
	// Handle BGM looping logic here
	if m.bgmPlayer != nil && !m.bgmPlayer.IsPlaying() {
		// BGM finished, implement looping logic if needed
		// This will be expanded in the future for proper looping control
	}

	return nil
}

// Cleanup properly closes all audio resources
func (m *Manager) Cleanup() {
	if m.bgmPlayer != nil {
		m.bgmPlayer.Close()
		m.bgmPlayer = nil
	}

	for i := range m.soundPlayers {
		if m.soundPlayers[i] != nil {
			m.soundPlayers[i].Close()
			m.soundPlayers[i] = nil
		}
	}

	log.Println("Audio manager cleaned up")
}
