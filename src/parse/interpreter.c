#include "interpreter.h"

#include "_cgo_export.h"

// Since we dlsym() the callbacks out of the parser .so, we have variables for them as
// well as extern definitions which cffi uses. The two must match, of course.
typedef void (RegisterPypyCallback)(char*, char*, void*);
typedef char* (ParseFileCallback)(char*, char*, size_t);
typedef void (SetConfigValueCallback)(char*, char*);
typedef char* (PreBuildCallbackRunner)(void*, size_t, char*);
typedef char* (PostBuildCallbackRunner)(void*, size_t, char*, char*);
ParseFileCallback* parse_file;
ParseFileCallback* parse_code;
SetConfigValueCallback* set_config_value;
PreBuildCallbackRunner* pre_build_callback_runner;
PostBuildCallbackRunner* post_build_callback_runner;

char* ParseFile(char* filename, char* package_name, size_t package) {
    return (*parse_file)(filename, package_name, package);
}

char* ParseCode(char* filename, char* package_name, size_t package) {
    return (*parse_code)(filename, package_name, package);
}

void SetConfigValue(char* name, char* value) {
    (*set_config_value)(name, value);
}

char* RunPreBuildFunction(size_t callback, size_t package, char* name) {
    return (*pre_build_callback_runner)((void*)callback, package, name);
}

char* RunPostBuildFunction(size_t callback, size_t package, char* name, char* output) {
    return (*post_build_callback_runner)((void*)callback, package, name, output);
}

int InitialiseInterpreter(char* parser_location) {
  void* parser = dlopen(parser_location, RTLD_NOW | RTLD_GLOBAL);
  if (parser == NULL) {
    return 1;
  }
  RegisterPypyCallback* reg = dlsym(parser, "RegisterCallback");
  parse_file = dlsym(parser, "ParseFile");
  parse_code = dlsym(parser, "ParseCode");
  set_config_value = dlsym(parser, "SetConfigValue");
  pre_build_callback_runner = dlsym(parser, "PreBuildFunctionRunner");
  post_build_callback_runner = dlsym(parser, "PostBuildFunctionRunner");
  if (!reg || !parse_file || !parse_code || !set_config_value ||
      !pre_build_callback_runner || !post_build_callback_runner) {
    return 2;
  }
  // TODO(pebers): it would be nicer if we could get rid of the explicit types here; something
  //               like reg("_add_target", typeof(AddTarget), AddTarget) would be sweet.
  //               As far as I know this is only possible in C++ using typeid though :(
  reg("_add_target", "AddTargetCallback*", AddTarget);
  reg("_add_src", "AddStringCallback*", AddSource);
  reg("_add_data", "AddStringCallback*", AddData);
  reg("_add_dep", "AddStringCallback*", AddDep);
  reg("_add_exported_dep", "AddStringCallback*", AddExportedDep);
  reg("_add_tool", "AddStringCallback*", AddTool);
  reg("_add_out", "AddStringCallback*", AddOutput);
  reg("_add_vis", "AddStringCallback*", AddVis);
  reg("_add_label", "AddStringCallback*", AddLabel);
  reg("_add_hash", "AddStringCallback*", AddHash);
  reg("_add_licence", "AddStringCallback*", AddLicence);
  reg("_add_test_output", "AddStringCallback*", AddTestOutput);
  reg("_add_require", "AddStringCallback*", AddRequire);
  reg("_add_provide", "AddTwoStringsCallback*", AddProvide);
  reg("_add_named_src", "AddTwoStringsCallback*", AddNamedSource);
  reg("_add_command", "AddTwoStringsCallback*", AddCommand);
  reg("_set_container_setting", "AddTwoStringsCallback*", SetContainerSetting);
  reg("_glob", "GlobCallback*", Glob);
  reg("_get_include_file", "GetIncludeFileCallback*", GetIncludeFile);
  reg("_get_subinclude_file", "GetIncludeFileCallback*", GetSubincludeFile);
  reg("_get_labels", "GetLabelsCallback*", GetLabels);
  reg("_set_pre_build_callback", "SetBuildFunctionCallback*", SetPreBuildFunction);
  reg("_set_post_build_callback", "SetBuildFunctionCallback*", SetPostBuildFunction);
  reg("_add_dependency", "AddDependencyCallback*", AddDependency);
  reg("_add_output", "AddOutputCallback*", AddOutputPost);
  reg("_add_licence_post", "AddTwoStringsCallback*", AddLicencePost);
  reg("_set_command", "AddThreeStringsCallback*", SetCommand);
  reg("_log", "LogCallback*", Log);
  reg("_is_valid_target_name", "ValidateCallback*", IsValidTargetName);
  return 0;
}
