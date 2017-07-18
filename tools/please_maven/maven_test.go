// This is a more or less end-to-end test with a fake web server on the whole package
// as a black box. Expected outputs are taken from the older version of the tool, so
// may not be 100% correct, but empirically they are pretty reasonable and so are
// a good place to start for testing the new version.

package maven

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"cli"
	"sort"
)

// concurrency is the number of concurrent goroutines we use during the test.
// TODO(peterebden): Make this configurable so we test with different numbers.
const concurrency = 10

// Packages that we exclude (they should be test-only dependencies but aren't marked as such)
var excludes = []string{"junit", "easymock", "easymockclassextension"}

var server *httptest.Server
var errorProne, grpc []Artifact

func TestAllDependenciesGRPC(t *testing.T) {
	f := NewFetch(server.URL, excludes, nil)
	expected := []string{
		"io.grpc:grpc-auth:1.1.2:src:BSD 3-Clause",
		"io.grpc:grpc-core:1.1.2:src:BSD 3-Clause",
		"com.google.guava:guava:20.0:src",
		"com.google.errorprone:error_prone_annotations:2.0.11:src",
		"com.google.code.findbugs:jsr305:3.0.0:src:The Apache Software License, Version 2.0",
		"io.grpc:grpc-context:1.1.2:src:BSD 3-Clause",
		"com.google.instrumentation:instrumentation-api:0.3.0:src:Apache License, Version 2.0",
		"com.google.auth:google-auth-library-credentials:0.4.0:src",
		"io.grpc:grpc-netty:1.1.2:src:BSD 3-Clause",
		"io.netty:netty-codec-http2:4.1.8.Final:src",
		"io.netty:netty-codec-http:4.1.8.Final:src",
		"io.netty:netty-codec:4.1.8.Final:src",
		"io.netty:netty-transport:4.1.8.Final:src",
		"io.netty:netty-buffer:4.1.8.Final:src",
		"io.netty:netty-common:4.1.8.Final:src",
		"io.netty:netty-resolver:4.1.8.Final:src",
		"io.netty:netty-handler:4.1.8.Final:src",
		"com.google.code.gson:gson:2.7:no_src",
		"io.netty:netty-handler-proxy:4.1.8.Final:src",
		"io.netty:netty-codec-socks:4.1.8.Final:src",
		"io.grpc:grpc-okhttp:1.1.2:src:BSD 3-Clause",
		"com.squareup.okhttp:okhttp:2.5.0:src",
		"com.squareup.okio:okio:1.6.0:no_src",
		"io.grpc:grpc-protobuf:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf:protobuf-java:3.1.0:src",
		"com.google.protobuf:protobuf-java-util:3.1.0:src",
		"io.grpc:grpc-protobuf-lite:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf:protobuf-lite:3.0.1:src",
		"io.grpc:grpc-protobuf-nano:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf.nano:protobuf-javanano:3.0.0-alpha-5:src:New BSD license",
		"io.grpc:grpc-stub:1.1.2:src:BSD 3-Clause",
	}
	actual := AllDependencies(f, grpc, concurrency, false, false)
	assert.Equal(t, expected, actual)
}

