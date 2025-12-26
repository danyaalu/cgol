#pragma once

#include "cgol/state.hpp"

namespace cgol {

class Engine {
public:
    // Stateless step function
    // Takes current state, returns next state
    static State step(const State& current);
};

} // namespace cgol
