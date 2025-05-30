#ifndef FILESYSTEM_H
#define FILESYSTEM_H

#include <string>
#include <vector>
#include "gpk.h"

class FileSystem {
public:
    explicit FileSystem(const std::string& gameRoot);
    ~FileSystem();

    void unpack_all();
    std::string normalize_name(const std::string& name);

private:
    std::string root;
    std::vector<GPK*> gpks;

    void findArchives();
    void mountGPK(const std::string& fileName);
    std::string normalize_name(const std::string& pkg, const std::string& name);
};

#endif // FILESYSTEM_H
