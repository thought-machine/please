package build.please.compile;

import com.google.common.base.Joiner;

import java.io.DataOutputStream;
import java.io.File;
import java.io.IOException;
import java.io.StringWriter;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.file.Files;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

import javax.tools.JavaFileObject;
import javax.tools.StandardJavaFileManager;
import javax.tools.ToolProvider;

import build.please.worker.WorkerProto.BuildRequest;
import build.please.worker.WorkerProto.BuildResponse;;

public class JavaCompiler {
    /**
     * run reads requests from stdin and sends them to stdout until they are closed.
     */
    public void run() {
        ExecutorService executor = Executors.newFixedThreadPool(8);
        final byte[] readBuffer = new byte[4];
        final byte[] writeBuffer = new byte[4];
        // Periodic heartbeat to the parent, to make sure that we exit when it goes away.
        /*
        new Thread(){
            @Override
            public void run() {
                while (true) {

                }
            }
        }.run();
        */

        while (true) {
            try {
                readStdin(readBuffer);
                ByteBuffer bb = ByteBuffer.wrap(readBuffer);
                bb.order(ByteOrder.LITTLE_ENDIAN);
                final byte[] pb = new byte[bb.getInt()];
                readStdin(pb);
                executor.submit(new Runnable() {
                   public void run() {
                       try {
                           BuildResponse response = build(BuildRequest.parseFrom(pb));
                           byte[] arr = response.toByteArray();
                           synchronized (writeBuffer) {
                               ByteBuffer bb = ByteBuffer.wrap(writeBuffer);
                               bb.order(ByteOrder.LITTLE_ENDIAN);
                               bb.putInt(arr.length);
                               System.out.write(bb.array());
                               System.out.write(arr);
                           }
                       } catch (IOException ex) {
                           System.err.printf("I/O error: %s", ex.toString());
                       }
                   }
                });
            } catch (IOException ex) {
                System.err.printf("I/O error: %s", ex.toString());
                break;
            }
        }
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
    public BuildResponse build(BuildRequest request) {
        try {
            return reallyBuild(request);
        } catch (Exception ex) {
            return BuildResponse.newBuilder()
                .setRule(request.getRule())
                .setSuccess(false)
                .addMessages(ex.toString())
                .build();
        }
    }

    private BuildResponse reallyBuild(BuildRequest request) throws IOException {
        BuildResponse.Builder builder = BuildResponse.newBuilder();
        // Try to create the output directory
        File file = new File(request.getTempDir() + "/_tmp/META-INF");
        if (!file.mkdirs()) {
            return builder
                .setRule(request.getRule())
                .addMessages("Failed to create directory " + file.getPath())
                .setSuccess(false)
                .build();
        }
        String tmpDir = request.getTempDir() + "/_tmp";
        DiagnosticReporter reporter = new DiagnosticReporter(builder);
        StringWriter writer = new StringWriter();
        javax.tools.JavaCompiler compiler = ToolProvider.getSystemJavaCompiler();
        StandardJavaFileManager fileManager = compiler.getStandardFileManager(reporter, null, null);
        ArrayList<String> srcs = new ArrayList<String>();
        for (String src : request.getSrcsList()) {
            srcs.add(src.startsWith("/") ? src : request.getTempDir() + "/" + src);
        }
        Iterable<? extends JavaFileObject> compilationUnits = fileManager.getJavaFileObjectsFromStrings(srcs);
        ArrayList<String> opts = new ArrayList<String>();
        opts.addAll(Arrays.asList(
            "-d", tmpDir,
            "-s", tmpDir,
            "-h", tmpDir,
            "-sourcepath", request.getTempDir()));
        opts.addAll(request.getOptsList());
        if (opts.contains("--src_dir")) {
            // Special flag that indicates that the sources are actually a directory and we should compile everything in it.
            opts.remove("--src_dir");
            FileFinder finder = new FileFinder(".java");
            Files.walkFileTree(new File(request.getSrcs(0)).toPath(), finder);
            compilationUnits = fileManager.getJavaFileObjectsFromStrings(finder.getFiles());
        }
        // Find any .jar files and add them to the classpath
        FileFinder finder = new FileFinder(".jar");
        Files.walkFileTree(new File(request.getTempDir()).toPath(), finder);
        opts.add("-classpath");
        opts.add(Joiner.on(':').join(finder.getFiles()));
        return builder
            .setRule(request.getRule())
            .setSuccess(compiler.getTask(writer, fileManager, reporter, opts, null, compilationUnits).call())
            .addMessages(writer.toString())
            .build();
    }

    public static void main(String[] args) {
        JavaCompiler compiler = new JavaCompiler();
        compiler.run();
    }
}
