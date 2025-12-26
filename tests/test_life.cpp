#include "cgol/state.hpp"
#include "cgol/engine.hpp"
#include "cgol/runner.hpp"
#include "cgol/normalize.hpp"
#include "cgol/hash.hpp"
#include <cassert>
#include <iostream>
#include <vector>

// Simple test framework
#define ASSERT(cond) \
    do { \
        if (!(cond)) { \
            std::cerr << "Assertion failed: " << #cond << " at " << __FILE__ << ":" << __LINE__ << std::endl; \
            std::exit(1); \
        } \
    } while (0)

void test_block_still_life() {
    std::cout << "Running test_block_still_life..." << std::endl;
    // Block:
    // OO
    // OO
    std::vector<cgol::Cell> cells = {{0, 0}, {1, 0}, {0, 1}, {1, 1}};
    cgol::State state(cells);
    
    cgol::State next = cgol::Engine::step(state);
    ASSERT(state == next);
    ASSERT(state.population() == 4);
}

void test_blinker_oscillation() {
    std::cout << "Running test_blinker_oscillation..." << std::endl;
    // Blinker (period 2):
    // OOO ->  Os
    //         O
    //         O
    std::vector<cgol::Cell> horizontal = {{0, 0}, {1, 0}, {2, 0}};
    std::vector<cgol::Cell> vertical = {{1, -1}, {1, 0}, {1, 1}};
    
    cgol::State s1(horizontal);
    cgol::State s2(vertical);

    cgol::State next1 = cgol::Engine::step(s1);
    ASSERT(next1 == s2);

    cgol::State next2 = cgol::Engine::step(next1);
    ASSERT(next2 == s1);
}

void test_glider_movement() {
    std::cout << "Running test_glider_movement..." << std::endl;
    // Glider at (0,0)
    // .O.
    // ..O
    // OOO
    std::vector<cgol::Cell> cells = {{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2}};
    cgol::State s0(cells);

    // Run 4 steps, should move (1, 1)
    cgol::State s = s0;
    for (int i = 0; i < 4; ++i) {
        s = cgol::Engine::step(s);
    }

    // Expected shifted glider
    std::vector<cgol::Cell> expected_cells;
    for (const auto& c : cells) {
        expected_cells.push_back({c.x + 1, c.y + 1});
    }
    cgol::State expected(expected_cells);

    ASSERT(s == expected);
}

void test_extinction() {
    std::cout << "Running test_extinction..." << std::endl;
    // Single cell dies immediately
    std::vector<cgol::Cell> cells = {{0, 0}};
    cgol::State s(cells);
    
    cgol::State next = cgol::Engine::step(s);
    ASSERT(next.empty());
}

void test_normalization() {
    std::cout << "Running test_normalization..." << std::endl;
    std::vector<cgol::Cell> c1 = {{10, 10}, {11, 10}};
    std::vector<cgol::Cell> c2 = {{0, 0}, {1, 0}};
    
    cgol::State s1(c1);
    cgol::State s2(c2);

    ASSERT(cgol::normalize(s1) == s2);
    ASSERT(cgol::hash_state(cgol::normalize(s1)) == cgol::hash_state(s2));
}

void test_runner_halt() {
    std::cout << "Running test_runner_halt..." << std::endl;
    // Single cell -> HALT
    std::vector<cgol::Cell> cells = {{0, 0}};
    cgol::State s(cells);
    
    cgol::Runner::Config cfg;
    auto res = cgol::Runner::run(s, cfg);
    
    ASSERT(res.outcome == cgol::Outcome::HALT);
    ASSERT(res.steps == 1);
}

void test_runner_loop() {
    std::cout << "Running test_runner_loop..." << std::endl;
    // Blinker -> LOOP
    std::vector<cgol::Cell> cells = {{0, 0}, {1, 0}, {2, 0}};
    cgol::State s(cells);
    
    cgol::Runner::Config cfg;
    auto res = cgol::Runner::run(s, cfg);
    
    ASSERT(res.outcome == cgol::Outcome::LOOP);
    // Steps might vary depending on when loop is detected (state 0 vs state 2)
    // Blinker is period 2.
    // 0: H
    // 1: V
    // 2: H (Loop detected here if we store 0)
    ASSERT(res.steps > 0);
}

int main() {
    test_block_still_life();
    test_blinker_oscillation();
    test_glider_movement();
    test_extinction();
    test_normalization();
    test_runner_halt();
    test_runner_loop();
    
    std::cout << "All tests passed!" << std::endl;
    return 0;
}
