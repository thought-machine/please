#ifndef _SRC_PARSE_INTERPRETER_H
#define _SRC_PARSE_INTERPRETER_H

#include <stdlib.h>
#include <stdio.h>
#include <PyPy.h>
#include "defs.h"

// AFAICT there isn't a way to call the function pointers directly.
char* ParseFile(ParseFileCallback* func, char* filename, char* package_name, void* package);
void SetConfigValue(SetConfigValueCallback* func, char* name, char* value);
char* RunPreBuildFunction(PreBuildCallbackRunner* runner, size_t callback, void* package, char* name);
char* RunPostBuildFunction(PostBuildCallbackRunner* runner, size_t callback, void* package, char* name, char* output);
void PreBuildFunctionSetter(void* callback, char* bytecode, void* target);
void PostBuildFunctionSetter(void* callback, char* bytecode, void* target);

// Helper functions for handling arrays of strings in C; seems to be nigh impossible in native Go.
inline char** allocateStringArray(int len) { return malloc(len * sizeof(char*)); }
inline void setStringInArray(char** arr, int i, char* s) { arr[i] = s; }
inline char* getStringFromArray(char** arr, int i) { return arr[i]; }

// Initialises interpreter.
// TODO(pebers): Second argument should change to 'struct PleaseCallbacks*' for go1.6.
int InitialiseInterpreter(char* data, struct PleaseCallbacks* callbacks);

#endif  // _SRC_PARSE_INTERPRETER_H
