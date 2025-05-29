#include "gpk.h"
#include <fstream>
#include <iostream>
#include <filesystem>
#include <cstring>
#include <algorithm>
#include <sstream>
#include <iomanip>
#include <stdexcept>
#include <zlib.h>

GPK::GPK() {
}

GPK::~GPK() {
}

// Improved UTF-16LE to UTF-8 conversion with surrogate pair support
std::string GPK::utf16le_to_utf8(const uint8_t* data, size_t length_in_chars) {
    std::string result;
    result.reserve(length_in_chars * 2); // Conservative estimate
    
    for (size_t i = 0; i < length_in_chars; i++) {
        uint16_t utf16_char = data[i * 2] | (data[i * 2 + 1] << 8);
        
        // Handle surrogate pairs for characters outside BMP
        if (utf16_char >= 0xD800 && utf16_char <= 0xDBFF) {
            // High surrogate - need to read next character
            if (i + 1 < length_in_chars) {
                uint16_t low_surrogate = data[(i + 1) * 2] | (data[(i + 1) * 2 + 1] << 8);
                if (low_surrogate >= 0xDC00 && low_surrogate <= 0xDFFF) {
                    // Valid surrogate pair
                    uint32_t codepoint = 0x10000 + ((utf16_char & 0x3FF) << 10) + (low_surrogate & 0x3FF);
                    
                    // Convert to UTF-8 (4 bytes)
                    result += static_cast<char>(0xF0 | (codepoint >> 18));
                    result += static_cast<char>(0x80 | ((codepoint >> 12) & 0x3F));
                    result += static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F));
                    result += static_cast<char>(0x80 | (codepoint & 0x3F));
                    
                    i++; // Skip next character as it's part of surrogate pair
                    continue;
                }
            }
        }
        
        // Regular BMP character handling
        if (utf16_char < 0x80) {
            result += static_cast<char>(utf16_char);
        } else if (utf16_char < 0x800) {
            result += static_cast<char>(0xC0 | (utf16_char >> 6));
            result += static_cast<char>(0x80 | (utf16_char & 0x3F));
        } else {
            result += static_cast<char>(0xE0 | (utf16_char >> 12));
            result += static_cast<char>(0x80 | ((utf16_char >> 6) & 0x3F));
            result += static_cast<char>(0x80 | (utf16_char & 0x3F));
        }
    }
    
    return result;
}

// Proper zlib decompression handling Qt's qUncompress format
std::vector<uint8_t> GPK::decompress_data(const std::vector<uint8_t>& compressed_data) {
    // Handle Qt's qUncompress format:
    // - First 4 bytes contain uncompressed size in big-endian
    // - Remaining bytes are standard zlib data
    
    if (compressed_data.size() < 4) {
        return compressed_data;
    }
    
    // Extract uncompressed size (big-endian)
    uint32_t uncompressed_size = (compressed_data[0] << 24) | 
                                (compressed_data[1] << 16) | 
                                (compressed_data[2] << 8) | 
                                compressed_data[3];
    
    if (uncompressed_size == 0) {
        throw std::runtime_error("Invalid uncompressed size in compressed data");
    }
    
    std::vector<uint8_t> result(uncompressed_size);
    uLongf dest_len = uncompressed_size;
    
    int ret = uncompress(result.data(), &dest_len, 
                        compressed_data.data() + 4, 
                        compressed_data.size() - 4);
    
    if (ret != Z_OK) {
        std::string error_msg = "Decompression failed with error code: " + std::to_string(ret);
        if (ret == Z_MEM_ERROR) error_msg += " (insufficient memory)";
        else if (ret == Z_BUF_ERROR) error_msg += " (insufficient buffer space)";
        else if (ret == Z_DATA_ERROR) error_msg += " (input data corrupted)";
        throw std::runtime_error(error_msg);
    }
    
    result.resize(dest_len);
    return result;
}

