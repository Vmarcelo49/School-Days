package filesystem

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles file system operations and archive loading
type Manager struct {
	rootDir   string
	extension string
	archives  []Archive
}

// Archive represents a GPK archive file
type Archive interface {
	Open(filename string) (io.ReadCloser, error)
	Exists(filename string) bool
	Close() error
}

// NewManager creates a new filesystem manager
func NewManager(rootDir string) *Manager {
	return &Manager{
		rootDir:   rootDir,
		extension: "",
		archives:  make([]Archive, 0),
	}
}

// Init initializes the filesystem
func (m *Manager) Init() error {
	// Verify root directory exists
	if _, err := os.Stat(m.rootDir); os.IsNotExist(err) {
		return fmt.Errorf("root directory does not exist: %s", m.rootDir)
	}

	log.Printf("Filesystem initialized with root: %s", m.rootDir)
	return nil
}

// SetExtension sets the file extension for archive files
func (m *Manager) SetExtension(ext string) {
	m.extension = ext
	log.Printf("File extension set to: %s", ext)
}

// MountArchive mounts a GPK archive file
func (m *Manager) MountArchive(filename string) error {
	// TODO: Implement GPK archive mounting
	// For now, this is a placeholder that logs the operation
	log.Printf("Mounting archive: %s", filename)

	// In a real implementation, this would:
	// 1. Open the GPK file
	// 2. Parse its header and file table
	// 3. Add it to the archives list

	return nil
}

// MountGPK mounts a GPK archive for file access
func (m *Manager) MountGPK(filename string) error {
	// TODO: Implement GPK archive mounting
	log.Printf("GPK mounting not yet implemented for: %s", filename)
	return fmt.Errorf("GPK archive support not implemented")
}

// Open opens a file, checking archives first, then filesystem
func (m *Manager) Open(filename string) (io.ReadCloser, error) {
	// First, try to find the file in mounted archives
	for _, archive := range m.archives {
		if archive.Exists(filename) {
			return archive.Open(filename)
		}
	}

	// If not found in archives, try the regular filesystem
	fullPath := m.getFullPath(filename)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	return file, nil
}

// Exists checks if a file exists in archives or filesystem
func (m *Manager) Exists(filename string) bool {
	// Check archives first
	for _, archive := range m.archives {
		if archive.Exists(filename) {
			return true
		}
	}

	// Check filesystem
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
	for _, archive := range m.archives {
		if err := archive.Close(); err != nil {
			log.Printf("Error closing archive: %v", err)
		}
	}

	m.archives = nil
	log.Println("Filesystem manager closed")
	return nil
}

// Archive wrapper placeholder for future GPK support
type ArchiveWrapper struct {
	// TODO: Implement when GPK support is added
}

func (w *ArchiveWrapper) Open(filename string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("archive support not implemented")
}

func (w *ArchiveWrapper) Exists(filename string) bool {
	return false
}

func (w *ArchiveWrapper) Close() error {
	return nil
}

// bytesReadCloser wraps byte data as ReadCloser
type bytesReadCloser struct {
	data   []byte
	offset int
}

func (r *bytesReadCloser) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

func (r *bytesReadCloser) Close() error {
	return nil
}

// LoadFile loads a file from filesystem or archives