func TestAllDependenciesGRPCWithIndent(t *testing.T) {
	f := NewFetch(server.URL, excludes, nil)
	expected := []string{
		"io.grpc:grpc-auth:1.1.2:src:BSD 3-Clause",
		"  io.grpc:grpc-core:1.1.2:src:BSD 3-Clause",
		"    com.google.guava:guava:20.0:src",
		"    com.google.errorprone:error_prone_annotations:2.0.11:src",
		"    com.google.code.findbugs:jsr305:3.0.0:src:The Apache Software License, Version 2.0",
		"    io.grpc:grpc-context:1.1.2:src:BSD 3-Clause",
		"    com.google.instrumentation:instrumentation-api:0.3.0:src:Apache License, Version 2.0",
		"  com.google.auth:google-auth-library-credentials:0.4.0:src",
		"io.grpc:grpc-netty:1.1.2:src:BSD 3-Clause",
		"  io.netty:netty-codec-http2:4.1.8.Final:src",
		"    io.netty:netty-codec-http:4.1.8.Final:src",
		"      io.netty:netty-codec:4.1.8.Final:src",
		"        io.netty:netty-transport:4.1.8.Final:src",
		"          io.netty:netty-buffer:4.1.8.Final:src",
		"            io.netty:netty-common:4.1.8.Final:src",
		"          io.netty:netty-resolver:4.1.8.Final:src",
		"    io.netty:netty-handler:4.1.8.Final:src",
		"    com.google.code.gson:gson:2.7:no_src",
		"  io.netty:netty-handler-proxy:4.1.8.Final:src",
		"    io.netty:netty-codec-socks:4.1.8.Final:src",
		"io.grpc:grpc-okhttp:1.1.2:src:BSD 3-Clause",
		"  com.squareup.okhttp:okhttp:2.5.0:src",
		"    com.squareup.okio:okio:1.6.0:no_src",
		"io.grpc:grpc-protobuf:1.1.2:src:BSD 3-Clause",
		"  com.google.protobuf:protobuf-java:3.1.0:src",
		"  com.google.protobuf:protobuf-java-util:3.1.0:src",
		"  io.grpc:grpc-protobuf-lite:1.1.2:src:BSD 3-Clause",
		"    com.google.protobuf:protobuf-lite:3.0.1:src",
		"io.grpc:grpc-protobuf-nano:1.1.2:src:BSD 3-Clause",
		"  com.google.protobuf.nano:protobuf-javanano:3.0.0-alpha-5:src:New BSD license",
		"io.grpc:grpc-stub:1.1.2:src:BSD 3-Clause",
	}
	actual := AllDependencies(f, grpc, concurrency, true, false)
	assert.Equal(t, expected, actual)
}

func TestAllDependenciesErrorProne(t *testing.T) {
	f := NewFetch(server.URL, nil, nil)
	expected := []string{
		"com.google.errorprone:error_prone_annotation:2.0.14:src",
		"com.google.guava:guava:19.0:no_src",
		"com.google.errorprone:error_prone_check_api:2.0.14:src",
		"com.google.code.findbugs:jsr305:3.0.0:src:The Apache Software License, Version 2.0",
		"org.checkerframework:dataflow:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"org.checkerframework:javacutil:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"com.google.errorprone:javac:1.9.0-dev-r2973-2:src:GNU General Public License, version 2, with the Classpath Exception",
		"com.googlecode.java-diff-utils:diffutils:1.3.0:src:The Apache Software License, Version 2.0",
		"com.google.auto.value:auto-value:1.1:src",
		"com.google.errorprone:error_prone_annotations:2.0.14:no_src",
		"com.github.stephenc.jcip:jcip-annotations:1.0-1:src:Apache License, Version 2.0",
		"org.pcollections:pcollections:2.1.2:src:The MIT License",
		"com.google.auto:auto-common:0.7:src",
		"com.google.code.findbugs:jFormatString:3.0.0:src:GNU Lesser Public License",
	}
	actual := AllDependencies(f, errorProne, concurrency, false, false)
	assert.Equal(t, expected, actual)
}

func TestAllDependenciesErrorProneWithIndent(t *testing.T) {
	f := NewFetch(server.URL, nil, nil)
	expected := []string{
		"com.google.errorprone:error_prone_annotation:2.0.14:src",
		"  com.google.guava:guava:19.0:no_src",
		"com.google.errorprone:error_prone_check_api:2.0.14:src",
		"  com.google.code.findbugs:jsr305:3.0.0:src:The Apache Software License, Version 2.0",
		"  org.checkerframework:dataflow:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"    org.checkerframework:javacutil:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"  com.google.errorprone:javac:1.9.0-dev-r2973-2:src:GNU General Public License, version 2, with the Classpath Exception",
		"  com.googlecode.java-diff-utils:diffutils:1.3.0:src:The Apache Software License, Version 2.0",
		"  com.google.auto.value:auto-value:1.1:src",
		"  com.google.errorprone:error_prone_annotations:2.0.14:no_src",
		"com.github.stephenc.jcip:jcip-annotations:1.0-1:src:Apache License, Version 2.0",
		"org.pcollections:pcollections:2.1.2:src:The MIT License",
		"com.google.auto:auto-common:0.7:src",
		"com.google.code.findbugs:jFormatString:3.0.0:src:GNU Lesser Public License",
	}
	actual := AllDependencies(f, errorProne, concurrency, true, false)
	assert.Equal(t, expected, actual)
}

