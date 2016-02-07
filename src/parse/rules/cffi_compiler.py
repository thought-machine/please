"""Compiler script to build the CFFI modules."""

from cffi import FFI


# Note that this must remain consistent with interpreter.h and interpreter.go
# otherwise Very Bad Things will happen.
CDEF = """
typedef unsigned char uint8;
typedef long long int64;
typedef char* (ParseFileCallback)(const char*, const char*, void*);
typedef void* (AddTargetCallback)(void*, char*, char*, char*, uint8, uint8, uint8, uint8, uint8, uint8, uint8, uint8, int64, int64, int64, char*);
typedef void (AddStringCallback)(void*, char*);
typedef void (AddTwoStringsCallback)(void*, char*, char*);
typedef void (AddThreeStringsCallback)(void*, char*, char*, char*);
typedef void (AddDependencyCallback)(void*, char*, char*, uint8);
typedef void (AddOutputCallback)(void*, char*, char*);
typedef char** (GlobCallback)(char*, char**, long long, char**, long long, uint8);
typedef char* (GetIncludeFileCallback)(void*, char*);
typedef char** (GetLabelsCallback)(void*, char*, char*);
typedef void (SetConfigValueCallback)(char*, char*);
typedef char* (PreBuildCallbackRunner)(void*, void*, char*);
typedef char* (PostBuildCallbackRunner)(void*, void*, char*, char*);
// We're not going to call this SetBuildCallbackCallback because we're not peasants.
typedef void (SetBuildFunctionCallback)(void*, char*, void*);
typedef void (LogCallback)(int64, void*, char*);

typedef struct _PleaseCallbacks {
    ParseFileCallback* parse_file;
    ParseFileCallback* parse_code;
    AddTargetCallback* add_target;
    AddStringCallback* add_src;
    AddStringCallback* add_data;
    AddStringCallback* add_dep;
    AddStringCallback* add_exported_dep;
    AddStringCallback* add_tool;
    AddStringCallback* add_out;
    AddStringCallback* add_vis;
    AddStringCallback* add_label;
    AddStringCallback* add_hash;
    AddStringCallback* add_licence;
    AddStringCallback* add_test_output;
    AddStringCallback* add_require;
    AddTwoStringsCallback* add_provide;
    AddTwoStringsCallback* add_named_src;
    AddTwoStringsCallback* add_command;
    AddTwoStringsCallback* set_container_setting;
    GlobCallback* glob;
    GetIncludeFileCallback* get_include_file;
    GetIncludeFileCallback* get_subinclude_file;
    GetLabelsCallback* get_labels;
    SetBuildFunctionCallback* set_pre_build_callback;
    SetBuildFunctionCallback* set_post_build_callback;
    AddDependencyCallback* add_dependency;
    AddOutputCallback* add_output;
    AddTwoStringsCallback* add_licence_post;
    AddThreeStringsCallback* set_command;
    SetConfigValueCallback* set_config_value;
    PreBuildCallbackRunner* pre_build_callback_runner;
    PostBuildCallbackRunner* post_build_callback_runner;
    LogCallback* log;
} PleaseCallbacks;
"""


if __name__ == '__main__':
    ffi = FFI()
    ffi.set_source('_parser_interface', None)
    ffi.cdef(CDEF)
    ffi.compile()
