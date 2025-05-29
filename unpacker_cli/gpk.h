#ifndef GPK_H
#define GPK_H

#include <string>
#include <vector>
#include <cstdint>

#define GPK_TAILER_IDENT0 "STKFile0PIDX"
#define GPK_TAILER_IDENT1 "STKFile0PACKFILE"

const unsigned char CIPHERCODE[16] = {0x82, 0xEE, 0x1D, 0xB3,
                                      0x57, 0xE9, 0x2C, 0xC2,
                                      0x2F, 0x54, 0x7B, 0x10,
                                      0x4C, 0x9A, 0x75, 0x49
                                     };

#pragma pack(1)
typedef struct {
    char        sig0[12];
    uint32_t    pidx_length;
    char        sig1[16];
} GPKsig;

typedef struct {
    uint16_t    sub_version;    // same as script.gpk.* suffix
    uint16_t    version;        // major version(always 1)
    uint16_t    zero;           // always 0
    uint32_t    offset;         // pidx data file offset
    uint32_t    comprlen;       // compressed pidx data length
    char        dflt[4];        // magic "DFLT" or "    "
    uint32_t    uncomprlen;     // raw pidx data length(if magic isn't DFLT, then this filed always zero)
    char        comprheadlen;   // pidx data header length
} GPKEntryHeader;

typedef struct {
    std::string         name;
    GPKEntryHeader      header;
} GPKentry;
#pragma pack()

class GPK {
public:
    explicit GPK();
    ~GPK();

    bool load(const std::string& file_name);
    std::string getName() const;
    void unpack_all(const std::string& dir);
    
    // Public for testing
    std::vector<uint8_t> decompress_data(const std::vector<uint8_t>& compressed_data);
    std::string utf16le_to_utf8(const uint8_t* data, size_t length_in_chars);

private:
    std::vector<GPKentry> entries;
    std::string name;
};

#endif // GPK_H