func TestAllDependenciesTogether(t *testing.T) {
	f := NewFetch(server.URL, excludes, nil)
	expected := []string{
		"com.google.errorprone:error_prone_annotation:2.0.14:src",
		"com.google.guava:guava:20.0:src",
		"com.google.errorprone:error_prone_check_api:2.0.14:src",
		"com.google.code.findbugs:jsr305:3.0.0:src:The Apache Software License, Version 2.0",
		"org.checkerframework:dataflow:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"org.checkerframework:javacutil:1.8.10:src:GNU General Public License, version 2 (GPL2), with the classpath exception|The MIT License",
		"com.google.errorprone:javac:1.9.0-dev-r2973-2:src:GNU General Public License, version 2, with the Classpath Exception",
		"com.googlecode.java-diff-utils:diffutils:1.3.0:src:The Apache Software License, Version 2.0",
		"com.google.auto.value:auto-value:1.1:src",
		"com.google.errorprone:error_prone_annotations:2.0.14:no_src",
		"com.github.stephenc.jcip:jcip-annotations:1.0-1:src:Apache License, Version 2.0",
		"org.pcollections:pcollections:2.1.2:src:The MIT License",
		"com.google.auto:auto-common:0.7:src",
		"com.google.code.findbugs:jFormatString:3.0.0:src:GNU Lesser Public License",
		"io.grpc:grpc-auth:1.1.2:src:BSD 3-Clause",
		"io.grpc:grpc-core:1.1.2:src:BSD 3-Clause",
		"io.grpc:grpc-context:1.1.2:src:BSD 3-Clause",
		"com.google.instrumentation:instrumentation-api:0.3.0:src:Apache License, Version 2.0",
		"com.google.auth:google-auth-library-credentials:0.4.0:src",
		"io.grpc:grpc-netty:1.1.2:src:BSD 3-Clause",
		"io.netty:netty-codec-http2:4.1.8.Final:src",
		"io.netty:netty-codec-http:4.1.8.Final:src",
		"io.netty:netty-codec:4.1.8.Final:src",
		"io.netty:netty-transport:4.1.8.Final:src",
		"io.netty:netty-buffer:4.1.8.Final:src",
		"io.netty:netty-common:4.1.8.Final:src",
		"io.netty:netty-resolver:4.1.8.Final:src",
		"io.netty:netty-handler:4.1.8.Final:src",
		"com.google.code.gson:gson:2.7:no_src",
		"io.netty:netty-handler-proxy:4.1.8.Final:src",
		"io.netty:netty-codec-socks:4.1.8.Final:src",
		"io.grpc:grpc-okhttp:1.1.2:src:BSD 3-Clause",
		"com.squareup.okhttp:okhttp:2.5.0:src",
		"com.squareup.okio:okio:1.6.0:no_src",
		"io.grpc:grpc-protobuf:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf:protobuf-java:3.1.0:src",
		"com.google.protobuf:protobuf-java-util:3.1.0:src",
		"io.grpc:grpc-protobuf-lite:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf:protobuf-lite:3.0.1:src",
		"io.grpc:grpc-protobuf-nano:1.1.2:src:BSD 3-Clause",
		"com.google.protobuf.nano:protobuf-javanano:3.0.0-alpha-5:src:New BSD license",
		"io.grpc:grpc-stub:1.1.2:src:BSD 3-Clause",
	}
	both := append(errorProne, grpc...)
	actual := AllDependencies(f, both, concurrency, false, false)
	assert.Equal(t, expected, actual)
}

