#ifndef _SRC_PARSE_INTERPRETER_H
#define _SRC_PARSE_INTERPRETER_H

#include <stdlib.h>
#include <stdio.h>
#include <PyPy.h>
#include "defs.h"

// AFAICT there isn't a way to call the function pointers directly.
char* ParseFile(char* filename, char* package_name, size_t package);
char* ParseCode(char* filename, char* package_name);
void SetConfigValue(char* name, char* value);
char* RunPreBuildFunction(size_t callback, size_t package, char* name);
char* RunPostBuildFunction(size_t callback, size_t package, char* name, char* output);
void PreBuildFunctionSetter(void* callback, char* bytecode, size_t target);
void PostBuildFunctionSetter(void* callback, char* bytecode, size_t target);

// Helper functions for handling arrays of strings in C; seems to be nigh impossible in native Go.
inline char** allocateStringArray(int len) { return malloc(len * sizeof(char*)); }
inline void setStringInArray(char** arr, int i, char* s) { arr[i] = s; }
inline char* getStringFromArray(char** arr, int i) { return arr[i]; }

// Initialises interpreter.
int InitialiseInterpreter(char* data);

#endif  // _SRC_PARSE_INTERPRETER_H
