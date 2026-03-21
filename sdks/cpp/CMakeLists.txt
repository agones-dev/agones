# Copyright 2018 Google LLC All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

cmake_minimum_required(VERSION 3.15)

# Silent cmake/build output (internal option)
# Extra command line options
# For windows: cmake --build . --config Release --target install -- /v:quiet /nologo
# For *nix: cmake --build . --target install -- -s
option(AGONES_SILENT_OUTPUT "Show only warnings/error messages" OFF)
if (AGONES_SILENT_OUTPUT)
    function(message)
        list(GET ARGV 0 MessageType)
        list(REMOVE_AT ARGV 0)
        if (MessageType STREQUAL FATAL_ERROR OR
            MessageType STREQUAL SEND_ERROR OR
            MessageType STREQUAL WARNING OR
            MessageType STREQUAL AUTHOR_WARNING OR
            NOT ${AGONES_SILENT_OUTPUT}
            )
            _message(${MessageType} "${ARGV}")
        endif()
    endfunction()

    set(CMAKE_INSTALL_MESSAGE NEVER)
    set(CMAKE_VERBOSE_MAKEFILE OFF)
    set_property(GLOBAL PROPERTY RULE_MESSAGES OFF)
    set_property(GLOBAL PROPERTY TARGET_MESSAGES OFF)
endif(AGONES_SILENT_OUTPUT)

# Getting version from git
find_package(Git QUIET)
set(AGONES_VERSION "0.0.0")
if (Git_FOUND)
    execute_process(
        COMMAND ${GIT_EXECUTABLE} describe --tags --abbrev=0
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        RESULT_VARIABLE result
        OUTPUT_VARIABLE output
        ERROR_QUIET
    )
    if (${result} EQUAL 0)
        string(REGEX MATCH "[0-9]+\\.[0-9]+\\.[0-9]+" AGONES_VERSION ${output})
    endif()
else()
    message(WARNING "Git was not found. Current Agones version is set to ${AGONES_VERSION}")
endif()

# Project AGONES
project(agones VERSION ${AGONES_VERSION} HOMEPAGE_URL https://github.com/agones-dev/agones LANGUAGES C CXX)

# Options
option(AGONES_FORCE_GRPC_VERSION "Build Agones C++ SDK only with supported gRPC version" ON)
option(AGONES_CLANG_FORMAT "Apply clang-format (if found) as a pre-build step" OFF)

# Prerequisities
# [markmandel] - not 100% sure what include(CMakeFindDependencyMacro) does, but it solves the building issue.
# https://cmake.org/cmake/help/latest/manual/cmake-packages.7.html#creating-a-package-configuration-file
include(CMakeFindDependencyMacro)
include(ProcessorCount)

ProcessorCount(CPU_COUNT)
if (CPU_COUNT GREATER 0 AND NOT DEFINED CMAKE_BUILD_PARALLEL_LEVEL)
    set(ENV{CMAKE_BUILD_PARALLEL_LEVEL} "${CPU_COUNT}")
    message(STATUS "Setting CMAKE_BUILD_PARALLEL_LEVEL to $ENV{CMAKE_BUILD_PARALLEL_LEVEL}")
endif()

# gRPC
find_package(Protobuf REQUIRED)
find_package(gRPC CONFIG REQUIRED)
find_package(Threads REQUIRED)

message(STATUS "gRPC version found: ${gRPC_VERSION}")

# Settings
set_property(GLOBAL PROPERTY USE_FOLDERS ON)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED TRUE)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_INCLUDE_CURRENT_DIR ON)
set(CMAKE_INCLUDE_CURRENT_DIR_IN_INTERFACE ON)
set(CMAKE_DEBUG_POSTFIX "d")

# Platform specific stuff
if (WIN32)
    # Windows
    add_compile_definitions(
        _WIN32_WINNT=0x0600
        WINDOWS
    )
elseif (APPLE)
    # Mac OS
elseif (UNIX AND NOT APPLE)
    # Linux
endif()

# Agones SDK
include(./sources.cmake)

# Global header
set(GLOBAL_HEADER "${PROJECT_NAME}_global.h")
set(GLOBAL_CONFIG_CONTENT "\n")
configure_file(cmake/${GLOBAL_HEADER}.in ${GLOBAL_HEADER} @ONLY)

if(MSVC)
  add_definitions(/FI"${GLOBAL_HEADER}")
else()
  # GCC or Clang
  add_definitions(-include ${GLOBAL_HEADER})
endif()

add_library(${PROJECT_NAME} STATIC ${ALL_FILES})
target_link_libraries(${PROJECT_NAME} PUBLIC protobuf::libprotobuf gRPC::grpc++_unsecure)

target_include_directories(${PROJECT_NAME} PUBLIC
    $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include>
    $<BUILD_INTERFACE:${CMAKE_CURRENT_LIST_DIR}/include/${PROJECT_NAME}>
    $<INSTALL_INTERFACE:${PROJECT_NAME}/include>
)

