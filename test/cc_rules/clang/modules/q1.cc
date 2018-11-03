import f1;
module q1;

int q(int n) {
  if (n == 0) {
    return 1;
  }
  return q(n-1)+n*f(n-1);
}
