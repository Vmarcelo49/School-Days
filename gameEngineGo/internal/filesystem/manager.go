package filesystem

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Manager represents the game filesystem with GPK package support
type Manager struct {
	rootDir  string
	archives []*GPK
}

// FileInfo represents information about a file in the filesystem
type FileInfo struct {
	Name     string
	IsInGPK  bool
	GPKName  string
	FullPath string
}

// NewManager creates a new filesystem manager
func NewManager(rootDir string) *Manager {
	return &Manager{
		rootDir:  rootDir,
		archives: make([]*GPK, 0),
	}
}

// Init initializes the filesystem manager
func (m *Manager) Init() error {
	// Ensure root directory exists
	if _, err := os.Stat(m.rootDir); os.IsNotExist(err) {
		return fmt.Errorf("root directory does not exist: %s", m.rootDir)
	}

	// Mount GPK archives from packs directory
	err := m.findAndMountArchives()
	if err != nil {
		// Log the error but continue - we can still work with loose files
		fmt.Printf("Warning: Failed to mount some archives: %v\n", err)
	}

	fmt.Printf("Filesystem initialized with %d GPK archives\n", len(m.archives))
	return nil
}

// findAndMountArchives searches for and mounts GPK files
func (m *Manager) findAndMountArchives() error {
	packsDir := filepath.Join(m.rootDir, "packs")

	// Check if packs directory exists
	if _, err := os.Stat(packsDir); os.IsNotExist(err) {
		fmt.Printf("No packs directory found at %s\n", packsDir)
		return nil
	}

	// Walk through the packs directory
	err := filepath.Walk(packsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and only process .GPK files
		if info.IsDir() || !strings.HasSuffix(strings.ToUpper(info.Name()), ".GPK") {
			return nil
		}

		// Try to mount the GPK file
		gpk, mountErr := NewGPK(path)
		if mountErr != nil {
			fmt.Printf("Warning: Failed to mount GPK %s: %v\n", info.Name(), mountErr)
			return nil // Continue with other files
		}

		m.archives = append(m.archives, gpk)
		fmt.Printf("Mounted GPK: %s (%d files)\n", info.Name(), len(gpk.GetEntries()))
		return nil
	})
	return err
}

// Open opens a file, checking archives first, then filesystem
func (m *Manager) Open(filename string) (io.ReadCloser, error) {
	// First check mounted GPK archives
	for _, gpk := range m.archives {
		if entry, found := gpk.FindEntry(filename); found {
			data, err := gpk.ExtractFile(entry)
			if err != nil {
				continue // Try next archive or filesystem
			}
			return &ByteReadCloser{data: data}, nil
		}
	}

	// Try to open from regular filesystem
	fullPath := m.getFullPath(filename)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	return file, nil
}

// ByteReadCloser wraps a byte slice to implement io.ReadCloser
type ByteReadCloser struct {
	data   []byte
	offset int
}

func (b *ByteReadCloser) Read(p []byte) (n int, err error) {
	if b.offset >= len(b.data) {
		return 0, io.EOF
	}

	n = copy(p, b.data[b.offset:])
	b.offset += n
	return n, nil
}

func (b *ByteReadCloser) Close() error {
	return nil
}

// Exists checks if a file exists in archives or filesystem
func (m *Manager) Exists(filename string) bool {
	// Check mounted GPK archives first
	for _, gpk := range m.archives {
		if _, found := gpk.FindEntry(filename); found {
			return true
		}
	}

	// Check regular filesystem
	fullPath := m.getFullPath(filename)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ReadFile reads an entire file into memory
func (m *Manager) ReadFile(filename string) ([]byte, error) {
	file, err := m.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// getFullPath constructs the full filesystem path for a file
func (m *Manager) getFullPath(filename string) string {
	// Handle different path separators
	cleanFilename := strings.ReplaceAll(filename, "\\", string(filepath.Separator))
	cleanFilename = strings.ReplaceAll(cleanFilename, "/", string(filepath.Separator))

	return filepath.Join(m.rootDir, cleanFilename)
}

// ListDirectory lists files in a directory
func (m *Manager) ListDirectory(dirPath string) ([]string, error) {
	fullPath := m.getFullPath(dirPath)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files, nil
}

// GetRootDir returns the root directory
func (m *Manager) GetRootDir() string {
	return m.rootDir
}

// Close closes all mounted archives
func (m *Manager) Close() error {
	for _, gpk := range m.archives {
		gpk.Close()
	}
	m.archives = m.archives[:0]
	return nil
}

// ListFiles returns all available files from both filesystem and archives
func (m *Manager) ListFiles() []FileInfo {
	var files []FileInfo

	// Add files from GPK archives
	for _, gpk := range m.archives {
		for _, entry := range gpk.GetEntries() {
			files = append(files, FileInfo{
				Name:     entry.Name,
				IsInGPK:  true,
				GPKName:  filepath.Base(gpk.fileName),
				FullPath: entry.Name,
			})
		}
	}

	return files
}

// GetArchiveCount returns the number of mounted GPK archives
func (m *Manager) GetArchiveCount() int {
	return len(m.archives)
}
