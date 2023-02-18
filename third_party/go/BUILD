package(default_visibility = ["PUBLIC"])

go_toolchain(
    name = "toolchain",
    hashes = [
        "df6509885f65f0d7a4eaf3dfbe7dda327569787e8a0a31cbf99ae3a6e23e9ea8",  # darwin_amd64
        "859e0a54b7fcea89d9dd1ec52aab415ac8f169999e5fdfb0f0c15b577c4ead5e",  # darwin_arm64
        "eca1a8f7b6ff6146efc285eed581096b12b59c1f0488bfe98ed053ab205267ca",  # freebsd_amd64
        "464b6b66591f6cf055bc5df90a9750bf5fbc9d038722bb84a9d56a2bea974be6",  # linux_amd64
        "efa97fac9574fc6ef6c9ff3e3758fb85f1439b046573bf434cccb5e012bd00c8",  # linux_arm64
    ],
    version = "1.19",
)

go_module(
    name = "logging",
    licences = ["BSD-3-Clause"],
    module = "gopkg.in/op/go-logging.v1",
    version = "v1.0.0-20160211212156-b2cb9fa56473",
    visibility = [
        "//src/build:build_step_test",
        "//src/cli/...",
        "//src/parse/asp:asp_test",
        "//src/update:update_test",
        "//test/...",
        "//tools/...",
    ],
)

