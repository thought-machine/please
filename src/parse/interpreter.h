// Interface code between Go and C.
// C is essentially just an intermediate translation layer.

#ifndef _SRC_PARSE_INTERPRETER_H
#define _SRC_PARSE_INTERPRETER_H

#include <dlfcn.h>
#include <stdlib.h>

// AFAICT there isn't a way to call the function pointers directly.
char* ParseFile(char* filename, char* package_name, size_t package);
char* ParseCode(char* filename, char* package_name, size_t package);
void SetConfigValue(char* name, char* value);
char* RunPreBuildFunction(size_t callback, size_t package, char* name);
char* RunPostBuildFunction(size_t callback, size_t package, char* name, char* output);
char* RunCode(char* code);

// Initialises interpreter. Returns 0 on success.
int InitialiseInterpreter(char* parser_location);

#endif  // _SRC_PARSE_INTERPRETER_H
