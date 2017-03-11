// Interface code between C and Python; this is used by cffi to generate its interface.
typedef unsigned char uint8;
typedef long long int64;
extern int RegisterCallback(char*, char*, void*);
extern char* ParseFile(char*, char*, size_t);
extern char* ParseCode(char*, char*, size_t);
extern void SetConfigValue(char*, char*);
extern char* PreBuildFunctionRunner(void*, size_t, char*);
extern char* PostBuildFunctionRunner(void*, size_t, char*, char*);
extern char* RunCode(char*);
