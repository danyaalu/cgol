#pragma once

#include "cgol/state.hpp"

namespace cgol {

// Computes a 64-bit hash of the state.
// Must be stable and deterministic.
// Used for loop detection.
uint64_t hash_state(const State& state);

} // namespace cgol