# Fix compiler warnings
# https://github.com/protocolbuffers/protobuf/blob/master/cmake/README.md#notes-on-compiler-warnings
if (MSVC)
    set(OPT_DISABLE_COMPILER_WARNINGS /wd4101 /wd4146 /wd4251 /wd4661)
    target_compile_options(${PROJECT_NAME} PUBLIC ${OPT_DISABLE_COMPILER_WARNINGS})
endif()

# Export header
include(GenerateExportHeader)
set(EXPORT_HEADER "${PROJECT_NAME}_export.h")
generate_export_header(${PROJECT_NAME} EXPORT_FILE_NAME ${EXPORT_HEADER} DEFINE_NO_DEPRECATED)

# CMake package generation
include(CMakePackageConfigHelpers)

set(_INCLUDE_DIRS "agones/include")
set(_CMAKE_CONFIG_DESTINATION "agones/cmake")

# Config for find_package
configure_package_config_file(
    cmake/${PROJECT_NAME}Config.cmake.in
    ${PROJECT_NAME}Config.cmake
    INSTALL_DESTINATION ${CMAKE_INSTALL_PREFIX}/${_CMAKE_CONFIG_DESTINATION}
    PATH_VARS _INCLUDE_DIRS PROJECT_VERSION
    NO_SET_AND_CHECK_MACRO
)

# Build artifacts
install(TARGETS ${PROJECT_NAME} EXPORT ${PROJECT_NAME}
    LIBRARY DESTINATION  ${PROJECT_NAME}/lib
    ARCHIVE DESTINATION  ${PROJECT_NAME}/lib
    RUNTIME DESTINATION  ${PROJECT_NAME}/bin
    INCLUDES DESTINATION ${_INCLUDE_DIRS}
)
install(EXPORT ${PROJECT_NAME} DESTINATION ${_CMAKE_CONFIG_DESTINATION} FILE ${PROJECT_NAME}Targets.cmake)
# Package config
install(
    FILES ${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}Config.cmake
    DESTINATION ${_CMAKE_CONFIG_DESTINATION}
)
# Agones header files
install(
    FILES ${HEADER_FILES} "${CMAKE_CURRENT_BINARY_DIR}/${EXPORT_HEADER}" "${CMAKE_CURRENT_BINARY_DIR}/${GLOBAL_HEADER}"
    DESTINATION ${_INCLUDE_DIRS}/${PROJECT_NAME}
)
# Generated header files
install(
    FILES ${GENERATED_HEADER_FILES}
    DESTINATION ${_INCLUDE_DIRS}/${PROJECT_NAME}
)
# Google header files
install(
    FILES ${GENERATED_GOOGLE_HEADER_FILES}
    DESTINATION ${_INCLUDE_DIRS}/google/api
)
# grpc-gateway header files
install(
    FILES ${GENERATED_GRPC_HEADER_FILES}
    DESTINATION ${_INCLUDE_DIRS}/protoc-gen-openapiv2/options
)

unset(_INCLUDE_DIRS)
unset(_CMAKE_CONFIG_DESTINATION)

# clang-format
find_program(
    CLANG_FORMAT_APP
    NAMES "clang-format"
    DOC "Path to clang-format"
)
if (NOT CLANG_FORMAT_APP)
    message(STATUS "clang-format not found.")
else()
    message(STATUS "clang-format found: ${CLANG_FORMAT_APP}")

    set(CLANGFORMAT_INPUT)
    foreach(relpath ${SDK_FILES})
        get_filename_component(fullpath "${CMAKE_CURRENT_LIST_DIR}/${relpath}" ABSOLUTE)
        list(APPEND CLANGFORMAT_INPUT ${fullpath})
    endforeach()

    # format
    add_custom_target(
        clang-format-apply
        COMMAND ${CLANG_FORMAT_APP} -i --style=file --fallback-style=Google ${CLANGFORMAT_INPUT}
        WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    )
    if (${AGONES_CLANG_FORMAT})
        add_dependencies(${PROJECT_NAME} "clang-format-apply")
    endif()

    # verification
    set(CLANGFORMAT_WORKING_DIR ${CMAKE_CURRENT_SOURCE_DIR})
    get_filename_component(AGONES_ROOT "${CMAKE_CURRENT_LIST_DIR}/../.." ABSOLUTE)
    get_filename_component(AGONES_TEMP "${CMAKE_CURRENT_BINARY_DIR}/clang-format/temp" ABSOLUTE)
    configure_file("cmake/clang-verify.in" "clang-format/CMakeLists.txt" @ONLY)
    add_custom_target(
        clang-format-verify
        COMMAND ${CMAKE_COMMAND} .
        WORKING_DIRECTORY "${CMAKE_CURRENT_BINARY_DIR}/clang-format"
    )
endif()
