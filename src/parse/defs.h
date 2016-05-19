// Interface code between Python & Go. C is a kind of intermediate translation layer.
// This is used by both cgo and cffi to generate their own interfaces.
typedef unsigned char uint8;
typedef long long int64;
typedef size_t (AddTargetCallback)(size_t, char*, char*, char*, uint8, uint8, uint8, uint8, uint8, uint8, uint8, uint8, uint8, int64, int64, int64, char*);
typedef char* (AddStringCallback)(size_t, char*);
typedef char* (AddTwoStringsCallback)(size_t, char*, char*);
typedef char* (AddThreeStringsCallback)(size_t, char*, char*, char*);
typedef char* (AddDependencyCallback)(size_t, char*, char*, uint8);
typedef char* (AddOutputCallback)(size_t, char*, char*);
typedef char** (GlobCallback)(char*, char**, long long, char**, long long, uint8);
typedef char* (GetIncludeFileCallback)(size_t, char*);
typedef char** (GetLabelsCallback)(size_t, char*, char*);
typedef void (SetBuildFunctionCallback)(void*, char*, size_t);
typedef void (LogCallback)(int64, size_t, char*);
typedef uint8 (ValidateCallback)(char*);

extern void RegisterCallback(char*, char*, void*);
extern char* ParseFile(char*, char*, size_t);
extern char* ParseCode(char*, char*, size_t);
extern void SetConfigValue(char*, char*);
extern char* PreBuildFunctionRunner(void*, size_t, char*);
extern char* PostBuildFunctionRunner(void*, size_t, char*, char*);
