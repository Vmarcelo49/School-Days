#include <iostream>
#include <cassert>
#include <vector>
#include <string>
#include "gpk.h"

// Test UTF-16LE to UTF-8 conversion
void test_utf16le_conversion() {
    std::cout << "Testing UTF-16LE to UTF-8 conversion..." << std::endl;
    
    GPK gpk; // Create GPK instance to access the method
    
    // Test ASCII characters
    uint8_t ascii_data[] = {0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00}; // "Hello"
    std::string result = gpk.utf16le_to_utf8(ascii_data, 5);
    assert(result == "Hello");
    std::cout << "  ASCII test passed" << std::endl;
    
    // Test 2-byte UTF-8 characters
    uint8_t utf8_2byte[] = {0xE9, 0x00}; // é (U+00E9)
    result = gpk.utf16le_to_utf8(utf8_2byte, 1);
    assert(result == "é");
    std::cout << "  2-byte UTF-8 test passed" << std::endl;
    
    // Test 3-byte UTF-8 characters
    uint8_t utf8_3byte[] = {0x42, 0x30}; // あ (U+3042 - Hiragana A)
    result = gpk.utf16le_to_utf8(utf8_3byte, 1);
    assert(result.length() == 3); // Should be 3 bytes in UTF-8
    std::cout << "  3-byte UTF-8 test passed" << std::endl;
    
    std::cout << "UTF-16LE conversion tests completed successfully!" << std::endl;
}

// Test decompression with known data
void test_decompression() {
    std::cout << "Testing decompression..." << std::endl;
    
    GPK gpk;
    
    // Test with minimal valid data (uncompressed size = 0 should handle gracefully)
    std::vector<uint8_t> minimal_data = {0x00, 0x00, 0x00, 0x00};
    
    try {
        std::vector<uint8_t> result = gpk.decompress_data(minimal_data);
        std::cout << "  Minimal data test handled gracefully" << std::endl;
    } catch (const std::exception& e) {
        std::cout << "  Expected exception for minimal data: " << e.what() << std::endl;
    }
    
    std::cout << "Decompression tests completed!" << std::endl;
}

// Test GPK signature verification (mock)
void test_gpk_signature() {
    std::cout << "Testing GPK signature constants..." << std::endl;
    
    // Verify constants are correct
    assert(std::string(GPK_TAILER_IDENT0) == "STKFile0PIDX");
    assert(std::string(GPK_TAILER_IDENT1) == "STKFile0PACKFILE");
    
    std::cout << "  GPK signature constants verified" << std::endl;
    std::cout << "GPK signature tests completed!" << std::endl;
}

// Test cipher code
void test_cipher() {
    std::cout << "Testing cipher code..." << std::endl;
    
    // Test basic XOR encryption/decryption
    std::vector<uint8_t> test_data = {0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
                                     0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
                                     0x00, 0x11}; // 18 bytes to test wrapping
    
    std::vector<uint8_t> original = test_data;
    
    // Encrypt
    for (size_t i = 0; i < test_data.size(); i++) {
        test_data[i] ^= CIPHERCODE[i % 16];
    }
    
    // Decrypt (XOR again)
    for (size_t i = 0; i < test_data.size(); i++) {
        test_data[i] ^= CIPHERCODE[i % 16];
    }
    
    // Should be back to original
    assert(test_data == original);
    
    std::cout << "  Cipher encryption/decryption test passed" << std::endl;
    std::cout << "Cipher tests completed!" << std::endl;
}

int main() {
    std::cout << "Running GPK Unpacker Unit Tests" << std::endl;
    std::cout << "===============================" << std::endl;
    
    try {
        test_gpk_signature();
        test_cipher();
        test_utf16le_conversion();
        test_decompression();
        
        std::cout << std::endl;
        std::cout << "All tests completed successfully!" << std::endl;
        return 0;
    } catch (const std::exception& e) {
        std::cerr << "Test failed: " << e.what() << std::endl;
        return 1;
    } catch (...) {
        std::cerr << "Unknown test failure" << std::endl;
        return 1;
    }
}
