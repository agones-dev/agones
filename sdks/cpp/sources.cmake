set(SOURCE_FILES
    src/agones/sdk.cc
)

set(HEADER_FILES
    include/agones/sdk.h
)

set(GENERATED_SOURCE_FILES
    src/agones/sdk.grpc.pb.cc
    src/agones/sdk.pb.cc
)

set(GENERATED_GOOGLE_SOURCE_FILES
    src/google/annotations.pb.cc
    src/google/http.pb.cc
    src/google/client.pb.cc
    src/google/field_behavior.pb.cc
    src/google/launch_stage.pb.cc
    src/google/resource.pb.cc
)

set(GENERATED_GRPC_SOURCE_FILES
    src/protoc-gen-openapiv2/annotations.pb.cc
    src/protoc-gen-openapiv2/openapiv2.pb.cc
)

set(GENERATED_HEADER_FILES
    include/agones/sdk.grpc.pb.h
    include/agones/sdk.pb.h
)

set(GENERATED_GOOGLE_HEADER_FILES
    include/google/api/annotations.pb.h
    include/google/api/http.pb.h
    include/google/api/client.pb.h
    include/google/api/field_behavior.pb.h
    include/google/api/launch_stage.pb.h
    include/google/api/resource.pb.h
)

set(GENERATED_GRPC_HEADER_FILES
    include/protoc-gen-openapiv2/options/annotations.pb.h
    include/protoc-gen-openapiv2/options/openapiv2.pb.h
)

set(ALL_FILES
    ${SOURCE_FILES}
    ${HEADER_FILES}
    ${GENERATED_SOURCE_FILES}
    ${GENERATED_GOOGLE_SOURCE_FILES}
    ${GENERATED_GRPC_SOURCE_FILES}
    ${GENERATED_HEADER_FILES}
    ${GENERATED_GOOGLE_HEADER_FILES}
    ${GENERATED_GRPC_HEADER_FILES}
    "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}_export.h"
    "${CMAKE_CURRENT_BINARY_DIR}/${PROJECT_NAME}_global.h"
)

set(SDK_FILES
    ${SOURCE_FILES}
    ${HEADER_FILES}
)