go_module(
    name = "xcrypto",
    install = [
        "cast5",
        "curve25519",
        "curve25519/internal/field",
        "ed25519",
        "internal/alias",
        "internal/poly1305",
        "nacl/secretbox",
        "ocsp",
        "pbkdf2",
        "salsa20/salsa",
        "scrypt",
        "sha3",
        "ssh/terminal",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/crypto",
    version = "v0.6.0",
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
        "bitcurves",
        "brainpool",
        "cast5",
        "curve25519",
        "eax",
        "ed25519",
        "internal/byteutil",
        "internal/randutil",
        "internal/syscall/...",
        "ocb",
        "openpgp/...",
        "rand",
        "rsa",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/ProtonMail/go-crypto",
    # IMPORTANT!!! Don't update until https://github.com/ProtonMail/go-crypto/issues/88 has been resolved
    version = "3900d675f39ba9686203d137886d74fad620ae87",
    visibility = ["PUBLIC"],
    deps = [":xcrypto"],
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
    module = "github.com/please-build/gcfg",
    version = "9b4d14cddd6148920e11c8bb7ee19369473b1019",
    deps = [
        ":warnings",
    ],
)

go_module(
    name = "xattr",
    licences = ["BSD-2-Clause"],
    module = "github.com/pkg/xattr",
    version = "v0.4.3",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "go-flags",
    licences = ["BSD-3-Clause"],
    module = "github.com/thought-machine/go-flags",
    version = "v1.6.2",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "humanize",
    licences = ["MIT"],
    module = "github.com/dustin/go-humanize",
    version = "v1.0.0",
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
        "benchmark/parse",
        "cover",
        "go/ast/astutil",
        "go/buildutil",
        "go/gcexportdata",
        "go/internal/cgo",
        "go/internal/gcimporter",
        "go/internal/packagesdriver",
        "go/loader",
        "go/packages",
        "internal/event",
        "internal/event/core",
        "internal/event/keys",
        "internal/event/label",
        "internal/gocommand",
        "internal/packagesinternal",
        "internal/typeparams",
        "internal/typesinternal",
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
        "encoding/...",
        "secure/...",
        "transform",
        "unicode/...",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/text",
    version = "v0.3.7",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cover",
    install = ["cover"],
    licences = ["BSD-3-Clause"],
    module = "github.com/peterebden/tools",
    version = "v0.0.0-20190805132753-b2a0db951d2a",
    visibility = ["PUBLIC"],
)

go_module(
    name = "grpc",
    install = [
        "credentials/alts",
        "credentials/alts/internal",
        "credentials/alts/internal/handshaker",
        "credentials/alts/internal/handshaker/service",
        "credentials/alts/internal/proto/grpc_gcp",
        "credentials/alts/internal/authinfo",
        "credentials/alts/internal/conn",
        "internal/googlecloud",
        "balancer/grpclb/...",
        "binarylog/...",
        "internal/status",
        "backoff",
        "encoding/gzip",
        "encoding/proto",
        "health/grpc_health_v1",
        "internal/resolver",
        "internal/resolver/dns",
        "balancer/roundrobin",
        "internal/buffer",
        "peer",
        "internal/grpcsync",
        "internal/grpcutil",
        "resolver",
        "internal/transport/networktype",
        "metadata",
        "attributes",
        "connectivity",
        "credentials/oauth",
        "grpclog",
        "health",
        "internal/grpcrand",
        "serviceconfig",
        "internal/grpclog",
        "internal/metadata",
        ".",
        "codes",
        "encoding",
        "internal/backoff",
        "internal/balancerload",
        "internal/channelz",
        "internal/syscall",
        "resolver/dns",
        "balancer/base",
        "internal",
        "internal/binarylog",
        "resolver/passthrough",
        "tap",
        "credentials",
        "internal/envconfig",
        "internal/transport",
        "stats",
        "status",
        "balancer",
        "internal/credentials",
        "internal/resolver/passthrough",
        "internal/resolver/unix",
        "internal/serviceconfig",
        "keepalive",
        "credentials/google",
        "credentials/insecure",
        "internal/balancer/gracefulswitch",
        "channelz",
        "internal/pretty",
    ],
    licences = ["Apache-2.0"],
    module = "google.golang.org/grpc",
    version = "v1.51.0",
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
    version = "v0.0.0-20221206210731-b1a01be3a5f6",
)

# genproto stuff for grpc to resolve cyclic deps
go_module(
    name = "genproto_rpc",
    download = ":genproto_download",
    install = [
        "googleapis/api",
        "googleapis/rpc/...",
        "googleapis/type/expr",
    ],
    module = "google.golang.org/genproto",
    deps = [":protobuf-go"],
)

go_module(
    name = "genproto_api",
    download = ":genproto_download",
    exported_deps = [":genproto_rpc"],
    install = [
        "googleapis/bytestream",
        "googleapis/cloud/location",
        "googleapis/iam/v1",
        "googleapis/longrunning",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":grpc",
        ":iam_proto",
        ":longrunning",
        ":protobuf-go",
    ],
)

go_module(
    name = "genproto_annotations",
    download = ":genproto_download",
    install = [
        "googleapis/api/annotations",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "bundler",
    install = [
        "googleapi",
        "googleapi/transport",
        "internal",
        "internal/impersonate",
        "internal/third_party/uritemplates",
        "iterator",
        "option",
        "option/internaloption",
        "support/bundler",
        "transport/cert",
        "transport/grpc",
        "transport/http",
        "transport/http/internal/propagation",
        "transport/internal/dca",
    ],
    licences = ["BSD-3-Clause"],
    module = "google.golang.org/api",
    version = "v0.106.0",
    visibility = ["PUBLIC"],
    deps = [
        ":cloud-metadata",
        ":enterprise-certificate-proxy",
        ":errgroup",
        ":go.opencensus.io",
        ":grpc",
        ":net",
        ":oauth2",
    ],
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
    version = "v1.1.1",
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
    version = "v1.0.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "objx",
    licences = ["MIT"],
    module = "github.com/stretchr/objx",
    version = "v0.3.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "yaml",
    licences = ["MIT"],
    module = "gopkg.in/yaml.v3",
    version = "v3.0.0-20210107192922-496545a6307b",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-isatty",
    licences = ["MIT"],
    module = "github.com/mattn/go-isatty",
    version = "v0.0.14",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "levenshtein",
    install = ["levenshtein"],
    licences = ["MIT"],
    module = "github.com/texttheater/golang-levenshtein",
    version = "v1.0.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "fsnotify",
    licences = ["BSD-3-Clause"],
    module = "github.com/fsnotify/fsnotify",
    version = "v1.5.1",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "grpc-middleware",
    install = [
        "retry",
        "util/backoffutils",
        "util/metautils",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/grpc-ecosystem/go-grpc-middleware",
    version = "v1.3.0",
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
    version = "v0.0.0-20181106134648-c34317bd91bf",
    visibility = ["PUBLIC"],
)

go_module(
    name = "semver",
    install = ["semver"],
    licences = ["Apache-2.0"],
    module = "github.com/coreos/go-semver",
    version = "v0.3.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-multierror",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-multierror",
    version = "v1.1.1",
    visibility = ["PUBLIC"],
    deps = [":errwrap"],
)

go_module(
    name = "errwrap",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/errwrap",
    version = "v1.1.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "atime",
    licences = ["MIT"],
    module = "github.com/djherbis/atime",
    version = "v1.1.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "errgroup",
    install = [
        "errgroup",
        "semaphore",
        "singleflight",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/sync",
    version = "v0.1.0",
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
    version = "v3.22.6",
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
    version = "v0.3.9",
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
    version = "v0.3.0",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "v0.0.0-20220315194320-039c03cc5b86",
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
    version = "v1.16.1",
    visibility = ["PUBLIC"],
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
    version = "v0.0.0-20210901212718-87f333178d59",
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
        ".",
        "internal/...",
        "lzma",
    ],
    licences = ["BSD-3-Clause"],
    module = "github.com/ulikunitz/xz",
    version = "v0.5.10",
    visibility = ["PUBLIC"],
)

go_module(
    name = "ar",
    licences = ["MIT"],
    module = "github.com/peterebden/ar",
    version = "v0.0.0-20181115090543-a0ae3a11a518",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cli-init",
    install = ["..."],
    module = "github.com/peterebden/go-cli-init/v5",
    version = "v5.2.1",
    deps = [
        ":go-flags",
        ":humanize",
        ":logging",
        ":xterm",
    ],
)

go_module(
    name = "diff",
    install = ["diff"],
    licences = ["MIT"],
    module = "github.com/sourcegraph/go-diff",
    version = "v0.6.1",
    visibility = ["PUBLIC"],
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
    version = "v0.0.0-20221114180157-e62cf9b8696a",
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
        ":xattr",
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
    version = "v0.0.0-20160126235308-23def4e6c14b",
    visibility = ["PUBLIC"],
)

go_module(
    name = "cloud-metadata",
    install = ["compute/metadata"],
    licences = ["Apache-2.0"],
    module = "cloud.google.com/go",
    version = "v0.65.0",
    visibility = ["PUBLIC"],
)

go_module(
    name = "oauth2",
    install = [
        ".",
        "authhandler",
        "google",
        "google/internal/externalaccount",
        "internal",
        "jws",
        "jwt",
    ],
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/oauth2",
    version = "v0.4.0",
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
    version = "v1.1.2",
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
    version = "v0.0.0-20200429204803-219e11d77f5d",
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
    version = "v0.0.0-20180603132655-2972be24d48e",
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
    version = "v0.1.8",
    visibility = ["PUBLIC"],
    deps = [":go-isatty"],
)

go_module(
    name = "sri",
    module = "github.com/peterebden/go-sri",
    test_only = True,
    version = "v1.1.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-cleanhttp",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-cleanhttp",
    version = "v0.5.2",
    visibility = ["PUBLIC"],
)

go_module(
    name = "go-retryablehttp",
    licences = ["MPL-2.0"],
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.7.0",
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
    name = "csslex",
    module = "github.com/x1ddos/csslex",
    version = "v0.0.0-20160125172232-7894d8ab8bfe",
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
        ".",
        "cmd",
        "fetch",
        "fetch/drive/auth",
        "nodes",
        "parser",
        "parser/gdoc",
        "parser/md",
        "render",
        "types",
        "util",
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
    install = ["..."],
    licences = ["MIT"],
    module = "github.com/zeebo/blake3",
    version = "v0.2.3",
    visibility = ["PUBLIC"],
    deps = [":cpuid"],
)

go_module(
    name = "cpuid",
    licences = ["MIT"],
    module = "github.com/klauspost/cpuid/v2",
    version = "v2.2.1",
    visibility = ["PUBLIC"],
    deps = [":xsys"],
)

go_module(
    name = "zstd",
    install = [
        ".",
        "fse",
        "huff0",
        "internal/snapref",
        "s2",
        "snappy",
        "zstd/...",
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
        "prometheus",
        "prometheus/internal",
        "prometheus/push",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/prometheus/client_golang",
    version = "v1.11.0",
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
    version = "v2.1.2",
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
    version = "v0.30.0",
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
        "internal/fs",
        "internal/util",
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

go_module(
    name = "shellescape",
    licences = ["MIT"],
    module = "github.com/alessio/shellescape",
    version = "v1.4.1",
    visibility = ["PUBLIC"],
)

go_module(
    name = "sigstore",
    install = [
        "pkg/cryptoutils",
        "pkg/signature",
        "pkg/signature/options",
        "pkg/signature/payload",
        "pkg/signature/kms",
        "pkg/signature/kms/gcp",
    ],
    licences = ["Apache-2.0"],
    module = "github.com/sigstore/sigstore",
    version = "v1.5.0",
    deps = [
        ":boulder",
        ":go-containerregistry",
        ":go-tuf",
        ":kms",
        ":ttlcache.v2",
    ],
)

go_module(
    name = "boulder",
    install = [
        "goodkey",
        "core",
        "errors",
        "identifier",
        "probs",
        "revocation",
        "core/proto",
        "sa/proto",
    ],
    licences = ["MPL-2.0"],
    module = "github.com/letsencrypt/boulder",
    version = "cfa524a7a139bd4cf21c4c09ab47b994a96fd11f",
    deps = [
        ":go-jose",
        ":grpc",
        ":protobuf-go",
        ":rocacheck",
        ":xcrypto",
        ":yaml",
    ],
)

go_module(
    name = "go-jose",
    install = [
        ".",
        "cipher",
        "json",
    ],
    licences = ["Apache-2.0"],
    module = "gopkg.in/square/go-jose.v2",
    version = "v2.6.0",
    deps = [":xcrypto"],
)

go_module(
    name = "rocacheck",
    licences = ["MIT"],
    module = "github.com/titanous/rocacheck",
    version = "afe73141d399b0c79c3d6412d4bdcf4c672c496a",
)

go_module(
    name = "go-tuf",
    install = ["encrypted"],
    module = "github.com/theupdateframework/go-tuf",
    version = "v0.5.1",
    deps = [":xcrypto"],
)

go_module(
    name = "go-containerregistry",
    install = ["pkg/name"],
    licences = ["Apache-2.0"],
    module = "github.com/google/go-containerregistry",
    version = "v0.12.0",
    deps = [":go-digest"],
)

go_module(
    name = "go-digest",
    licences = ["Apache-2.0"],
    module = "github.com/opencontainers/go-digest",
    version = "v1.0.0",
)

go_module(
    name = "gax-go.v2",
    install = [
        ".",
        "apierror",
        "apierror/internal/proto",
        "internal",
    ],
    module = "github.com/googleapis/gax-go/v2",
    version = "v2.7.0",
    visibility = ["PUBLIC"],
    deps = [
        ":bundler",
        ":genproto_rpc",
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "genproto",
    download = ":genproto_download",
    exported_deps = [
        ":genproto_api",
        ":genproto_rpc",
    ],
    install = ["googleapis/cloud/kms/v1"],
    module = "google.golang.org/genproto",
    visibility = ["PUBLIC"],
    deps = [
        ":grpc",
        ":kms_1",
    ],
)

go_module(
    name = "longrunning",
    install = ["autogen/longrunningpb"],
    module = "cloud.google.com/go/longrunning",
    version = "v0.4.0",
    visibility = ["PUBLIC"],
    deps = [
        ":genproto_annotations",
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "enterprise-certificate-proxy",
    install = [
        "client",
        "client/util",
    ],
    module = "github.com/googleapis/enterprise-certificate-proxy",
    version = "v0.2.0",
    visibility = ["PUBLIC"],
)

go_mod_download(
    name = "kms_dl",
    module = "cloud.google.com/go/kms",
    version = "v1.8.0",
)

go_module(
    name = "kms_1",
    download = ":kms_dl",
    install = [
        "apiv1/kmspb",
        "internal",
    ],
    module = "cloud.google.com/go/kms",
    deps = [
        ":genproto_api",
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "kms",
    download = ":kms_dl",
    exported_deps = [":kms_1"],
    install = ["apiv1"],
    module = "cloud.google.com/go/kms",
    visibility = ["PUBLIC"],
    deps = [
        ":bundler",
        ":gax-go.v2",
        ":genproto",
        ":genproto_api",
        ":grpc",
        ":iam",
        ":protobuf-go",
    ],
)

go_mod_download(
    name = "iam_dl",
    module = "cloud.google.com/go/iam",
    version = "v0.8.0",
)

go_module(
    name = "iam",
    download = ":iam_dl",
    module = "cloud.google.com/go/iam",
    visibility = ["PUBLIC"],
    deps = [
        ":gax-go.v2",
        ":genproto_api",
        ":grpc",
    ],
)

go_module(
    name = "iam_proto",
    download = ":iam_dl",
    install = ["apiv1/iampb"],
    module = "cloud.google.com/go/iam",
    visibility = ["PUBLIC"],
    deps = [
        ":genproto_annotations",
        ":genproto_rpc",
        ":grpc",
        ":protobuf-go",
    ],
)

go_module(
    name = "go.opencensus.io",
    install = [
        ".",
        "internal",
        "internal/tagencoding",
        "metric/metricdata",
        "metric/metricproducer",
        "plugin/ocgrpc",
        "plugin/ochttp",
        "plugin/ochttp/propagation/b3",
        "resource",
        "stats",
        "stats/internal",
        "stats/view",
        "tag",
        "trace",
        "trace/internal",
        "trace/propagation",
        "trace/tracestate",
    ],
    module = "go.opencensus.io",
    version = "v0.24.0",
    visibility = ["PUBLIC"],
    deps = [
        ":groupcache",
        ":grpc",
    ],
)

go_module(
    name = "groupcache",
    install = ["lru"],
    module = "github.com/golang/groupcache",
    version = "v0.0.0-20200121045136-8c9f03a8e57e",
    visibility = ["PUBLIC"],
)

go_module(
    name = "ttlcache.v2",
    module = "github.com/jellydator/ttlcache/v2",
    version = "v2.11.1",
    visibility = ["PUBLIC"],
    deps = [":errgroup"],
)

go_module(
    name = "x_exp",
    install = [
        "constraints",
        "maps",
        "slices",
    ],
    licences = ["BSD"],
    module = "golang.org/x/exp",
    version = "1de6713980dea447778ef6e71194d5eb54288072",
    visibility = ["PUBLIC"],
)
