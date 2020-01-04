// This is a pretty simple example of using select(), copied from the glibc docs.

#include <errno.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/time.h>

int input_timeout(int fd, unsigned int seconds) {
  fd_set set;
  struct timeval timeout;
  FD_ZERO(&set);
  FD_SET(fd, &set);
  timeout.tv_sec = seconds;
  timeout.tv_usec = 0;
  return select(FD_SETSIZE, &set, NULL, NULL, &timeout);
}

int main(int argc, char* argv[]) {
  int ret = input_timeout(STDIN_FILENO, 5);
  fprintf(stderr, "select returned %d.\n", ret);
  return ret == 1 ? 0 : 1;
}
