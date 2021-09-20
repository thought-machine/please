package(default_visibility = ["PUBLIC"])

# This is needed to break a circular dependency.
package(jarcat_tool = "//:jarcat_unzip")

go_toolchain(
    name = "toolchain",
    hashes = [
        "da4e3e3c194bf9eed081de8842a157120ef44a7a8d7c820201adae7b0e28b20b",  # darwin_arm64
        "355bd544ce08d7d484d9d7de05a71b5c6f5bc10aa4b316688c2192aeb3dacfd1",  # darwin_amd64
        "6bf89fc4f5ad763871cf7eac80a2d594492de7a818303283f1366a7f6a30372d",  # linux_amd64
        "15c184c83d99441d719da201b26256455eee85a808747c404b4183e9aa6c64b4",  # freebsd_amd64
    ],
    strip_srcs = True,
    version = "1.17",
)

go_module(
    name = "logging",
    licences = ["BSD-3-Clause"],
    module = "gopkg.in/op/go-logging.v1",
    version = "b2cb9fa56473e98db8caba80237377e83fe44db5",
    visibility = ["PUBLIC"],
)

go_module(
    name = "xcrypto",
    install = [
        "cast5",
        "ssh/terminal",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/crypto",
    version = "v0.0.0-20210711020723-a769d52b0f97",
    visibility = ["PUBLIC"],
    deps = [":xterm"],
)

go_module(
    name = "xterm",
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/term",
    version = "v0.0.0-20210615171337-6886f2dfbf5b",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "go-crypto",
    install = [
        "internal/syscall/...",
        "openpgp/...",
        "eax",
        "curve25519",
        "cast5",
        "internal/randutil",
        "bitcurves",
        "brainpool",
        "internal/byteutil",
        "rand",
        "rsa",
        "ocb",
        "ed25519",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/ProtonMail/go-crypto",
    version = "v0.0.0-20210329181949-3900d675f39b",
    visibility = ["PUBLIC"],
)

go_module(
    name = "warnings",
    licences = ["BSD-2-Clause"],
    module = "gopkg.in/warnings.v0",
    version = "v0.1.2",
    visibility = ["PUBLIC"],
)

go_module(
    name = "gcfg",
    install = [
        "...",
    ],
    module = "github.com/peterebden/gcfg",
    version = "v1.3.0",
    deps = [
        ":warnings",
    ],
)

go_module(
    name = "xattr",
    licences = ["BSD-2-Clause"],
    module = "github.com/pkg/xattr",
    version = "v0.4.0",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "go-bindata",
    binary = True,
    install = [
        "go-bindata",
        ".",
    ],
    module = "github.com/kevinburke/go-bindata",
    strip = ["testdata"],
    version = "v3.22.0+incompatible",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-flags",
    licences = ["BSD-3-Clause"],
    module = "github.com/thought-machine/go-flags",
    version = "v1.6.0",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "humanize",
    licences = ["MIT"],
    module = "github.com/dustin/go-humanize",
    version = "8929fe90cee4b2cb9deb468b51fb34eba64d1bf0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "net",
    install = ["..."],
    module = "golang.org/x/net",
    version = "853a461950ffd0dac439017081f0f3bd9242bb17",
    deps = [
        ":text",
        ":xcrypto",
    ],
)

go_module(
    name = "tools",
    install = [
        "cover",
        "internal/gocommand",
        "internal/event/label",
        "go/gcexportdata",
        "internal/typesinternal",
        "go/packages",
        "benchmark/parse",
        "internal/packagesinternal",
        "internal/typeparams",
        "go/internal/packagesdriver",
        "go/buildutil",
        "go/ast/astutil",
        "internal/event/keys",
        "go/loader",
        "go/internal/gcimporter",
        "go/internal/cgo",
        "internal/event",
        "internal/event/core",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/tools",
    version = "v0.1.5",
    visibility = ["PUBLIC"],
    deps = [
        ":mod",
        ":xerrors",
        ":xsys",
    ],
)

go_module(
    name = "text",
    install = [
        "secure/...",
        "unicode/...",
        "encoding/...",
        "transform",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/text",
    version = "v0.3.6",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cover",
    install = ["cover"],
    licences = ["BSD-3-Clause"],
    module = "github.com/peterebden/tools",
    version = "b2a0db951d2a49fe30dfd9a5f4aad49df75fc12f",
    visibility = ["PUBLIC"],
)

go_module(
    name = "grpc",
    install = [
        "balancer/grpclb/...",
        "binarylog/...",
        "metadata",
        "serviceconfig",
        "peer",
        "health",
        "keepalive",
        "internal/binarylog",
        "resolver",
        "status",
        "balancer",
        "internal/grpclog",
        "internal/balancerload",
        "connectivity",
        "internal/grpcutil",
        "internal/resolver",
        "encoding/proto",
        "grpclog",
        "internal/resolver/unix",
        "internal/channelz",
        "internal/resolver/passthrough",
        "balancer/base",
        "attributes",
        "encoding",
        "encoding/gzip",
        "internal/transport",
        "internal/backoff",
        "internal/syscall",
        "internal/metadata",
        "credentials/oauth",
        "resolver/dns",
        "internal/transport/networktype",
        "internal/credentials",
        "internal/envconfig",
        "internal/status",
        "stats",
        "codes",
        "credentials",
        "internal/serviceconfig",
        "internal/grpcsync",
        "internal/buffer",
        "internal/resolver/dns",
        "internal",
        "internal/grpcrand",
        "health/grpc_health_v1",
        "tap",
        ".",
        "backoff",
        "resolver/passthrough",
        "balancer/roundrobin",
    ],
    licences = ["Apache-2.0"],
    module = "google.golang.org/grpc",
    version = "v1.40.0",
    visibility = ["PUBLIC"],
    deps = [
        ":genproto_rpc",
        ":net",
        ":oauth2",
        ":protobuf",
        ":protobuf-go",
        ":xsys",
    ],
)

go_mod_download(
    name = "genproto_download",
    licences = ["Apache-2.0"],
    module = "google.golang.org/genproto",
    version = "v0.0.0-20210917145530-b395a37504d4",
)

# genproto stuff for grpc to resolve cyclic deps
go_module(
    name = "genproto_rpc",
    download = ":genproto_download",
    install = [
        "googleapis/rpc/...",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":protobuf-go",
    ],
)

go_module(
    name = "genproto_api",
    download = ":genproto_download",
    exported_deps = [":genproto_rpc"],
    install = [
        "googleapis/bytestream",
        "googleapis/api/annotations",
        "googleapis/longrunning",
    ],
    module = "google.golang.org/genproto",
    visibility = ["PUBLIC"],
    deps = [
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "bundler",
    install = ["support/bundler"],
    licences = ["BSD-3-Clause"],
    module = "google.golang.org/api",
    version = "v0.30.0",
    visibility = ["PUBLIC"],
    deps = [":errgroup"],
)

go_mod_download(
    name = "protobuf_download",
    licences = ["BSD-3-Clause"],
    module = "github.com/golang/protobuf",
    version = "v1.5.2",
)

go_module(
    name = "protobuf",
    download = ":protobuf_download",
    exported_deps = [
        ":protobuf-go",
    ],
    install = ["..."],
    module = "github.com/golang/protobuf",
)

go_module(
    name = "protoc-gen-go",
    binary = True,
    download = ":protobuf_download",
    exported_deps = [":protobuf"],
    install = ["protoc-gen-go"],
    module = "github.com/golang/protobuf",
    visibility = ["PUBLIC"],
    deps = [":protobuf-go"],
)

go_module(
    name = "protobuf-go",
    install = ["..."],
    module = "google.golang.org/protobuf",
    version = "v1.25.0",
    deps = [
        ":cmp",
    ],
)

go_module(
    name = "xerrors",
    install = ["..."],
    module = "golang.org/x/xerrors",
    version = "5ec99f83aff198f5fbd629d6c8d8eb38a04218ca",
)

go_module(
    name = "protoc-gen-go-grpc",
    binary = True,
    module = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    version = "v1.1.0",
    visibility = ["PUBLIC"],
    deps = [":protobuf-go"],
)

go_module(
    name = "spew",
    install = ["spew"],
    licences = ["ISC"],
    module = "github.com/davecgh/go-spew",
    patch = "spew_omit_empty.patch",
    version = "ecdeabc65495df2dec95d7c4a4c3e021903035e5",
    visibility = ["PUBLIC"],
)

go_module(
    name = "testify",
    install = [
        "assert",
        "require",
    ],
    licences = ["MIT"],
    module = "github.com/stretchr/testify",
    version = "v1.7.0",
    visibility = ["PUBLIC"],
    deps = [
        ":difflib",
        ":spew",
        ":yaml",
    ],
)

go_module(
    name = "difflib",
    install = ["difflib"],
    licences = ["BSD-3-Clause"],
    module = "github.com/pmezard/go-difflib",
    version = "792786c7400a136282c1664665ae0a8db921c6c2",
    visibility = ["PUBLIC"],
)

go_module(
    name = "objx",
    licences = ["MIT"],
    module = "github.com/stretchr/objx",
    version = "1a9d0bb9f541897e62256577b352fdbc1fb4fd94",
    visibility = ["PUBLIC"],
)

go_module(
    name = "yaml",
    licences = ["MIT"],
    module = "gopkg.in/yaml.v3",
    version = "v3.0.0-20200313102051-9f266ea9e77c",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-isatty",
    licences = ["MIT"],
    module = "github.com/mattn/go-isatty",
    version = "6ca4dbf54d38eea1a992b3c722a76a5d1c4cb25c",
    visibility = ["PUBLIC"],
)

go_module(
    name = "levenshtein",
    install = ["levenshtein"],
    licences = ["MIT"],
    module = "github.com/texttheater/golang-levenshtein",
    version = "14026fface0cb806188c85e792a93d625dc37d0f",
    visibility = ["PUBLIC"],
)

go_module(
    name = "fsnotify",
    licences = ["BSD-3-Clause"],
    module = "github.com/fsnotify/fsnotify",
    version = "v1.4.7",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "concurrent-map",
    licences = ["MIT"],
    module = "github.com/streamrail/concurrent-map",
    version = "5fc745307dc80a1883243b978f7e7c0fd5ce7206",
    visibility = ["PUBLIC"],
)

go_module(
    name = "grpc-middleware",
    install = [
        "retry",
        "util/metautils",
        "util/backoffutils",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/grpc-ecosystem/go-grpc-middleware",
    version = "v1.0.1-0.20190118093823-f849b5445de4",
    visibility = ["PUBLIC"],
    deps = [
        ":grpc",
        ":net",
    ],
)

go_module(
    name = "shlex",
    licences = ["Apache-2.0"],
    module = "github.com/google/shlex",
    version = "6f45313302b9c56850fc17f99e40caebce98c716",
    visibility = ["PUBLIC"],
)

go_module(
    name = "semver",
    install = ["semver"],
    licences = ["Apache-2.0"],
    module = "github.com/coreos/go-semver",
    version = "8ab6407b697782a06568d4b7f1db25550ec2e4c6",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-multierror",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-multierror",
    version = "v1.0.0",
    visibility = ["PUBLIC"],
    deps = [":errwrap"],
)

go_module(
    name = "errwrap",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/errwrap",
    version = "7554cd9344cec97297fa6649b055a8c98c2a1e55",
    visibility = ["PUBLIC"],
)

go_module(
    name = "atime",
    licences = ["MIT"],
    module = "github.com/djherbis/atime",
    version = "89517e96e10b93292169a79fd4523807bdc5d5fa",
    visibility = ["PUBLIC"],
)

go_module(
    name = "errgroup",
    install = [
        "errgroup",
        "semaphore",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/sync",
    version = "036812b2e83c0ddf193dd5a34e034151da389d09",
    visibility = ["PUBLIC"],
)

go_module(
    name = "psutil",
    install = [
        "cpu",
        "internal/common",
        "mem",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/shirou/gopsutil/v3",
    version = "v3.21.2",
    visibility = ["PUBLIC"],
    deps = [
        ":sysconf",
        ":xsys",
    ],
)

go_module(
    name = "sysconf",
    licences = ["BSD-3-Clause"],
    module = "github.com/tklauser/go-sysconf",
    version = "v0.3.4",
    visibility = ["PUBLIC"],
    deps = [
        ":numcpus",
        ":xsys",
    ],
)

go_module(
    name = "numcpus",
    licences = ["Apache-2.0"],
    module = "github.com/tklauser/numcpus",
    version = "v0.2.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "63515b42dcdf9544f4e6a02fd7632793fde2f72d",
)

go_mod_download(
    name = "go-fuzz-download",
    module = "github.com/dvyukov/go-fuzz",
    version = "v0.0.0-20210914135545-4980593459a1",
)

go_module(
    name = "go-fuzz-build",
    binary = True,
    download = ":go-fuzz-download",
    install = [
        "internal/go-fuzz-types",
        "go-fuzz/versifier",
        "go-fuzz/internal/pcg",
        "go-fuzz-build",
    ],
    module = "github.com/dvyukov/go-fuzz",
    deps = [":tools"],
)

go_module(
    name = "go-fuzz",
    download = ":go-fuzz-download",
    exported_deps = [":go-fuzz-build"],
    install = [
        "go-fuzz",
        "go-fuzz-dep",
        "go-fuzz-defs",
    ],
    module = "github.com/dvyukov/go-fuzz",
    visibility = ["PUBLIC"],
    deps = [
        ":go-bindata-assetfs",
        ":tools",
        ":writerset",
    ],
)

go_module(
    name = "writerset",
    module = "github.com/stephens2424/writerset",
    version = "v1.0.2",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-bindata-assetfs",
    module = "github.com/elazarl/go-bindata-assetfs",
    version = "v1.0.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "godirwalk",
    licences = ["BSD-2-Clause"],
    module = "github.com/karrick/godirwalk",
    strip = [
        "examples",
        "testdata",
        "vendor",
    ],
    version = "v1.7.8",
    visibility = ["PUBLIC"],
    deps = [":errors"],
)

go_module(
    name = "errors",
    licences = ["BSD-2-Clause"],
    module = "github.com/pkg/errors",
    version = "v0.9.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "gogo-protobuf",
    install = ["proto"],
    licences = ["BSD-3-Clause"],
    module = "github.com/gogo/protobuf",
    version = "v1.3.2",
    visibility = ["PUBLIC"],
)

go_module(
    name = "skylark",
    install = ["syntax"],
    module = "go.starlark.net",
    version = "v0.0.0-20210126161401-bc864be25151",
    visibility = ["PUBLIC"],
)

go_mod_download(
    name = "buildtools_download",
    module = "github.com/peterebden/buildtools",
    version = "f7a36c689cc9e038956d51005332c67ade7aa9c6",
)

go_module(
    name = "buildtools",
    download = ":buildtools_download",
    install = ["..."],
    module = "github.com/bazelbuild/buildtools",
    deps = [
        ":protobuf",
        ":skylark",
    ],
)

go_module(
    name = "jsonrpc2",
    licences = ["MIT"],
    module = "github.com/sourcegraph/jsonrpc2",
    version = "v0.1.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "xz",
    install = [
        "internal/...",
        "lzma",
        ".",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/ulikunitz/xz",
    version = "v0.5.8",
    visibility = ["PUBLIC"],
)

go_module(
    name = "ar",
    licences = ["MIT"],
    module = "github.com/peterebden/ar",
    version = "a0ae3a11a518fb9559c95ed151ec07865b31980b",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cli-init",
    install = ["..."],
    module = "github.com/peterebden/go-cli-init/v5",
    version = "v5.0.2",
    deps = [
        ":go-flags",
        ":humanize",
        ":logging",
        ":xcrypto",
    ],
)

go_module(
    name = "diff",
    install = ["diff"],
    licences = ["MIT"],
    module = "github.com/sourcegraph/go-diff",
    version = "v0.5.1",
    visibility = ["PUBLIC"],
    deps = [
        ":gogo-protobuf",
        ":sqs-pbtypes",
    ],
)

go_module(
    name = "sqs-pbtypes",
    licences = ["Apache-2.0"],
    module = "sourcegraph.com/sqs/pbtypes",
    version = "v0.0.0-20180604144634-d3ebe8f20ae4",
    visibility = ["PUBLIC"],
    deps = [":gogo-protobuf"],
)

go_module(
    name = "remote-apis",
    install = ["build/..."],
    module = "github.com/bazelbuild/remote-apis",
    version = "3e816456ee28f01ab2e0abf72306c1f340c7b229",
    deps = [
        ":genproto_api",
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "remote-apis-sdks",
    install = ["go/..."],
    module = "github.com/bazelbuild/remote-apis-sdks",
    version = "ddba20599b6eafae1669d4da1aa4772fb8c80efd",
    deps = [
        ":bundler",
        ":cmp",
        ":errgroup",
        ":errors",
        ":genproto_api",
        ":glog",
        ":grpc",
        ":oauth2",
        ":pborman_uuid",
        ":protobuf-go",
        ":remote-apis",
        ":zstd",
        ":zstdpool",
    ],
)

go_module(
    name = "cmp",
    install = [
        "cmp/...",
    ],
    module = "github.com/google/go-cmp",
    version = "b1c9c4891a6525d98001fea424c8926c6d77bb56",
)

go_module(
    name = "glog",
    licences = ["Apache-2.0"],
    module = "github.com/golang/glog",
    patch = "glog_disable.patch",
    version = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cloud-metadata",
    install = ["compute/metadata"],
    licences = ["Apache-2.0"],
    module = "cloud.google.com/go",
    version = "v0.40.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "oauth2",
    install = [
        ".",
        "google",
        "internal",
        "jws",
        "jwt",
        "google/internal/externalaccount",
        "authhandler",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/oauth2",
    version = "v0.0.0-20210514164344-f6687ab2804c",
    visibility = ["PUBLIC"],
    deps = [
        ":cloud-metadata",
        ":net",
    ],
)

go_module(
    name = "uuid",
    licences = ["BSD-3-Clause"],
    module = "github.com/google/uuid",
    version = "v1.1.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "pborman_uuid",
    licences = ["BSD-3-Clause"],
    module = "github.com/pborman/uuid",
    version = "v1.2.0",
    visibility = ["PUBLIC"],
    deps = [":uuid"],
)

go_module(
    name = "lsp",
    licences = ["MIT"],
    module = "github.com/sourcegraph/go-lsp",
    version = "0c7d621186c1c8684f3d97868f6970b326609cc7",
    visibility = ["PUBLIC"],
)

go_module(
    name = "promptui",
    install = [
        "...",
    ],
    module = "github.com/manifoldco/promptui",
    version = "v0.7.0",
    deps = [
        ":ansiterm",
        ":readline",
    ],
)

go_module(
    name = "readline",
    licences = ["MIT"],
    module = "github.com/chzyer/readline",
    version = "2972be24d48e78746da79ba8e24e8b488c9880de",
    visibility = ["PUBLIC"],
)

go_module(
    name = "ansiterm",
    install = [
        "...",
    ],
    module = "github.com/juju/ansiterm",
    version = "720a0952cc2ac777afc295d9861263e2a4cf96a1",
    deps = [
        ":colorable",
        ":go-isatty",
        ":vtclean",
    ],
)

go_module(
    name = "vtclean",
    licences = ["MIT"],
    module = "github.com/lunixbochs/vtclean",
    version = "v1.0.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "colorable",
    licences = ["MIT"],
    module = "github.com/mattn/go-colorable",
    version = "v0.0.9",
    visibility = ["PUBLIC"],
    deps = [":go-isatty"],
)

go_module(
    name = "sri",
    module = "github.com/peterebden/go-sri",
    test_only = True,
    version = "v1.0.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-cleanhttp",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-cleanhttp",
    version = "v0.5.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-retryablehttp",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.6.7",
    visibility = ["PUBLIC"],
    deps = [":go-cleanhttp"],
)

go_module(
    name = "goldmark",
    install = [
        "...",
    ],
    module = "github.com/yuin/goldmark",
    version = "v1.2.1",
)

go_module(
    name = "cmap",
    install = [
        "...",
    ],
    module = "github.com/OneOfOne/cmap",
    version = "ccaef7657ab896465978ddf580830288b61f9f6e",
)

go_module(
    name = "csslex",
    module = "github.com/x1ddos/csslex",
    version = "7894d8ab8bfebee68b6c32bacec8f5189b45c67f",
    visibility = ["PUBLIC"],
)

go_module(
    name = "sanitized_anchor_name",
    module = "github.com/shurcooL/sanitized_anchor_name",
    version = "v1.0.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "claat",
    binary = True,
    install = [
        "fetch/drive/auth",
        "parser/md",
        ".",
        "parser",
        "util",
        "render",
        "parser/gdoc",
        "nodes",
        "types",
        "fetch",
        "cmd",
    ],
    module = "github.com/googlecodelabs/tools/claat",
    version = "v0.0.0-20210914205149-d1177395e3b8",
    visibility = ["PUBLIC"],
    deps = [
        ":csslex",
        ":goldmark",
        ":net",
        ":oauth2",
    ],
)

go_module(
    name = "blake3",
    licences = ["MIT"],
    module = "lukechampine.com/blake3",
    version = "v1.1.5",
    visibility = ["PUBLIC"],
    deps = [":cpuid"],
)

go_module(
    name = "cpuid",
    licences = ["MIT"],
    module = "github.com/klauspost/cpuid",
    version = "v1.3.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "zstd",
    install = [
        "zstd/...",
        "fse",
        "huff0",
        "snappy",
        "s2",
        "internal/snapref",
        ".",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/klauspost/compress",
    version = "v1.13.6",
    visibility = ["PUBLIC"],
)

go_module(
    name = "zstdpool",
    licences = ["Apache-2.0"],
    module = "github.com/mostynb/zstdpool-syncpool",
    version = "v0.0.7",
    visibility = ["PUBLIC"],
    deps = [":zstd"],
)

go_module(
    name = "semver2",
    licences = ["MIT"],
    module = "github.com/Masterminds/semver/v3",
    version = "v3.1.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "prometheus",
    install = [
        "prometheus/push",
        "prometheus",
        "prometheus/internal",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/prometheus/client_golang",
    version = "v1.10.0",
    visibility = ["PUBLIC"],
    deps = [
        ":prometheus_common",
        ":prometheus_model",
        ":prometheus_procfs",
        ":protobuf",
        ":quantile",
        ":xxhash",
    ],
)

go_module(
    name = "xxhash",
    licences = ["MIT"],
    module = "github.com/cespare/xxhash/v2",
    version = "v2.1.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "prometheus_model",
    install = ["go"],
    licences = ["Apache-2.0"],
    module = "github.com/prometheus/client_model",
    version = "v0.2.0",
    visibility = ["PUBLIC"],
    deps = [":protobuf"],
)

go_module(
    name = "prometheus_common",
    install = [
        "expfmt",
        "internal/bitbucket.org/ww/goautoneg",
        "model",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/prometheus/common",
    version = "v0.18.0",
    visibility = ["PUBLIC"],
    deps = [
        ":pbutil",
        ":prometheus_model",
        ":protobuf",
    ],
)

go_module(
    name = "prometheus_procfs",
    install = [
        ".",
        "internal/util",
        "internal/fs",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/prometheus/procfs",
    version = "v0.6.0",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "pbutil",
    install = ["pbutil"],
    licences = ["Apache-2.0"],
    module = "github.com/matttproud/golang_protobuf_extensions",
    version = "v1.0.1",
    visibility = ["PUBLIC"],
    deps = [":protobuf"],
)

go_module(
    name = "quantile",
    install = ["quantile"],
    licences = ["MIT"],
    module = "github.com/beorn7/perks",
    version = "v1.0.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "automaxprocs",
    install = [
        "...",
    ],
    licences = ["MIT"],
    module = "go.uber.org/automaxprocs",
    version = "v1.4.0",
)

go_module(
    name = "deferred_regex",
    licences = ["Apache-2.0"],
    module = "github.com/peterebden/go-deferred-regex",
    version = "v1.0.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "snappy",
    module = "github.com/golang/snappy",
    version = "v0.0.3",
    visibility = ["PUBLIC"],
)

go_module(
    name = "mod",
    install = ["semver"],
    module = "golang.org/x/mod",
    version = "v0.5.0",
    visibility = ["PUBLIC"],
)
