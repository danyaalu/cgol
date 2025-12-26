#pragma once

#include "cgol/state.hpp"

namespace cgol {

// Normalizes the state by shifting live cells so min_x = 0 and min_y = 0.
// This is crucial for loop detection (translation invariance).
State normalize(const State& state);

} // namespace cgol
