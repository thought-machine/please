#include "interpreter.h"

#include "_cgo_export.h"

static struct PleaseCallbacks callbacks;

void PreBuildFunctionSetter(void* callback, char* bytecode, size_t target) {
    SetPreBuildFunction((size_t)callback, bytecode, target);
}

void PostBuildFunctionSetter(void* callback, char* bytecode, size_t target) {
    SetPostBuildFunction((size_t)callback, bytecode, target);
}

char* ParseFile(char* filename, char* package_name, size_t package) {
    return (*callbacks.parse_file)(filename, package_name, package);
}

char* ParseCode(char* filename, char* package_name) {
    return (*callbacks.parse_code)(filename, package_name, 0);
}

void SetConfigValue(char* name, char* value) {
    (*callbacks.set_config_value)(name, value);
}

char* RunPreBuildFunction(size_t callback, size_t package, char* name) {
    return (*callbacks.pre_build_callback_runner)((void*)callback, package, name);
}

char* RunPostBuildFunction(size_t callback, size_t package, char* name, char* output) {
    return (*callbacks.post_build_callback_runner)((void*)callback, package, name, output);
}

int InitialiseInterpreter(char* data) {
  callbacks.add_target = (AddTargetCallback*)AddTarget;
  callbacks.add_src = AddSource;
  callbacks.add_data = AddData;
  callbacks.add_out = AddOutput;
  callbacks.add_dep = AddDep;
  callbacks.add_exported_dep = AddExportedDep;
  callbacks.add_tool = AddTool;
  callbacks.add_vis = AddVis;
  callbacks.add_label = AddLabel;
  callbacks.add_hash = AddHash;
  callbacks.add_licence = AddLicence;
  callbacks.add_test_output = AddTestOutput;
  callbacks.add_require = AddRequire;
  callbacks.add_provide = AddProvide;
  callbacks.add_named_src = AddNamedSource;
  callbacks.add_command = AddCommand;
  callbacks.set_container_setting = SetContainerSetting;
  callbacks.glob = Glob;
  callbacks.get_include_file = GetIncludeFile;
  callbacks.get_subinclude_file = GetSubincludeFile;
  callbacks.get_labels = GetLabels;
  callbacks.set_pre_build_function = PreBuildFunctionSetter;
  callbacks.set_post_build_function = PostBuildFunctionSetter;
  callbacks.add_dependency = AddDependency;
  callbacks.add_output = AddOutputPost;
  callbacks.add_licence_post = AddLicencePost;
  callbacks.set_command = SetCommand;
  callbacks.log = Log;
  callbacks.is_valid_target_name = IsValidTargetName;
  return pypy_execute_source_ptr(data, &callbacks);
}
