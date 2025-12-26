#include "cgol/runner.hpp"
#include "cgol/engine.hpp"
#include "cgol/normalize.hpp"
#include "cgol/hash.hpp"
#include <unordered_set>

namespace cgol {

Result Runner::run(const State& initial_state, const Config& config) {
    State current = initial_state;
    uint64_t steps = 0;
    uint64_t max_pop = current.population();

    // Store hashes of normalized states for loop detection
    std::unordered_set<uint64_t> history;

    if (config.detect_loops) {
        State normalized = normalize(current);
        history.insert(hash_state(normalized));
    }

    while (steps < config.max_steps) {
        if (current.empty()) {
            return {Outcome::HALT, steps, max_pop};
        }

        current = Engine::step(current);
        steps++;
        
        size_t pop = current.population();
        if (pop > max_pop) {
            max_pop = pop;
        }

        if (config.detect_loops) {
            State normalized = normalize(current);
            uint64_t h = hash_state(normalized);
            if (history.contains(h)) {
                return {Outcome::LOOP, steps, max_pop};
            }
            history.insert(h);
        }
    }

    return {Outcome::TIMEOUT, steps, max_pop};
}

} // namespace cgol
