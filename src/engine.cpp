#include "cgol/engine.hpp"
#include <unordered_map>
#include <algorithm>
#include <vector>

namespace cgol {

// Helper for neighbor counting
// Using a flat map approach for sparse grid evolution
// This avoids allocating a full grid and only processes relevant cells.
// Performance note: For very dense patterns, other algorithms might be faster,
// but for general sparse patterns, this is efficient and memory-safe.
State Engine::step(const State& current) {
    if (current.empty()) {
        return State();
    }

    // Map: Cell -> Neighbor Count
    // We use a custom hash for Cell to ensure speed
    // However, since we need to iterate and produce a sorted State,
    // we can use a vector of potential cells and sort/count.
    // This "sort-and-scan" approach often beats hash maps for small-medium patterns
    // due to cache locality and lack of allocation overhead for map nodes.
    
    std::vector<Cell> neighbor_counts;
    neighbor_counts.reserve(current.population() * 8);

    for (const auto& cell : current) {
        // Add neighbors
        neighbor_counts.push_back({cell.x - 1, cell.y - 1});
        neighbor_counts.push_back({cell.x,     cell.y - 1});
        neighbor_counts.push_back({cell.x + 1, cell.y - 1});
        neighbor_counts.push_back({cell.x - 1, cell.y});
        neighbor_counts.push_back({cell.x + 1, cell.y});
        neighbor_counts.push_back({cell.x - 1, cell.y + 1});
        neighbor_counts.push_back({cell.x,     cell.y + 1});
        neighbor_counts.push_back({cell.x + 1, cell.y + 1});
    }

    std::sort(neighbor_counts.begin(), neighbor_counts.end());

    std::vector<Cell> next_generation;
    next_generation.reserve(current.population()); // Heuristic

    // Iterate through the sorted neighbors to count occurrences
    auto it = neighbor_counts.begin();
    while (it != neighbor_counts.end()) {
        const auto& current_cell = *it;
        int count = 0;
        
        // Count duplicates of current_cell
        while (it != neighbor_counts.end() && *it == current_cell) {
            count++;
            ++it;
        }

        // Apply rules
        // 1. Any live cell with 2 or 3 live neighbours lives.
        // 2. Any dead cell with exactly 3 live neighbours becomes a live cell.
        
        // We need to know if current_cell was alive in the previous generation.
        // Since 'current' is sorted, we can use binary search.
        // Optimization: We could iterate 'current' in parallel, but binary_search is O(log N).
        // Given N is usually small for "programs", this is acceptable.
        
        bool was_alive = std::binary_search(current.begin(), current.end(), current_cell);

        if (was_alive) {
            if (count == 2 || count == 3) {
                next_generation.push_back(current_cell);
            }
        } else {
            if (count == 3) {
                next_generation.push_back(current_cell);
            }
        }
    }

    // State constructor expects sorted unique cells, which we produced naturally
    // by processing sorted neighbor_counts in order.
    // However, State constructor re-sorts/uniques to be safe. 
    // We can optimize State later to accept "known sorted unique" if needed.
    return State(std::move(next_generation));
}

} // namespace cgol
