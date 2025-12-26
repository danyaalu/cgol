#include "cgol/normalize.hpp"

namespace cgol {

State normalize(const State& state) {
    if (state.empty()) {
        return state;
    }

    BoundingBox bb = state.bounding_box();
    
    // If already normalized, return copy
    if (bb.min_x == 0 && bb.min_y == 0) {
        return state;
    }

    std::vector<Cell> normalized_cells;
    normalized_cells.reserve(state.population());

    for (const auto& cell : state) {
        normalized_cells.push_back({cell.x - bb.min_x, cell.y - bb.min_y});
    }

    // State constructor will sort, but since we subtracted constants from a sorted list,
    // it remains sorted.
    return State(std::move(normalized_cells));
}

} // namespace cgol
