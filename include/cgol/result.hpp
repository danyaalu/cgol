#pragma once

#include <cstdint>
#include <string>

namespace cgol {

enum class Outcome {
    HALT,    // Empty state reached
    LOOP,    // Repeating state detected
    TIMEOUT  // Max steps exceeded
};

struct Result {
    Outcome outcome;
    uint64_t steps;
    uint64_t max_population;
};

// Helper to convert Outcome to string for JSON
inline std::string to_string(Outcome o) {
    switch (o) {
        case Outcome::HALT: return "HALT";
        case Outcome::LOOP: return "LOOP";
        case Outcome::TIMEOUT: return "TIMEOUT";
        default: return "UNKNOWN";
    }
}

} // namespace cgol
