package settings

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Config holds all engine configuration
type Config struct {
	ScreenWidth  int     `json:"screen_width"`
	ScreenHeight int     `json:"screen_height"`
	Fullscreen   bool    `json:"fullscreen"`
	VSync        bool    `json:"vsync"`
	BGMVolume    float64 `json:"bgm_volume"`
	SFXVolume    float64 `json:"sfx_volume"`
	VoiceVolume  float64 `json:"voice_volume"`
	AssetsPath   string  `json:"assets_path"`
	DebugMode    bool    `json:"debug_mode"`
	Language     string  `json:"language"`
	TextSpeed    int     `json:"text_speed"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ScreenWidth:  800,
		ScreenHeight: 600,
		Fullscreen:   false,
		VSync:        true,
		BGMVolume:    0.8,
		SFXVolume:    0.8,
		VoiceVolume:  0.8,
		AssetsPath:   "./assets",
		DebugMode:    true,
		Language:     "en",
		TextSpeed:    3,
	}
}

// Manager handles configuration loading and saving
type Manager struct {
	config     *Config
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		config:     DefaultConfig(),
		configPath: configPath,
	}
}

// Load loads configuration from file
func (m *Manager) Load() error {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		log.Printf("Config file %s not found, using defaults", m.configPath)
		return m.Save()
	}

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	log.Printf("Loaded configuration from %s", m.configPath)
	return nil
}

// Save saves configuration to file
func (m *Manager) Save() error {
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	log.Printf("Saved configuration to %s", m.configPath)
	return nil
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}
