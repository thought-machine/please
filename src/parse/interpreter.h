// Interface code between Go and C.
// C is essentially just an intermediate translation layer.

#ifndef _SRC_PARSE_INTERPRETER_H
#define _SRC_PARSE_INTERPRETER_H

#include <dlfcn.h>
#include <stdlib.h>

// AFAICT there isn't a way to call the function pointers directly.
char* PlzParseFile(char* filename, char* contents, char* package_name, size_t package);
char* PlzParseCode(char* filename, char* package_name, size_t package);
void PlzSetConfigValue(char* name, char* value);
char* PlzRunPreBuildFunction(size_t callback, size_t package, char* name);
char* PlzRunPostBuildFunction(size_t callback, size_t package, char* name, char* output);
char* PlzRunCode(char* code);

// Initialises interpreter. Returns 0 on success.
int InitialiseInterpreter(char* parser_location);

// Initialises an interpreter that's statically linked into the binary. Returns 0 on success.
int InitialiseStaticInterpreter(char* preload_so);

#endif  // _SRC_PARSE_INTERPRETER_H
