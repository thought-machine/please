package build.please.compile;

import com.google.gson.FieldNamingPolicy;
import com.google.gson.Gson;
import com.google.gson.GsonBuilder;

import java.io.File;
import java.io.IOException;
import java.io.StringWriter;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Scanner;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import javax.tools.JavaFileObject;
import javax.tools.StandardJavaFileManager;
import javax.tools.ToolProvider;


public class JavaCompiler {
    /**
     * run reads requests from stdin and sends them to stdout until they are closed.
     */
    public void run() {
        ExecutorService executor = createExecutor();
        final Gson gson = new GsonBuilder()
            .setFieldNamingPolicy(FieldNamingPolicy.LOWER_CASE_WITH_UNDERSCORES)
            .create();
        Scanner input = new Scanner(System.in);
        while (input.hasNext()) {
            final BuildRequest request = gson.fromJson(input.nextLine(), BuildRequest.class);
            executor.submit(new Runnable() {
                    public void run() {
                        try {
                            BuildResponse response = build(request);
                            String msg = gson.toJson(response);
                            synchronized (this) {
                                System.out.println(msg);
                            }
                        } catch (IOException ex) {
                            System.err.printf("I/O error: %s", ex.toString());
                        }
                    }
                });
        }
    }

    /**
     * Overriding this allows customising the threadpool that is used.
     */
    protected ExecutorService createExecutor() {
        return Executors.newFixedThreadPool(Runtime.getRuntime().availableProcessors());
    }

    /**
     * readStdin wraps System.in.read to convert EOF to an exception.
     */
    private void readStdin(byte[] b) throws IOException {
        if (System.in.read(b) == -1) {
            throw new IOException("EOF on stdin, exiting");
        }
    }

    /**
     * build handles building a single build rule.
     */
    public BuildResponse build(BuildRequest request) throws IOException {
        try {
            return reallyBuild(request);
        } catch (Exception ex) {
            return new BuildResponse(request.rule).withMessage(ex.toString());
        }
    }

    /**
     * newCompiler creates a new compiler instance.
     * This is added to allow subclasses to define their own compiler supplier (e.g. ErrorProne).
     */
    public javax.tools.JavaCompiler newCompiler(BuildRequest request) {
        return ToolProvider.getSystemJavaCompiler();
    }

    private BuildResponse reallyBuild(BuildRequest request) throws IOException {
        BuildResponse response = new BuildResponse(request.rule);
        // Try to create the output directory
        File file = new File(request.tempDir + "/_tmp/META-INF");
        if (!file.mkdirs()) {
            return response.withMessage("Failed to create directory " + file.getPath());
        }
        String tmpDir = request.tempDir + "/_tmp";
        DiagnosticReporter reporter = new DiagnosticReporter(response);
        try (StringWriter writer = new StringWriter()) {
            javax.tools.JavaCompiler compiler = newCompiler(request);
            try (StandardJavaFileManager fileManager = compiler.getStandardFileManager(reporter, null, null)) {
                ArrayList<String> srcs = new ArrayList<>();
                for (String src : request.srcs) {
                    srcs.add(src.startsWith("/") ? src : request.tempDir + "/" + src);
                }
                Iterable<? extends JavaFileObject> compilationUnits;
                ArrayList<String> opts = new ArrayList<>();
                opts.addAll(Arrays.asList(
                    "-d", tmpDir,
                    "-s", tmpDir,
                    "-sourcepath", request.tempDir));
                opts.addAll(request.opts);
                if (opts.contains("--src_dir")) {
                    // Special flag that indicates that the sources are actually a directory and we should compile everything in it.
                    opts.remove("--src_dir");
                    FileFinder finder = new FileFinder(".java");
                    Files.walkFileTree(new File(request.tempDir + "/" + request.srcs.get(0)).toPath(), finder);
                    compilationUnits = fileManager.getJavaFileObjectsFromStrings(finder.getFiles());
                } else {
                    compilationUnits = fileManager.getJavaFileObjectsFromStrings(srcs);
                }

                // Find any .jar files and add them to the classpath or module-path
                FileFinder finder = new FileFinder(".jar");
                Files.walkFileTree(new File(request.tempDir).toPath(), finder);

                if (opts.contains("--modular")) {
                    if (!areModulesSupported()) {
                        return response.withMessage("The system java compiler does not support modules");               }
                    // Special flag that indicates that we're trying to use the new java 9 modular JVM
                    opts.remove("--modular");
                    opts.add("--module-path");
                } else {
                    opts.add("-classpath");
                }
                opts.add(finder.joinFiles(':'));
                response.success = compiler.getTask(writer, fileManager, reporter, opts, null, compilationUnits).call();
                return response.withMessage(writer.toString());
            }
        }
    }

    private boolean areModulesSupported() {
        // The java version scheme changed in java 9 from '1.<major>.<minor>' to '<major>.<minor>'
        return !System.getProperty("java.version").startsWith("1.");
    }

    public static void main(String[] args) {
        JavaCompiler compiler = new JavaCompiler();
        compiler.run();
    }
}