func TestBuildRulesErrorProne(t *testing.T) {
	const expected = `maven_jar(
    name = 'jsr305',
    id = 'com.google.code.findbugs:jsr305:3.0.0',
    hash = '',
)

maven_jar(
    name = 'error_prone_annotations',
    id = 'com.google.errorprone:error_prone_annotations:2.0.14',
    hash = '',
    deps = [
        ':junit-dep',
    ],
)

maven_jar(
    name = 'guava',
    id = 'com.google.guava:guava:19.0',
    hash = '',
    deps = [
        ':jsr305',
        ':error_prone_annotations',
        ':j2objc-annotations',
        ':animal-sniffer-annotations',
    ],
)

maven_jar(
    name = 'error_prone_annotation',
    id = 'com.google.errorprone:error_prone_annotation:2.0.14',
    hash = '',
    deps = [
        ':guava',
        ':junit-dep',
    ],
)

maven_jar(
    name = 'javacutil',
    id = 'org.checkerframework:javacutil:1.8.10',
    hash = '',
)

maven_jar(
    name = 'dataflow',
    id = 'org.checkerframework:dataflow:1.8.10',
    hash = '',
    deps = [
        ':javacutil',
    ],
)

maven_jar(
    name = 'javac',
    id = 'com.google.errorprone:javac:1.9.0-dev-r2973-2',
    hash = '',
)

maven_jar(
    name = 'diffutils',
    id = 'com.googlecode.java-diff-utils:diffutils:1.3.0',
    hash = '',
    deps = [
        ':junit',
    ],
)

maven_jar(
    name = 'auto-value',
    id = 'com.google.auto.value:auto-value:1.1',
    hash = '',
    deps = [
        ':guava-testlib',
        ':compile-testing',
        ':junit',
        ':truth',
    ],
)

maven_jar(
    name = 'error_prone_check_api',
    id = 'com.google.errorprone:error_prone_check_api:2.0.14',
    hash = '',
    deps = [
        ':error_prone_annotation',
        ':jsr305',
        ':dataflow',
        ':javac',
        ':diffutils',
        ':auto-value',
        ':error_prone_annotations',
        ':junit',
        ':hamcrest-core',
        ':truth',
        ':mockito-core',
        ':guava-testlib',
    ],
)

maven_jar(
    name = 'jcip-annotations',
    id = 'com.github.stephenc.jcip:jcip-annotations:1.0-1',
    hash = '',
    deps = [
        ':junit',
    ],
)

maven_jar(
    name = 'pcollections',
    id = 'org.pcollections:pcollections:2.1.2',
    hash = '',
    deps = [
        ':junit',
    ],
)

maven_jar(
    name = 'auto-common',
    id = 'com.google.auto:auto-common:0.7',
    hash = '',
    deps = [
        ':guava',
        ':guava-testlib',
        ':compile-testing',
        ':junit',
        ':truth',
    ],
)

maven_jar(
    name = 'jFormatString',
    id = 'com.google.code.findbugs:jFormatString:3.0.0',
    hash = '',
)

maven_jar(
    name = 'error_prone_core',
    id = 'com.google.errorprone:error_prone_core:2.0.14',
    hash = '',
    deps = [
        ':error_prone_annotation',
        ':error_prone_check_api',
        ':error_prone_test_helpers',
        ':jcip-annotations',
        ':pcollections',
        ':guava',
        ':auto-common',
        ':jFormatString',
        ':jsr305',
        ':dataflow',
        ':javac',
        ':auto-value',
        ':error_prone_annotations',
        ':junit',
        ':hamcrest-core',
        ':hamcrest-library',
        ':truth',
        ':guice',
        ':guice-assistedinject',
        ':guice-servlet',
        ':gin',
        ':mockito-core',
        ':jmock',
        ':jmock-junit4',
        ':protobuf-java',
        ':dagger',
        ':dagger-producers',
        ':auto-factory',
        ':guava-testlib',
        ':compile-testing',
        ':icu4j',
        ':android',
        ':support-v4',
    ],
)`
	f := NewFetch(server.URL, nil, nil)
	actual := AllDependencies(f, errorProne, concurrency, false, true)
	// The rules come out in a different order to the original tool; this doesn't
	// really matter since order of rules in a BUILD file is unimportant.
	expectedSlice := strings.Split(expected, "\n\n")
	sort.Strings(actual)
	sort.Strings(expectedSlice)
	assert.Equal(t, expectedSlice, actual)
}