bool GPK::load(const std::string& file_name) {
    name = file_name;
    
    std::ifstream package(file_name, std::ios::binary);
    if (!package.is_open()) {
        std::cerr << "Failed to open package: " << file_name << std::endl;
        return false;
    }
    
    // Get file size
    package.seekg(0, std::ios::end);
    std::streamsize fileSize = package.tellg();
    
    // Read GPK signature from the end of file
    GPKsig sign;
    package.seekg(fileSize - sizeof(GPKsig));
    package.read(reinterpret_cast<char*>(&sign), sizeof(GPKsig));
    
    if (package.gcount() != sizeof(GPKsig) ||
        strncmp(GPK_TAILER_IDENT0, sign.sig0, strlen(GPK_TAILER_IDENT0)) != 0 ||
        strncmp(GPK_TAILER_IDENT1, sign.sig1, strlen(GPK_TAILER_IDENT1)) != 0) {
        std::cerr << "GPK: broken signature in " << file_name << std::endl;
        return false;
    }
    
    // Read compressed index data
    package.seekg(fileSize - sizeof(GPKsig) - sign.pidx_length);
    std::vector<uint8_t> compressedData(sign.pidx_length);
    package.read(reinterpret_cast<char*>(compressedData.data()), sign.pidx_length);
    
    // Decrypt the data
    for (size_t i = 0; i < compressedData.size(); i++) {
        compressedData[i] ^= CIPHERCODE[i % 16];
    }
    
    // Decompress the data using proper zlib decompression
    std::vector<uint8_t> uncompressedData;
    try {
        uncompressedData = decompress_data(compressedData);
        std::cout << "Successfully decompressed " << compressedData.size() 
                  << " bytes to " << uncompressedData.size() << " bytes" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "Decompression failed: " << e.what() << std::endl;
        return false;
    }
    
    // Parse the index data
    size_t pos = 0;
    while (pos < uncompressedData.size()) {
        if (pos + sizeof(uint16_t) > uncompressedData.size()) break;
        
        GPKentry entry;
        
        // Read filename length
        uint16_t filename_len;
        memcpy(&filename_len, &uncompressedData[pos], sizeof(uint16_t));
        pos += sizeof(uint16_t);
        
        if (filename_len == 0) break; // End of entries
        
        if (pos + filename_len * 2 > uncompressedData.size()) break;
        
        // Read UTF-16LE filename and convert to UTF-8
        std::string filename = utf16le_to_utf8(&uncompressedData[pos], filename_len);
        pos += filename_len * 2;
        
        // Clean up the filename - remove null characters and invalid chars
        filename.erase(std::find(filename.begin(), filename.end(), '\0'), filename.end());
        
        // Replace invalid filesystem characters and normalize path separators
        std::string cleanFilename;
        for (char c : filename) {
            if (c >= 32 && c <= 126 && c != '<' && c != '>' && c != ':' && 
                c != '"' && c != '|' && c != '?' && c != '*') {
                cleanFilename += c;
            } else if (c == '\\' || c == '/') {
                cleanFilename += '/'; // Normalize path separators
            }
        }
        
        // Skip entries with empty or invalid filenames
        if (cleanFilename.empty()) {
            pos += sizeof(GPKEntryHeader);
            continue;
        }
        
        entry.name = cleanFilename;
        
        if (pos + sizeof(GPKEntryHeader) > uncompressedData.size()) break;
        
        // Read entry header
        memcpy(&entry.header, &uncompressedData[pos], sizeof(GPKEntryHeader));
        pos += sizeof(GPKEntryHeader);
        
        entries.push_back(entry);
    }
      package.close();
    std::cout << "Loaded " << entries.size() << " entries from " << getName() << std::endl;
    return true;
}

std::string GPK::getName() const {
    size_t lastSeparator = name.find_last_of("\\/");
    std::string filename = (lastSeparator != std::string::npos) ? 
                          name.substr(lastSeparator + 1) : name;
    
    // Remove .GPK extension
    size_t dotPos = filename.find_last_of('.');
    if (dotPos != std::string::npos) {
        filename = filename.substr(0, dotPos);
    }
    
    return filename;
}

void GPK::unpack_all(const std::string& dir) {
    std::ifstream package(name, std::ios::binary);
    if (!package.is_open()) {
        std::cerr << "Failed to open package for unpacking: " << name << std::endl;
        return;
    }
    
    // Create output directory
    std::filesystem::create_directories(dir);
    
    for (const GPKentry& entry : entries) {
        // Create subdirectories if needed
        std::string fullPath = dir + entry.name;
        std::string dirPath = std::filesystem::path(fullPath).parent_path().string();
        
        if (!dirPath.empty()) {
            std::filesystem::create_directories(dirPath);
        }
        
        // Read file data from package
        package.seekg(entry.header.offset);
        std::vector<uint8_t> buffer(entry.header.comprlen);
        package.read(reinterpret_cast<char*>(buffer.data()), entry.header.comprlen);
        
        // Write to output file
        std::ofstream outFile(fullPath, std::ios::binary);
        if (outFile.is_open()) {
            outFile.write(reinterpret_cast<const char*>(buffer.data()), buffer.size());
            outFile.close();
            std::cout << "Extracted: " << entry.name << std::endl;
        } else {
            std::cerr << "Failed to create output file: " << fullPath << std::endl;
        }
    }
    
    package.close();
}
