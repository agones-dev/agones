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

#
# Makefile for testing and building the C++ SDK
#

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
source_path := $(dir $(mkfile_path))
build_path := $(source_path)/.build
package_path := $(build_path)/.install
install_path := /opt/local
archive_path := $(source_path)/.archives

build:
	-mkdir $(build_path)
	-mkdir $(archive_path)
	cd $(build_path) && cmake .. -DCMAKE_BUILD_TYPE=Release -DAGONES_SILENT_OUTPUT=ON -G "Unix Makefiles" -Wno-dev -DCMAKE_INSTALL_PREFIX=.install
	cd $(build_path) && cmake --build . --target install -j$$(nproc) -- -s
    
verify:
	cd $(build_path)/clang-format && cmake . -DAGONES_SILENT_OUTPUT=ON
    
install:
	cp -r $(build_path)/.install $(install_path)
	
archive: VERSION = "dev"
archive:
	cd $(package_path) && tar cf $(archive_path)/agonessdk-$(VERSION)-linux-arch_64.tar.gz *
	
clean:
	-rm -rf $(build_path)