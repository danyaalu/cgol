#pragma once

#include <cstdint>
#include <vector>
#include <functional>
#include <limits>

namespace cgol {

struct Cell {
    int64_t x;
    int64_t y;

    auto operator<=>(const Cell&) const = default;
};

struct BoundingBox {
    int64_t min_x = std::numeric_limits<int64_t>::max();
    int64_t max_x = std::numeric_limits<int64_t>::min();
    int64_t min_y = std::numeric_limits<int64_t>::max();
    int64_t max_y = std::numeric_limits<int64_t>::min();
    
    bool valid() const {
        return min_x <= max_x && min_y <= max_y;
    }
};

class State {
public:
    // Using a sorted vector for deterministic behavior, fast iteration, 
    // and easy normalization/hashing.
    // This satisfies "sparse set" requirement.
    using Container = std::vector<Cell>;
    using const_iterator = Container::const_iterator;

    State() = default;
    explicit State(Container cells);

    bool empty() const;
    size_t population() const;
    BoundingBox bounding_box() const;

    const_iterator begin() const;
    const_iterator end() const;
    
    const Container& get_cells() const;

    // Equality for tests/logic
    bool operator==(const State& other) const;

private:
    Container cells_;
};

} // namespace cgol

// Hash support for Cell
template <>
struct std::hash<cgol::Cell> {
    std::size_t operator()(const cgol::Cell& c) const {
        // Simple hash combination
        return std::hash<int64_t>{}(c.x) ^ (std::hash<int64_t>{}(c.y) << 1);
    }
};
