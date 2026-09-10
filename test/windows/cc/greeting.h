// Deliberately not marked with __declspec(dllexport): MinGW exports every symbol from a DLL
// that declares none explicitly, and a rule that only works with source annotations would be a
// worse test of the rule.
const char *greeting();
