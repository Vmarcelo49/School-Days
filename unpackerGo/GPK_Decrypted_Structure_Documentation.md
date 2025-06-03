# GPK Decrypted File Structure Documentation

## Overview

This document describes the internal structure of GPK (Game Package) files after they have been decrypted using the `-decryptOnly` functionality. Understanding this structure is crucial for developers working with GPK archives or implementing custom tools.

## Table of Contents

1. [File Layout Overview](#file-layout-overview)
2. [Encryption Details](#encryption-details)
3. [File Structure Components](#file-structure-components)
4. [PIDX (Package Index) Section](#pidx-package-index-section)
5. [Entry Header Structure](#entry-header-structure)
6. [File Data Section](#file-data-section)
7. [Signature Section](#signature-section)
8. [Decryption Process](#decryption-process)
9. [Working with Decrypted Files](#working-with-decrypted-files)
10. [Technical Implementation Notes](#technical-implementation-notes)

---

## File Layout Overview

A decrypted GPK file follows this general structure from beginning to end:

```
[File Data Section]
[PIDX Section (Compressed & Encrypted)]
[Signature Section]
```

**Key Points:**
- File data comes first, stored sequentially
- PIDX (Package Index) is at the end, before the signature
- The signature is always the last 32 bytes
- All sections except file data are encrypted in the original file

---

## Encryption Details

### Cipher Algorithm
GPK files use a simple XOR cipher with a 16-byte repeating key:

```
Cipher Key (Hex): 82 EE 1D B3 57 E9 2C C2 2F 54 7B 10 4C 9A 75 49
```

### Encryption Process
- Each byte is XORed with the corresponding cipher key byte
- The key repeats every 16 bytes: `data[i] ^= cipherKey[i % 16]`
- **Important**: The file data section is typically NOT encrypted, only metadata sections

---

## File Structure Components

### 1. File Data Section (Unencrypted)
- **Location**: Beginning of file
- **Content**: Actual game files stored sequentially
- **Encryption**: Usually unencrypted (raw file data)
- **Format**: Files are stored as-is, maintaining their original headers and format

### 2. PIDX Section (Encrypted & Compressed)
- **Location**: End of file, before signature
- **Content**: Package index with file entries and metadata
- **Encryption**: XOR encrypted
- **Compression**: zlib compressed after decryption
- **Size**: Variable, specified in signature

### 3. Signature Section (Encrypted)
- **Location**: Last 32 bytes of file
- **Content**: File signatures and PIDX length
- **Encryption**: XOR encrypted
- **Format**: Fixed 32-byte structure

---

## PIDX (Package Index) Section

The PIDX section contains the file directory structure and metadata. After decryption and decompression, it contains:

### Structure Format
```
[FileEntry1][FileEntry2][FileEntry3]...[FileEntryN]
```

### File Entry Format
Each file entry consists of:

1. **Filename Length** (2 bytes, little-endian)
   - `uint16` specifying UTF-16LE filename length in bytes

2. **Filename** (variable length)
   - UTF-16LE encoded filename
   - Length specified by previous field

3. **Entry Header** (23 bytes, fixed size)
   - Contains file metadata and offset information

---

## Entry Header Structure

Each file entry has a 23-byte header with the following layout:

```go
type GPKEntryHeader struct {
    SubVersion   uint16  // Bytes 0-1:  Always 0x0000
    Version      uint16  // Bytes 2-3:  Always 0x0000  
    Zero         uint16  // Bytes 4-5:  Always 0x0000
    Offset       uint32  // Bytes 6-9:  File data offset in GPK
    ComprLen     uint32  // Bytes 10-13: Compressed file size
    Reserved     [4]byte // Bytes 14-17: Always 0x20202020 (ASCII spaces)
    UncomprLen   uint32  // Bytes 18-21: Uncompressed size (always 0)
    ComprHeadLen uint8   // Byte 22:     Variable compression header length
}
```

### Field Descriptions

- **SubVersion/Version/Zero**: Always 0, used for format versioning
- **Offset**: Absolute byte offset where file data begins in GPK
- **ComprLen**: Size of the file data in bytes
- **Reserved**: Padding bytes, always contains four ASCII space characters (0x20)
- **UncomprLen**: Always 0 in this format (uncompressed size unknown)
- **ComprHeadLen**: Length of any compression header (often 0)

---

## File Data Section

### Organization
- Files are stored sequentially in the order they appear in PIDX
- Each file begins at the offset specified in its entry header
- Files maintain their original format and headers (OGG, PNG, etc.)

### File Types Commonly Found
- **Audio Files**: OGG Vorbis format
- **Image Files**: PNG format  
- **Text Files**: Various text formats
- **Binary Data**: Game-specific formats

### Extraction Process
1. Read entry header to get offset and size
2. Seek to the specified offset in the GPK file
3. Read exactly `ComprLen` bytes
4. Write data directly to output file (no decompression needed)

---

## Signature Section

The signature is always the last 32 bytes of the file:

```go
type GPKSignature struct {
    Sig0       [12]byte  // "STKFile0PIDX" identifier
    PidxLength uint32    // Size of PIDX section in bytes
    Sig1       [16]byte  // "STKFile0PACKFILE" identifier
}
```

### Signature Identifiers
- **Sig0**: `"STKFile0PIDX"` (12 bytes)
- **Sig1**: `"STKFile0PACKFILE"` (16 bytes)
- **PidxLength**: Size of the compressed PIDX section

---

## Decryption Process

### Full File Decryption
When using `-decryptOnly`, the entire file is decrypted:

1. **Read** the original encrypted GPK file in chunks
2. **Apply** XOR decryption to each byte using the cipher key
3. **Write** decrypted data to new file with `_decrypted.gpk` suffix

### Result
- All metadata sections become readable
- PIDX section becomes a decrypted zlib stream
- Signature section becomes readable text
- File data may become corrupted if it was originally unencrypted

---

## Working with Decrypted Files

### Limitations
- **Cannot be read by standard GPK parser**: The parser expects encrypted metadata
- **File data may be corrupted**: If files were originally unencrypted, XOR will corrupt them
- **For analysis only**: Primary use is reverse engineering and debugging

### Analysis Uses
- **Structure Investigation**: Understanding GPK format
- **Debugging**: Examining file organization
- **Tool Development**: Creating custom extractors
- **Format Documentation**: Studying the archive format

### Reading Decrypted PIDX
To manually parse a decrypted PIDX:

1. **Locate PIDX**: Use signature to find PIDX offset
2. **Decrypt**: Apply XOR cipher to PIDX section  
3. **Decompress**: Use zlib to decompress
4. **Parse**: Read entries sequentially

---

## Technical Implementation Notes

### Byte Ordering
- All multi-byte integers use **little-endian** format
- UTF-16 strings use **little-endian** encoding (UTF-16LE)

### Memory Considerations
- Large GPK files (70+ MB) require chunked processing
- PIDX section is typically small (few KB) and can be loaded entirely
- File data should be streamed for memory efficiency

### Error Handling
- Invalid signatures indicate corruption or wrong file type
- PIDX decompression failure suggests encryption/compression issues
- Filename length validation prevents buffer overflows

### Compatibility Notes
- This format appears specific to School Days HQ and related games
- Different games may use variations of this structure
- Always validate signatures before processing

---

## Example: Manual Structure Analysis

To analyze a decrypted GPK file manually:

```bash
# 1. Decrypt the GPK file
.\unpacker.exe -decryptOnly source.gpk

# 2. Examine the signature (last 32 bytes)
# Should contain "STKFile0PIDX" and "STKFile0PACKFILE"

# 3. Use signature to locate PIDX section
# PIDX starts at: (file_size - 32 - pidx_length)

# 4. Extract and analyze PIDX entries
# Each entry: [filename_len][filename][23-byte header]
```

### File Offset Calculation
```
PIDX Offset = File Size - 32 - PIDX Length
File Data Ends = PIDX Offset
```

---

## Conclusion

Understanding the decrypted GPK structure enables:
- **Custom tool development** for GPK archives
- **Debugging and analysis** of archive contents  
- **Format reverse engineering** for research
- **Educational purposes** for learning archive formats

The decryption feature provides valuable insight into the GPK format while maintaining the integrity of the original extraction functionality.

---

*Document Version: 1.0*  
*Last Updated: June 2, 2025*  
*Compatible with: School Days HQ GPK Format*
