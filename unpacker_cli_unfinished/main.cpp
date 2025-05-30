#include <iostream>
#include <string>
#include <filesystem>
#include "filesystem.h"

#ifdef _WIN32
    const char PATH_SEPARATOR = '\\';
    const std::string PATH_SEPARATOR_STR = "\\";
#else
    const char PATH_SEPARATOR = '/';
    const std::string PATH_SEPARATOR_STR = "/";
#endif

void printUsage(const char* programName) {
    std::cout << "Usage: " << programName << " <path_to_game_directory>" << std::endl;
    std::cout << "Example: " << programName << " \"D:\\Games\\Overflow\\SCHOOLDAYS HQ\"" << std::endl;
}

int main(int argc, char* argv[]) {
    std::cout << "School Days GPK Unpacker (CLI Version)" << std::endl;
    std::cout << "=====================================" << std::endl;
    
    if (argc != 2) {
        printUsage(argv[0]);
        return 1;
    }
    
    std::string gameRoot = argv[1];
    
    // Validate directory exists
    if (!std::filesystem::exists(gameRoot)) {
        std::cerr << "Error: Directory does not exist: " << gameRoot << std::endl;
        return 1;
    }
    
    // Check if it's actually a directory
    if (!std::filesystem::is_directory(gameRoot)) {
        std::cerr << "Error: Path is not a directory: " << gameRoot << std::endl;
        return 1;
    }
    
    // Check for packs subdirectory
    std::string packsDir = gameRoot + PATH_SEPARATOR_STR + "packs";
    if (!std::filesystem::exists(packsDir)) {
        std::cerr << "Warning: packs directory not found at: " << packsDir << std::endl;
        std::cerr << "Make sure this is a valid School Days game directory." << std::endl;
    }
    
    std::cout << "Game directory: " << gameRoot << std::endl;
    std::cout << "Starting extraction..." << std::endl;
    
    try {
        FileSystem fs(gameRoot);
        fs.unpack_all();
        std::cout << "Extraction completed successfully!" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "Error during extraction: " << e.what() << std::endl;
        return 1;
    }
    
    return 0;
}
