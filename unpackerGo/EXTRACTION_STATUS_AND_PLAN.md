# School Days BGM.GPK Extraction Status and Fix Plan

## Project Overview
This document tracks the progress of extracting and validating OGG Vorbis audio files from the School Days game's BGM.GPK archive using a Go-based unpacker.

## Current Status ✅ Partial Success

### What We've Accomplished
1. **Successfully extracted 63 OGG files** from BGM.GPK using the Go unpacker
2. **Fixed OGG container detection** - all files now have valid OGG page headers
3. **Identified the core issue** - extracted files start with Vorbis comment headers (type 3) instead of identification headers (type 1)

### Files Created and Modified

#### Main Extraction Code
- `main.go` - Main extraction program
- `gpk.go` - GPK file format handler (✅ MODIFIED to auto-detect OGG start position)
- `gpkfile.go` - Individual file handling within GPK archives
- `filesystem.go` - File system utilities

#### Analysis and Validation Tools
- `validate_ogg.go` - ✅ OGG Vorbis validation utility (detects packet type issues)
- `hexdump.go` - ✅ Hex data examination utility
- `analyze_ogg_stream.go` - ⚠️ INCOMPLETE OGG stream structure analyzer
- `test_single.go` - Single file extraction tester

#### Documentation
- `oggSpec.txt` - Vorbis specification reference
- `vorbis_packet_types.txt` - Vorbis packet type documentation

#### Extracted Content
- `extracted/SD_BGM/` - 53 background music OGG files
- `extracted/VOCAL/` - 10 vocal track OGG files

### Key Technical Findings

#### ✅ Fixed Issues
1. **Header Offset Detection**: Modified `gpk.go` to automatically search for "OggS" pattern instead of using fixed offsets
2. **OGG Container Validation**: All 63 files now pass OGG page header validation:
   - ✅ Valid "OggS" capture pattern
   - ✅ Valid OGG version (0)
   - ✅ Proper page segment structure

#### ❌ Remaining Issues
1. **Missing Vorbis Identification Header**: All files show packet type 3 (comment header) instead of expected type 1 (identification header)
2. **Incomplete Vorbis Stream**: Files appear to be missing the first header packet required by the Vorbis specification

### Validation Results
```
SUMMARY:
  Total files processed: 63
  Valid OGG Vorbis files: 0
  Invalid files: 63
⚠️  63 files failed validation
```

**Common Error Pattern:**
```
✓ OGG capture pattern: OggS
✓ OGG version: 0
✓ Page segments: 18, Payload size: 4198 bytes
❌ Invalid Vorbis packet type: 3 (expected: 1)
```

## Root Cause Analysis

### Vorbis Stream Structure
According to the Vorbis specification, a proper stream contains three header packets:
1. **Identification Header (type 1)** - Basic codec info ❌ MISSING
2. **Comment Header (type 3)** - Metadata ✅ PRESENT (incorrectly as first packet)
3. **Setup Header (type 5)** - Codec setup ❓ UNKNOWN

### Hypothesis
The GPK extraction is starting from the **second** Vorbis header packet (comment header) instead of the **first** header packet (identification header). This suggests:

1. **Offset miscalculation**: The "OggS" search finds the wrong OGG page
2. **Multi-page headers**: The identification header might be in a preceding OGG page
3. **Custom format**: The game might use a non-standard Vorbis implementation

## Action Plan for Resolution

### Phase 1: Complete Stream Analysis Tool ⚠️ IN PROGRESS
1. **Finish `analyze_ogg_stream.go`** to examine complete OGG page structure
   - Fix compilation error (remove unused `strings` import)
   - Test on extracted files to map entire stream structure
   - Identify if identification header exists in earlier pages

### Phase 2: Enhanced GPK Investigation 🔄 NEXT STEPS
1. **Examine raw GPK data** around OGG boundaries
   ```bash
   go run hexdump.go BGM.GPK <start_offset> <length>
   ```
2. **Search for multiple "OggS" patterns** within each file entry
3. **Create GPK structure analyzer** to understand file entry metadata

### Phase 3: Extraction Algorithm Refinement 🔄 PLANNED
Based on findings from Phases 1-2:

#### Option A: If identification header exists before current extraction point
- Modify `gpk.go` to search for **first** "OggS" occurrence, not the one currently found
- Adjust offset calculation to include all Vorbis headers

#### Option B: If headers are split across multiple file entries  
- Investigate if identification header is stored separately in GPK
- Modify extraction to concatenate related headers

#### Option C: If format uses custom Vorbis variant
- Research School Days-specific audio format documentation
- Create custom validation that accounts for non-standard structure

### Phase 4: Validation and Testing 🔄 PLANNED
1. **Update validation tool** to handle discovered format specifics
2. **Test audio playback** with media players to verify functional extraction
3. **Compare with reference implementations** if available

## Technical Implementation Details

### Current GPK Extraction Logic
```go
// In gpk.go UnpackAll() method
oggPattern := []byte("OggS")
oggIndex := bytes.Index(data, oggPattern)
if oggIndex != -1 {
    // Extract from OggS position - MAY BE WRONG POSITION
    oggData := data[oggIndex:]
    // Write as .ogg file
}
```

### Required Investigation Code
```bash
# Analyze complete OGG stream structure
go run analyze_ogg_stream.go extracted/SD_BGM/SDBGM01_INT.OGG

# Examine raw data around extraction boundaries  
go run hexdump.go BGM.GPK <file_offset> 200
```

## Expected Outcomes

### Success Criteria
- [ ] All 63 files pass Vorbis identification header validation (packet type 1)
- [ ] Files contain complete Vorbis header sequence (types 1, 3, 5)
- [ ] Audio files play correctly in standard media players
- [ ] Extraction process is robust and works for other GPK archives

### Validation Targets
```
SUMMARY:
  Total files processed: 63
  Valid OGG Vorbis files: 63  ← TARGET
  Invalid files: 0           ← TARGET
🎉 ALL FILES ARE VALID OGG VORBIS FILES!  ← GOAL
```

## Files Requiring Attention

### High Priority
1. `analyze_ogg_stream.go` - Fix compilation and complete implementation
2. `gpk.go` - Refine OGG detection and extraction logic  
3. `validate_ogg.go` - Update validation for discovered format specifics

### Medium Priority  
1. Create GPK structure analyzer for better understanding of archive format
2. Implement multi-page OGG analysis capabilities
3. Add audio playback testing utilities

### Low Priority
1. Performance optimizations for large archives
2. Error handling improvements  
3. Cross-platform compatibility testing

## Resources and References
- `oggSpec.txt` - Core Vorbis specification reference
- `vorbis_packet_types.txt` - Packet type documentation  
- [RFC 3533](https://tools.ietf.org/html/rfc3533) - OGG container format
- [Vorbis I Specification](https://xiph.org/vorbis/doc/Vorbis_I_spec.html) - Complete Vorbis codec specification

---
**Last Updated**: Current session  
**Status**: Investigation phase - OGG container extraction successful, Vorbis codec validation requires header sequence fix
