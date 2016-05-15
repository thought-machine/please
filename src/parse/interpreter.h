#ifndef _SRC_PARSE_INTERPRETER_H
#define _SRC_PARSE_INTERPRETER_H

#include <dlfcn.h>
#include <stdlib.h>
#include <stdio.h>
#include "defs.h"

// AFAICT there isn't a way to call the function pointers directly.
char* ParseFile(char* filename, char* package_name, size_t package);
char* ParseCode(char* filename, char* package_name, size_t package);
void SetConfigValue(char* name, char* value);
char* RunPreBuildFunction(size_t callback, size_t package, char* name);
char* RunPostBuildFunction(size_t callback, size_t package, char* name, char* output);
void PreBuildFunctionSetter(void* callback, char* bytecode, size_t target);
void PostBuildFunctionSetter(void* callback, char* bytecode, size_t target);

// Initialises interpreter. Returns 0 on success.
int InitialiseInterpreter(char* parser_location);

#endif  // _SRC_PARSE_INTERPRETER_H
