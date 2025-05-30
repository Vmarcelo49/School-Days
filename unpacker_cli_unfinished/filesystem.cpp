#include "filesystem.h"
#include <iostream>
#include <filesystem>
#include <algorithm>

#ifdef _WIN32
    const char PATH_SEPARATOR = '\\';
    const std::string PATH_SEPARATOR_STR = "\\";
#else
    const char PATH_SEPARATOR = '/';
    const std::string PATH_SEPARATOR_STR = "/";
#endif

FileSystem::FileSystem(const std::string& gameRoot) : root(gameRoot) {
    // Ensure the path ends with a separator
    if (!root.empty() && root.back() != PATH_SEPARATOR) {
        root += PATH_SEPARATOR_STR;
    }
    findArchives();
}

FileSystem::~FileSystem() {
    for (GPK* gpk : gpks) {
        delete gpk;
    }
}

void FileSystem::unpack_all() {
    for (GPK* gpk : gpks) {
        std::cout << "Unpacking: " << gpk->getName() << std::endl;
        std::string outputDir = root + gpk->getName() + PATH_SEPARATOR_STR;
        gpk->unpack_all(outputDir);
    }
}

void FileSystem::findArchives() {
    std::string packsRoot = root + "packs" + PATH_SEPARATOR_STR;
    
    try {
        if (!std::filesystem::exists(packsRoot)) {
            std::cout << "Warning: packs directory not found at: " << packsRoot << std::endl;
            return;
        }
        
        for (const auto& entry : std::filesystem::directory_iterator(packsRoot)) {
            if (entry.is_regular_file()) {
                std::string filename = entry.path().filename().string();
                std::string extension = entry.path().extension().string();
                
                // Convert to uppercase for comparison
                std::transform(extension.begin(), extension.end(), extension.begin(), ::toupper);
                
                if (extension == ".GPK") {
                    std::string fullPath = entry.path().string();
                    mountGPK(fullPath);
                    std::cout << "Mounted package: " << filename << std::endl;
                }
            }
        }
    } catch (const std::filesystem::filesystem_error& ex) {
        std::cerr << "Filesystem error: " << ex.what() << std::endl;
    }
}

void FileSystem::mountGPK(const std::string& fileName) {
    GPK* gpk = new GPK();
    if (gpk->load(fileName)) {
        gpks.push_back(gpk);
    } else {
        delete gpk;
    }
}

std::string FileSystem::normalize_name(const std::string& name) {
    size_t slashPos = name.find('/');
    if (slashPos != std::string::npos) {
        std::string pkg = name.substr(0, slashPos);
        return normalize_name(pkg, name);
    }
    return name;
}

std::string FileSystem::normalize_name(const std::string& pkg, const std::string& name) {
    if (pkg.substr(0, 5) == "SysSe" || pkg.substr(0, 2) == "Se" || pkg.substr(0, 5) == "Voice") {
        return name + ".ogg";
    } else if (pkg.substr(0, 3) == "BGM") {
        return name + "_loop.ogg";
    } else if (pkg.substr(0, 5) == "Event") {
        return name + ".PNG";
    } else {
        return name;
    }
}
