// OGG File Format - Types and Constants
package main

// OGG analysis status constants
type OGGStatus int

const (
	OGGStatusValid OGGStatus = iota
	OGGStatusMissingFirstOggS
	OGGStatusCorruptedHeader
	OGGStatusNoOggS
	OGGStatusUnknown
)

// OGGAnalysis contains the results of analyzing an OGG file structure
type OGGAnalysis struct {
	Status            OGGStatus
	Description       string
	FirstOggSPos      int
	SecondOggSPos     int
	HasValidStructure bool
	VorbisPos         int
}

// String returns a human-readable description of the OGG status
func (s OGGStatus) String() string {
	switch s {
	case OGGStatusValid:
		return "Valid"
	case OGGStatusMissingFirstOggS:
		return "Missing First OggS"
	case OGGStatusCorruptedHeader:
		return "Corrupted Header"
	case OGGStatusNoOggS:
		return "No OggS Found"
	case OGGStatusUnknown:
		return "Unknown"
	default:
		return "Invalid Status"
	}
}
