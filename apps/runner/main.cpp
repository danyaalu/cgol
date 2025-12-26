#include "cgol/state.hpp"
#include "cgol/runner.hpp"
#include <iostream>
#include <string>
#include <vector>
#include <sstream>
#include <iomanip>
#include <charconv>
#include <cstring>

// Simple argument parser
struct Args {
    int width = 0;
    int height = 0;
    std::string bitmask;
    std::string rle_file;
    uint64_t max_steps = 10000;
    bool detect_loops = true;
};

void print_usage(const char* prog) {
    std::cerr << "Usage: " << prog << " [options]\n"
              << "Options:\n"
              << "  --box W H           Width and Height of the bounding box\n"
              << "  --bitmask HEX       Row-major packed bits (hex string)\n"
              << "  --rle FILE          Path to RLE file (optional testing)\n"
              << "  --max-steps N       Maximum simulation steps (default 10000)\n"
              << "  --no-loop-detect    Disable loop detection\n";
}

// Hex char to int
int hex_char_to_int(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return 0;
}

cgol::State parse_bitmask(int w, int h, const std::string& hex) {
    std::vector<cgol::Cell> cells;
    
    // Each hex char represents 4 bits
    // We iterate through the hex string and map bits to the WxH grid
    // Row-major order
    
    int total_bits = w * h;
    int current_bit = 0;

    for (char c : hex) {
        int val = hex_char_to_int(c);
        // Process 4 bits from MSB to LSB
        for (int i = 3; i >= 0; --i) {
            if (current_bit >= total_bits) break;

            if ((val >> i) & 1) {
                int y = current_bit / w;
                int x = current_bit % w;
                cells.push_back({x, y});
            }
            current_bit++;
        }
    }

    return cgol::State(std::move(cells));
}

// Minimal RLE parser for testing
cgol::State parse_rle(const std::string& /*filename*/) {
    // Placeholder: In a real scenario, read file and parse RLE.
    // For this prompt, we focus on the bitmask input as primary.
    // Returning empty state or throwing error if used without implementation.
    std::cerr << "RLE support not fully implemented in this minimal runner.\n";
    return cgol::State();
}

int main(int argc, char* argv[]) {
    Args args;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--box") {
            if (i + 2 < argc) {
                args.width = std::stoi(argv[++i]);
                args.height = std::stoi(argv[++i]);
            }
        } else if (arg == "--bitmask") {
            if (i + 1 < argc) {
                args.bitmask = argv[++i];
            }
        } else if (arg == "--rle") {
            if (i + 1 < argc) {
                args.rle_file = argv[++i];
            }
        } else if (arg == "--max-steps") {
            if (i + 1 < argc) {
                args.max_steps = std::stoull(argv[++i]);
            }
        } else if (arg == "--no-loop-detect") {
            args.detect_loops = false;
        } else {
            print_usage(argv[0]);
            return 1;
        }
    }

    cgol::State initial_state;
    if (!args.bitmask.empty() && args.width > 0 && args.height > 0) {
        initial_state = parse_bitmask(args.width, args.height, args.bitmask);
    } else if (!args.rle_file.empty()) {
        initial_state = parse_rle(args.rle_file);
    } else {
        std::cerr << "Error: Must provide --box W H --bitmask HEX or --rle FILE\n";
        return 1;
    }

    cgol::Runner::Config config;
    config.max_steps = args.max_steps;
    config.detect_loops = args.detect_loops;

    cgol::Result result = cgol::Runner::run(initial_state, config);

    // JSONL Output
    std::cout << "{\"steps\":" << result.steps 
              << ",\"outcome\":\"" << cgol::to_string(result.outcome) << "\""
              << ",\"max_population\":" << result.max_population << "}" << std::endl;

    return 0;
}
