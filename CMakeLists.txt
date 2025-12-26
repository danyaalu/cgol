cmake_minimum_required(VERSION 3.16)
project(cgol_project LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Compiler warnings
if(MSVC)
    add_compile_options(/W4)
else()
    add_compile_options(-Wall -Wextra -Wpedantic -O3)
endif()

# Include directories
include_directories(include)

# Core Library
add_library(cgol_core
    src/state.cpp
    src/engine.cpp
    src/hash.cpp
    src/normalize.cpp
    src/runner.cpp
)
target_include_directories(cgol_core PUBLIC include)

# Runner Executable
add_executable(cgol_runner apps/runner/main.cpp)
target_link_libraries(cgol_runner PRIVATE cgol_core)

# Tests
enable_testing()
add_executable(test_life tests/test_life.cpp)
target_link_libraries(test_life PRIVATE cgol_core)
add_test(NAME test_life COMMAND test_life)
