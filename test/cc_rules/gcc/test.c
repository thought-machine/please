// Manual test to prove the C rules work.

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

char kTemplate[] = "<testsuite errors=\"0\" failures=\"0\"></testsuite>";

int main(int argc, char** argv) {
  // The following line is legal C but not legal C++.
  char* buf = malloc(strlen(kTemplate) * sizeof(char) + 1);
  strcpy(buf, kTemplate);
  FILE* f = fopen("test.results", "w");
  fprintf(f, "%s\n", buf);
  fclose(f);
  printf("%d %s\n", argc, argv[0]);
  return 0;
}
