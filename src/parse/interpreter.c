#include "interpreter.h"

#include "_cgo_export.h"

void PreBuildFunctionSetter(void* callback, char* bytecode, void* target) {
    SetPreBuildFunction((size_t)callback, bytecode, target);
}

void PostBuildFunctionSetter(void* callback, char* bytecode, void* target) {
    SetPostBuildFunction((size_t)callback, bytecode, target);
}

char* ParseFile(ParseFileCallback* func, char* filename, char* package_name, void* package) {
    return (*func)(filename, package_name, package);
}

void SetConfigValue(SetConfigValueCallback* func, char* name, char* value) {
    func(name, value);
}

char* RunPreBuildFunction(PreBuildCallbackRunner* runner, size_t callback, void* package, char* name) {
    return runner((void*)callback, package, name);
}

char* RunPostBuildFunction(PostBuildCallbackRunner* runner, size_t callback, void* package, char* name, char* output) {
    return runner((void*)callback, package, name, output);
}

int InitialiseInterpreter(char* data, void* vcallbacks) {
  struct PleaseCallbacks* callbacks = (struct PleaseCallbacks*)vcallbacks;
  callbacks->add_target = (AddTargetCallback*)AddTarget;
  callbacks->add_src = AddSource;
  callbacks->add_data = AddData;
  callbacks->add_out = AddOutput;
  callbacks->add_dep = AddDep;
  callbacks->add_exported_dep = AddExportedDep;
  callbacks->add_tool = AddTool;
  callbacks->add_vis = AddVis;
  callbacks->add_label = AddLabel;
  callbacks->add_hash = AddHash;
  callbacks->add_licence = AddLicence;
  callbacks->add_test_output = AddTestOutput;
  callbacks->add_require = AddRequire;
  callbacks->add_provide = AddProvide;
  callbacks->add_named_src = AddNamedSource;
  callbacks->set_container_setting = SetContainerSetting;
  callbacks->glob = Glob;
  callbacks->get_include_file = GetIncludeFile;
  callbacks->get_subinclude_file = GetSubincludeFile;
  callbacks->get_labels = GetLabels;
  callbacks->set_pre_build_function = PreBuildFunctionSetter;
  callbacks->set_post_build_function = PostBuildFunctionSetter;
  callbacks->add_dependency = AddDependency;
  callbacks->add_output = AddOutputPost;
  callbacks->add_licence_post = AddLicencePost;
  callbacks->set_command = SetCommand;
  callbacks->log = Log;
  return pypy_execute_source_ptr(data, callbacks);
}
