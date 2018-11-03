import std.core;
import q1;
import f1;

int main() {
    for (int n = 0; n <= 10; n++) {
       std::cout << "n= " << n << " f(n)= " << f(n) << " q(n)= " << q(n) << std::endl;
    }
}
