#pragma once

#include "cgol/state.hpp"
#include "cgol/result.hpp"

namespace cgol {

class Runner {
public:
    struct Config {
        uint64_t max_steps = 10000;
        bool detect_loops = true;
    };

    static Result run(const State& initial_state, const Config& config);
};

} // namespace cgol
