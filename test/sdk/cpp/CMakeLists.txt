# Copyright Contributors to Agones a Series of LF Projects, LLC.
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

project(cpp-simple CXX)

# Settings
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED TRUE)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_INCLUDE_CURRENT_DIR ON)
set(CMAKE_INCLUDE_CURRENT_DIR_IN_INTERFACE ON)
set_property(GLOBAL PROPERTY USE_FOLDERS ON)
set(CMAKE_DEBUG_POSTFIX "d")

find_package(Threads REQUIRED)
find_package(agones CONFIG REQUIRED)

if(WIN32)
  # Windows
  add_definitions(-D_WIN32_WINNT=0x0600)
  set_property(DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR} PROPERTY VS_STARTUP_PROJECT cpp-simple)
elseif(APPLE)
  # Mac OS
elseif(UNIX AND NOT APPLE)
  # Linux
endif()

include_directories(${agones_INCLUDE_DIRS})
set(SRC_FILES "server.cc")

# Executable
add_executable(${PROJECT_NAME} ${SRC_FILES})
target_link_libraries(${PROJECT_NAME} PUBLIC agones)

if (MSVS)
  target_compile_options(${PROJECT_NAME} PUBLIC /wd4101 /wd4146 /wd4251 /wd4661)
endif()

# Installation
install(TARGETS ${PROJECT_NAME} EXPORT ${PROJECT_NAME} RUNTIME DESTINATION .)