#include "cgol/hash.hpp"

namespace cgol {

// FNV-1a hash constants
constexpr uint64_t FNV_offset_basis = 0xcbf29ce484222325;
constexpr uint64_t FNV_prime = 0x100000001b3;

uint64_t hash_state(const State& state) {
    uint64_t hash = FNV_offset_basis;

    for (const auto& cell : state) {
        // Hash x
        const uint8_t* p = reinterpret_cast<const uint8_t*>(&cell.x);
        for (size_t i = 0; i < sizeof(int64_t); ++i) {
            hash ^= p[i];
            hash *= FNV_prime;
        }
        
        // Hash y
        p = reinterpret_cast<const uint8_t*>(&cell.y);
        for (size_t i = 0; i < sizeof(int64_t); ++i) {
            hash ^= p[i];
            hash *= FNV_prime;
        }
    }

    return hash;
}

} // namespace cgol
