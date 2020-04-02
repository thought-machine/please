import java.net.URL;

public class MyClass {
    public static void main(String... args) {
        URL res = MyClass.class.getResource("some.txt");
        if (res == null) {
            System.exit(1);
        }
    }
}
