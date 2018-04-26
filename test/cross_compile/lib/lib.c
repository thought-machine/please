#include "test/cross_compile/lib/embed.h"

#include <stdlib.h>

int GetAnswer() {
  return atoi(embed_start());
}
