#include <stdio.h>

#include "test/cc_rules/so_deps/lib.h"

int main(int argc, char* argv[]) {
  printf("%d\n", WhatIsTheAnswerToLifeTheUniverseAndEverything());
  return 0;
}
