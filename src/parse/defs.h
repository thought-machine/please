// Interface code between Python & Go. C is a kind of intermediate translation layer.
// This is used by both cgo and cffi to generate their own interfaces.
typedef unsigned char uint8;
typedef long long int64;
typedef size_t PackagePtr;
typedef size_t TargetPtr;
typedef char* (ParseFileCallback)(char*, char*, PackagePtr);
typedef void* (AddTargetCallback)(PackagePtr, char*, char*, char*, uint8, uint8, uint8, uint8, uint8, uint8, uint8, uint8, int64, int64, int64, char*);
typedef void (AddStringCallback)(TargetPtr, char*);
typedef void (AddTwoStringsCallback)(TargetPtr, char*, char*);
typedef void (AddThreeStringsCallback)(TargetPtr, char*, char*, char*);
typedef void (AddDependencyCallback)(TargetPtr, char*, char*, uint8);
typedef void (AddOutputCallback)(TargetPtr, char*, char*);
typedef char** (GlobCallback)(char*, char**, long long, char**, long long, uint8);
typedef char* (GetIncludeFileCallback)(PackagePtr, char*);
typedef char** (GetLabelsCallback)(PackagePtr, char*, char*);
typedef void (SetConfigValueCallback)(char*, char*);
typedef char* (PreBuildCallbackRunner)(void*, PackagePtr, char*);
typedef char* (PostBuildCallbackRunner)(void*, PackagePtr, char*, char*);
typedef void (SetBuildFunctionCallback)(void*, char*, TargetPtr);
typedef void (LogCallback)(int64, PackagePtr, char*);

// NB. This struct must remain consistent with the PleaseCallbacks type
//     in interpreter.go, otherwise Bad Things will happen.
struct PleaseCallbacks {
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
    SetBuildFunctionCallback* set_pre_build_function;
    SetBuildFunctionCallback* set_post_build_function;
    AddDependencyCallback* add_dependency;
    AddOutputCallback* add_output;
    AddTwoStringsCallback* add_licence_post;
    AddThreeStringsCallback* set_command;
    SetConfigValueCallback* set_config_value;
    PreBuildCallbackRunner* pre_build_callback_runner;
    PostBuildCallbackRunner* post_build_callback_runner;
    LogCallback* log;
};
