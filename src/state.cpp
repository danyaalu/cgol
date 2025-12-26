#include "cgol/state.hpp"
#include <algorithm>

namespace cgol {

State::State(Container cells) : cells_(std::move(cells)) {
    // Ensure sorted and unique for canonical representation
    std::sort(cells_.begin(), cells_.end());
    cells_.erase(std::unique(cells_.begin(), cells_.end()), cells_.end());
}

bool State::empty() const {
    return cells_.empty();
}

size_t State::population() const {
    return cells_.size();
}

BoundingBox State::bounding_box() const {
    BoundingBox bb;
    if (empty()) return bb;

    for (const auto& cell : cells_) {
        bb.min_x = std::min(bb.min_x, cell.x);
        bb.max_x = std::max(bb.max_x, cell.x);
        bb.min_y = std::min(bb.min_y, cell.y);
        bb.max_y = std::max(bb.max_y, cell.y);
    }
    return bb;
}

State::const_iterator State::begin() const {
    return cells_.begin();
}

State::const_iterator State::end() const {
    return cells_.end();
}

const State::Container& State::get_cells() const {
    return cells_;
}

bool State::operator==(const State& other) const {
    return cells_ == other.cells_;
}

} // namespace cgol