func TestBuildRulesGRPC(t *testing.T) {
	const expected = `maven_jar(
    name = 'guava',
    id = 'com.google.guava:guava:20.0',
    hash = '',
    deps = [
        ':jsr305',
        ':error_prone_annotations',
        ':j2objc-annotations',
        ':animal-sniffer-annotations',
    ],
)

maven_jar(
    name = 'error_prone_annotations',
    id = 'com.google.errorprone:error_prone_annotations:2.0.11',
    hash = '',
    deps = [
        ':junit-dep',
    ],
)

maven_jar(
    name = 'jsr305',
    id = 'com.google.code.findbugs:jsr305:3.0.0',
    hash = '',
)

maven_jar(
    name = 'grpc-context',
    id = 'io.grpc:grpc-context:1.1.2',
    hash = '',
    deps = [
        ':junit',
        ':mockito-core',
        ':grpc-testing',
    ],
)

maven_jar(
    name = 'instrumentation-api',
    id = 'com.google.instrumentation:instrumentation-api:0.3.0',
    hash = '',
    deps = [
        ':jsr305',
    ],
)

maven_jar(
    name = 'grpc-core',
    id = 'io.grpc:grpc-core:1.1.2',
    hash = '',
    deps = [
        ':guava',
        ':error_prone_annotations',
        ':jsr305',
        ':grpc-context',
        ':instrumentation-api',
        ':junit',
        ':mockito-core',
        ':grpc-testing',
    ],
)

maven_jar(
    name = 'google-auth-library-credentials',
    id = 'com.google.auth:google-auth-library-credentials:0.4.0',
    hash = '',
)

maven_jar(
    name = 'grpc-auth',
    id = 'io.grpc:grpc-auth:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':google-auth-library-credentials',
        ':junit',
        ':mockito-core',
        ':google-auth-library-oauth2-http',
    ],
)

maven_jar(
    name = 'netty-common',
    id = 'io.netty:netty-common:4.1.8.Final',
    hash = '',
    deps = [
        ':javassist',
        ':slf4j-api',
        ':commons-logging',
        ':log4j',
        ':log4j-api',
        ':log4j-core',
        ':junit',
        ':netty-build',
        ':hamcrest-library',
        ':easymock',
        ':easymockclassextension',
        ':jmock-junit4',
        ':mockito-core',
        ':logback-classic',
    ],
)

maven_jar(
    name = 'netty-buffer',
    id = 'io.netty:netty-buffer:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-common',
    ],
)

maven_jar(
    name = 'netty-resolver',
    id = 'io.netty:netty-resolver:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-common',
    ],
)

maven_jar(
    name = 'netty-transport',
    id = 'io.netty:netty-transport:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-buffer',
        ':netty-resolver',
    ],
)

maven_jar(
    name = 'netty-codec',
    id = 'io.netty:netty-codec:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-transport',
        ':protobuf-java',
        ':protobuf-javanano',
        ':jboss-marshalling',
        ':jzlib',
        ':compress-lzf',
        ':lz4',
        ':lzma-java',
        ':jboss-marshalling-serial',
        ':jboss-marshalling-river',
        ':commons-compress',
    ],
)

maven_jar(
    name = 'netty-codec-http',
    id = 'io.netty:netty-codec-http:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-codec',
        ':netty-handler',
        ':jzlib',
    ],
)

maven_jar(
    name = 'netty-handler',
    id = 'io.netty:netty-handler:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-buffer',
        ':netty-transport',
        ':netty-codec',
        ':${tcnative.artifactId}',
        ':bcpkix-jdk15on',
        ':npn-api',
        ':alpn-api',
    ],
)

maven_jar(
    name = 'gson',
    id = 'com.google.code.gson:gson:2.7',
    hash = '',
    deps = [
        ':junit',
    ],
)

maven_jar(
    name = 'netty-codec-http2',
    id = 'io.netty:netty-codec-http2:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-codec-http',
        ':netty-handler',
        ':jzlib',
        ':gson',
    ],
)

maven_jar(
    name = 'netty-codec-socks',
    id = 'io.netty:netty-codec-socks:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-codec',
    ],
)

maven_jar(
    name = 'netty-handler-proxy',
    id = 'io.netty:netty-handler-proxy:4.1.8.Final',
    hash = '',
    deps = [
        ':netty-transport',
        ':netty-codec-socks',
        ':netty-codec-http',
        ':netty-handler',
    ],
)

maven_jar(
    name = 'grpc-netty',
    id = 'io.grpc:grpc-netty:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':netty-codec-http2',
        ':netty-handler-proxy',
        ':junit',
        ':mockito-core',
        ':grpc-testing',
    ],
)

maven_jar(
    name = 'okio',
    id = 'com.squareup.okio:okio:1.6.0',
    hash = '',
    deps = [
        ':animal-sniffer-annotations',
        ':junit',
    ],
)

maven_jar(
    name = 'okhttp',
    id = 'com.squareup.okhttp:okhttp:2.5.0',
    hash = '',
    deps = [
        ':okio',
    ],
)

maven_jar(
    name = 'grpc-okhttp',
    id = 'io.grpc:grpc-okhttp:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':okhttp',
        ':okio',
        ':junit',
        ':mockito-core',
        ':grpc-testing',
        ':grpc-netty',
    ],
)

maven_jar(
    name = 'protobuf-java',
    id = 'com.google.protobuf:protobuf-java:3.1.0',
    hash = '',
    deps = [
        ':junit',
        ':easymock',
        ':easymockclassextension',
    ],
)

maven_jar(
    name = 'protobuf-java-util',
    id = 'com.google.protobuf:protobuf-java-util:3.1.0',
    hash = '',
    deps = [
        ':protobuf-java',
        ':guava',
        ':gson',
        ':junit',
        ':easymock',
        ':easymockclassextension',
    ],
)

maven_jar(
    name = 'protobuf-lite',
    id = 'com.google.protobuf:protobuf-lite:3.0.1',
    hash = '',
    deps = [
        ':junit',
        ':easymock',
        ':easymockclassextension',
    ],
)

maven_jar(
    name = 'grpc-protobuf-lite',
    id = 'io.grpc:grpc-protobuf-lite:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':protobuf-lite',
        ':guava',
        ':junit',
        ':mockito-core',
    ],
)

maven_jar(
    name = 'grpc-protobuf',
    id = 'io.grpc:grpc-protobuf:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':protobuf-java',
        ':guava',
        ':protobuf-java-util',
        ':grpc-protobuf-lite',
        ':junit',
        ':mockito-core',
    ],
)

maven_jar(
    name = 'protobuf-javanano',
    id = 'com.google.protobuf.nano:protobuf-javanano:3.0.0-alpha-5',
    hash = '',
    deps = [
        ':junit',
        ':easymock',
        ':easymockclassextension',
    ],
)

maven_jar(
    name = 'grpc-protobuf-nano',
    id = 'io.grpc:grpc-protobuf-nano:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':protobuf-javanano',
        ':guava',
        ':junit',
        ':mockito-core',
    ],
)

maven_jar(
    name = 'grpc-stub',
    id = 'io.grpc:grpc-stub:1.1.2',
    hash = '',
    deps = [
        ':grpc-core',
        ':junit',
        ':mockito-core',
        ':truth',
        ':grpc-testing',
    ],
)

maven_jar(
    name = 'grpc-all',
    id = 'io.grpc:grpc-all:1.1.2',
    hash = '',
    deps = [
        ':grpc-auth',
        ':grpc-core',
        ':grpc-context',
        ':grpc-netty',
        ':grpc-okhttp',
        ':grpc-protobuf',
        ':grpc-protobuf-lite',
        ':grpc-protobuf-nano',
        ':grpc-stub',
        ':junit',
        ':mockito-core',
    ],
)`
	f := NewFetch(server.URL, excludes, nil)
	actual := AllDependencies(f, grpc, concurrency, false, true)
	// The rules come out in a different order to the original tool; this doesn't
	// really matter since order of rules in a BUILD file is unimportant.
	expectedSlice := strings.Split(expected, "\n\n")
	sort.Strings(actual)
	sort.Strings(expectedSlice)
	assert.Equal(t, expectedSlice, actual)
}

func TestMain(m *testing.M) {
	cli.InitLogging(1) // Suppress informational messages which there can be an awful lot of
	errorProne = []Artifact{{}}
	grpc = []Artifact{{}}
	errorProne[0].FromId("com.google.errorprone:error_prone_core:2.0.14")
	grpc[0].FromId("io.grpc:grpc-all:1.1.2")
	server = httptest.NewServer(http.FileServer(http.Dir("tools/please_maven/test_data")))
	ret := m.Run()
	server.Close()
	os.Exit(ret)
}
