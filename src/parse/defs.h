// Interface code between Python & Go. C is a kind of intermediate translation layer.
// This is used by both cgo and cffi to generate their own interfaces.
typedef unsigned char uint8;
typedef long long int64;
extern void RegisterCallback(char*, char*, void*);
extern char* ParseFile(char*, char*, size_t);
extern char* ParseCode(char*, char*, size_t);
extern void SetConfigValue(char*, char*);
extern char* PreBuildFunctionRunner(void*, size_t, char*);
extern char* PostBuildFunctionRunner(void*, size_t, char*, char*);
