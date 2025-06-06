package audio

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

// Audio constants
const (
	SampleRate = 44100
)

// AudioFile represents an audio file that can be loaded from disk or GPK
type AudioFile struct {
	Name      string // Display name
	Path      string // Full path for disk files or GPK package name
	Data      []byte // Cached audio data
	IsFromGPK bool   // Whether this file is from a GPK package
	GPKEntry  string // Entry name within GPK (if IsFromGPK is true)
}

// AudioLoadConfig represents configuration for loading audio files
type AudioLoadConfig struct {
	PreloadSystemSounds bool
	AudioDirectory      string
	GPKPackages         []string
	VolumeSettings      VolumeSettings
}

// VolumeSettings represents volume configuration
type VolumeSettings struct {
	BGMVolume   float64
	SEVolume    float64
	VoiceVolume float64
	Muted       bool
}

// DefaultAudioConfig returns default audio configuration
func DefaultAudioConfig() AudioLoadConfig {
	return AudioLoadConfig{
		PreloadSystemSounds: true,
		AudioDirectory:      "assets/audio",
		GPKPackages:         []string{},
		VolumeSettings: VolumeSettings{
			BGMVolume:   1.0,
			SEVolume:    1.0,
			VoiceVolume: 1.0,
			Muted:       false,
		},
	}
}
