package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem represents the game filesystem with GPK packages
type FileSystem struct {
	root string
	gpks []*GPK
}

// NewFileSystem creates a new filesystem instance
func NewFileSystem(gameRoot string) (*FileSystem, error) {
	fs := &FileSystem{
		root: gameRoot,
		gpks: make([]*GPK, 0),
	}

	err := fs.findArchives()
	if err != nil {
		return nil, fmt.Errorf("failed to find archives: %w", err)
	}

	return fs, nil
}

// findArchives locates and mounts all GPK files in the packs directory
func (fs *FileSystem) findArchives() error {
	packsRoot := filepath.Join(fs.root, "packs")

	err := filepath.Walk(packsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToUpper(filepath.Ext(info.Name())) == ".GPK" {
			err := fs.mountGPK(path)
			if err != nil {
				fmt.Printf("Warning: Failed to mount GPK %s: %v\n", path, err)
			} else {
				fmt.Printf("Mounted package: %s\n", info.Name())
			}
		}
		return nil
	})

	return err
}

// mountGPK loads a GPK file and adds it to the filesystem
func (fs *FileSystem) mountGPK(fileName string) error {
	gpk := NewGPK()
	err := gpk.Load(fileName)
	if err != nil {
		return err
	}

	fs.gpks = append(fs.gpks, gpk)
	return nil
}

// UnpackAll unpacks all mounted GPK files
func (fs *FileSystem) UnpackAll() error {
	for _, gpk := range fs.gpks {
		fmt.Printf("Unpacking: %s\n", gpk.GetName())
		outputDir := filepath.Join(fs.root, gpk.GetName())
		err := gpk.UnpackAll(outputDir)
		if err != nil {
			return fmt.Errorf("failed to unpack %s: %w", gpk.GetName(), err)
		}
	}
	return nil
}

// Open opens a file from the filesystem (either from disk or GPK)
func (fs *FileSystem) Open(filename string) (*GPKFile, error) {
	// First check if file exists on disk
	fullPath := filepath.Join(fs.root, filename)
	if _, err := os.Stat(fullPath); err == nil {
		return NewGPKFileFromDisk(fullPath)
	}

	// Extract package name and file path
	parts := strings.SplitN(filename, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filename format: %s", filename)
	}

	pkg := parts[0]
	filePath := parts[1]
	normalizedName := normalizeName(pkg, filePath)

	// Search in GPK packages
	for _, gpk := range fs.gpks {
		if strings.EqualFold(gpk.GetName(), pkg) {
			return gpk.Open(normalizedName)
		}
	}

	return nil, fmt.Errorf("file not found: %s", filename)
}

// List lists files matching a pattern in a specific package
func (fs *FileSystem) List(mask string) ([]string, error) {
	parts := strings.SplitN(mask, string(filepath.Separator), 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid mask format: %s", mask)
	}

	pkg := parts[0]
	pattern := parts[1]

	for _, gpk := range fs.gpks {
		if strings.EqualFold(gpk.GetName(), pkg) {
			return gpk.List(pattern), nil
		}
	}

	return []string{}, nil
}

// GetRoot returns the root directory of the filesystem
func (fs *FileSystem) GetRoot() string {
	return fs.root
}

// normalizeName normalizes file names based on package type
func normalizeName(pkg, name string) string {
	pkgUpper := strings.ToUpper(pkg)

	if strings.HasPrefix(pkgUpper, "SYSSE") || strings.HasPrefix(pkgUpper, "SE") || strings.HasPrefix(pkgUpper, "VOICE") {
		return name + ".ogg"
	} else if strings.HasPrefix(pkgUpper, "BGM") {
		return name + "_loop.ogg"
	} else if strings.HasPrefix(pkgUpper, "EVENT") {
		return name + ".PNG"
	}

	return name
}

// NormalizeName normalizes a file name using the filesystem's package information
func (fs *FileSystem) NormalizeName(name string) string {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 {
		return name
	}
	return normalizeName(parts[0], parts[1])
}
