#include "interpreter.h"

#include "_cgo_export.h"

// Since we dlsym() the callbacks out of the parser .so, we have variables for them as
// well as extern definitions which cffi uses. The two must match, of course.
char* (*parse_file)(char*, char*, size_t);
char* (*parse_code)(char*, char*, size_t);
void (*set_config_value)(char*, char*);
char* (*pre_build_callback_runner)(void*, size_t, char*);
char* (*post_build_callback_runner)(void*, size_t, char*, char*);
char* (*run_code)(char*);

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

char* RunCode(char* code) {
  return (*run_code)(code);
}

int InitialiseInterpreter(char* parser_location) {
  void* parser = dlopen(parser_location, RTLD_NOW | RTLD_GLOBAL);
  if (parser == NULL) {
    return 1;
  }
  int (*reg)(char*, char*, void*) = dlsym(parser, "RegisterCallback");
  parse_file = dlsym(parser, "ParseFile");
  parse_code = dlsym(parser, "ParseCode");
  set_config_value = dlsym(parser, "SetConfigValue");
  pre_build_callback_runner = dlsym(parser, "PreBuildFunctionRunner");
  post_build_callback_runner = dlsym(parser, "PostBuildFunctionRunner");
  run_code = dlsym(parser, "RunCode");
  if (!reg || !parse_file || !parse_code || !set_config_value ||
      !pre_build_callback_runner || !post_build_callback_runner) {
    return 2;
  }
  // TODO(pebers): it would be nicer if we could get rid of the explicit types here; something
  //               like reg("_add_target", typeof(AddTarget), AddTarget) would be sweet.
  //               As far as I know this is only possible in C++ using typeid though :(

  if (reg("_log", "void (*)(int64, size_t, char*)", Log) != 1) {
    return 3;  // This happens if Python is available but cffi isn't.
  }
  reg("_add_target", "size_t (*)(size_t, char*, char*, char*, uint8, uint8, uint8, uint8, "
      "uint8, uint8, uint8, uint8, uint8, int64, int64, int64, char*)", AddTarget);
  reg("_add_src", "char* (*)(size_t, char*)", AddSource);
  reg("_add_data", "char* (*)(size_t, char*)", AddData);
  reg("_add_dep", "char* (*)(size_t, char*)", AddDep);
  reg("_add_exported_dep", "char* (*)(size_t, char*)", AddExportedDep);
  reg("_add_tool", "char* (*)(size_t, char*)", AddTool);
  reg("_add_out", "char* (*)(size_t, char*)", AddOutput);
  reg("_add_optional_out", "char* (*)(size_t, char*)", AddOptionalOutput);
  reg("_add_vis", "char* (*)(size_t, char*)", AddVis);
  reg("_add_label", "char* (*)(size_t, char*)", AddLabel);
  reg("_add_hash", "char* (*)(size_t, char*)", AddHash);
  reg("_add_licence", "char* (*)(size_t, char*)", AddLicence);
  reg("_add_test_output", "char* (*)(size_t, char*)", AddTestOutput);
  reg("_add_require", "char* (*)(size_t, char*)", AddRequire);
  reg("_add_provide", "char* (*)(size_t, char*, char*)", AddProvide);
  reg("_add_named_src", "char* (*)(size_t, char*, char*)", AddNamedSource);
  reg("_add_command", "char* (*)(size_t, char*, char*)", AddCommand);
  reg("_add_test_command", "char* (*)(size_t, char*, char*)", AddTestCommand);
  reg("_set_container_setting", "char* (*)(size_t, char*, char*)", SetContainerSetting);
  reg("_glob", "char** (*)(char*, char**, long long, char**, long long, uint8)", Glob);
  reg("_get_include_file", "char* (*)(size_t, char*)", GetIncludeFile);
  reg("_get_subinclude_file", "char* (*)(size_t, char*)", GetSubincludeFile);
  reg("_get_labels", "char** (*)(size_t, char*, char*)", GetLabels);
  reg("_set_pre_build_callback", "char** (*)(void*, char*, size_t)", SetPreBuildFunction);
  reg("_set_post_build_callback", "char** (*)(void*, char*, size_t)", SetPostBuildFunction);
  reg("_add_dependency", "char* (*)(size_t, char*, char*, uint8)", AddDependency);
  reg("_add_output", "char* (*)(size_t, char*, char*)", AddOutputPost);
  reg("_add_licence_post", "char* (*)(size_t, char*, char*)", AddLicencePost);
  reg("_get_command", "char* (*)(size_t, char*, char*)", GetCommand);
  reg("_set_command", "char* (*)(size_t, char*, char*, char*)", SetCommand);
  reg("_is_valid_target_name", "uint8 (*)(char*)", IsValidTargetName);
  return 0;
}
