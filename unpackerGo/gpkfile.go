package main

import (
	"fmt"
	"io"
	"os"
)

// GPKFile represents a file that can be read from either disk or a GPK package
type GPKFile struct {
	realFile *os.File
	isPKG    bool
	entry    *GPKEntryHeader
	position int64
	gpkFile  *os.File
}

// NewGPKFileFromDisk creates a GPKFile from a real file on disk
func NewGPKFileFromDisk(filename string) (*GPKFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return &GPKFile{
		realFile: file,
		isPKG:    false,
	}, nil
}

// NewGPKFileFromPackage creates a GPKFile from a GPK package entry
func NewGPKFileFromPackage(entry *GPKEntryHeader, gpkFileName string) (*GPKFile, error) {
	gpkFile, err := os.Open(gpkFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPK file: %w", err)
	}

	_, err = gpkFile.Seek(int64(entry.Offset), 0)
	if err != nil {
		gpkFile.Close()
		return nil, fmt.Errorf("failed to seek to entry offset: %w", err)
	}

	return &GPKFile{
		isPKG:    true,
		entry:    entry,
		position: 0,
		gpkFile:  gpkFile,
	}, nil
}

// Read reads data from the file
func (gf *GPKFile) Read(data []byte) (int, error) {
	if gf.isPKG {
		// Read from GPK package
		maxLen := int64(len(data))
		remaining := int64(gf.entry.ComprLen) - gf.position
		if maxLen > remaining {
			maxLen = remaining
		}

		if maxLen <= 0 {
			return 0, io.EOF
		}

		n, err := gf.gpkFile.Read(data[:maxLen])
		gf.position += int64(n)
		return n, err
	} else {
		// Read from real file
		return gf.realFile.Read(data)
	}
}

// Seek seeks to a position in the file
func (gf *GPKFile) Seek(offset int64, whence int) (int64, error) {
	if gf.isPKG {
		switch whence {
		case io.SeekStart:
			gf.position = offset
		case io.SeekCurrent:
			gf.position += offset
		case io.SeekEnd:
			gf.position = int64(gf.entry.ComprLen) + offset
		}

		// Ensure position is within bounds
		if gf.position < 0 {
			gf.position = 0
		}
		if gf.position > int64(gf.entry.ComprLen) {
			gf.position = int64(gf.entry.ComprLen)
		}

		// Seek in the underlying GPK file
		_, err := gf.gpkFile.Seek(int64(gf.entry.Offset)+gf.position, io.SeekStart)
		return gf.position, err
	} else {
		return gf.realFile.Seek(offset, whence)
	}
}

// Size returns the size of the file
func (gf *GPKFile) Size() int64 {
	if gf.isPKG {
		return int64(gf.entry.ComprLen)
	} else {
		stat, err := gf.realFile.Stat()
		if err != nil {
			return 0
		}
		return stat.Size()
	}
}

// Close closes the file
func (gf *GPKFile) Close() error {
	if gf.isPKG {
		if gf.gpkFile != nil {
			return gf.gpkFile.Close()
		}
	} else {
		if gf.realFile != nil {
			return gf.realFile.Close()
		}
	}
	return nil
}

// ReadAll reads the entire file content
func (gf *GPKFile) ReadAll() ([]byte, error) {
	size := gf.Size()
	data := make([]byte, size)

	if gf.isPKG {
		// Reset position for GPK files
		gf.position = 0
		_, err := gf.gpkFile.Seek(int64(gf.entry.Offset), io.SeekStart)
		if err != nil {
			return nil, err
		}
	}

	_, err := io.ReadFull(gf, data)
	return data, err
}

// AtEnd returns true if at end of file
func (gf *GPKFile) AtEnd() bool {
	if gf.isPKG {
		return gf.position >= int64(gf.entry.ComprLen)
	} else {
		currentPos, _ := gf.realFile.Seek(0, io.SeekCurrent)
		size := gf.Size()
		return currentPos >= size
	}
}

// Position returns the current position in the file
func (gf *GPKFile) Position() int64 {
	if gf.isPKG {
		return gf.position
	} else {
		pos, _ := gf.realFile.Seek(0, io.SeekCurrent)
		return pos
	}
}
