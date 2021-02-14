package(default_visibility = ["PUBLIC"])

# This is needed to break a circular dependency.
package(jarcat_tool = "//:jarcat_unzip")

go_module(
    name = "logging",
    module = "gopkg.in/op/go-logging.v1",
    version = "b2cb9fa56473e98db8caba80237377e83fe44db5",
)

go_module(
    name = "xcrypto",
    install = [
        "ssh/terminal",
        "cast5",

        # TODO(jpoole): consider if requiring this to be explicit is a good idea or if we should do the
        # `openpgp/openpgp.a` "hack" here
        "openpgp/...",
        "openpgp",
    ],
    module = "golang.org/x/crypto",
    version = "7b85b097bf7527677d54d3220065e966a0e3b613",
)

go_module(
    name = "warnings",
    module = "gopkg.in/warnings.v0",
    version = "v0.1.2",
)

go_module(
    name = "gcfg",
    install = [
        ".",
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
    module = "github.com/pkg/xattr",
    version = "v0.4.1",
    deps = [":xsys"],
)

go_module(
    name = "go-bindata",
    binary = True,
    install = ["go-bindata"],
    module = "github.com/kevinburke/go-bindata",
    strip = ["testdata"],
    version = "v3.21.0",
)

go_module(
    name = "go-flags",
    module = "github.com/thought-machine/go-flags",
    version = "v1.5.0",
    deps = [":xsys"],
)

go_module(
    name = "humanize",
    module = "github.com/dustin/go-humanize",
    version = "8929fe90cee4b2cb9deb468b51fb34eba64d1bf0",
)

go_module(
    name = "net",
    install = ["..."],
    module = "golang.org/x/net",
    version = "136a25c244d3019482a795d728110278d6ba09a4",
    deps = [
        ":text",
        ":xcrypto",
    ],
)

go_module(
    name = "tools",
    install = [
        "cover",
        "go/loader",
        "go/buildutil",
        "go/ast/astutil",
    ],
    module = "golang.org/x/tools",
    version = "2ae76fd1560b622911f444c1e66b70a857e1de67",
)

go_module(
    name = "text",
    install = [
        "secure/...",
        "unicode/...",
        "transform",
        "encoding",
        "encoding/...",
    ],
    module = "golang.org/x/text",
    version = "4e4a3210bb54bb31f6ab2cdca2edcc0b50c420c1",
    deps = [
        ":tools",
    ],
)

go_module(
    name = "cover",
    install = ["cover"],
    module = "github.com/peterebden/tools",
    version = "b2a0db951d2a49fe30dfd9a5f4aad49df75fc12f",
)

go_module(
    name = "grpc",
    exported_deps = [
        ":net",
        ":protobuf-go",
    ],
    install = [
        ".",
        "attributes",
        "backoff",
        "balancer",
        "balancer/base",
        "balancer/grpclb/...",
        "balancer/grpclb",
        "balancer/roundrobin",
        "binarylog/...",
        "codes",
        "connectivity",
        "credentials",
        "credentials/oauth",
        "encoding",
        "encoding/gzip",
        "encoding/proto",
        "grpclog",
        "health",
        "health/grpc_health_v1",
        "internal/...",
        "internal",
        "keepalive",
        "metadata",
        "peer",
        "resolver",
        "resolver/dns",
        "resolver/passthrough",
        "serviceconfig",
        "stats",
        "status",
        "tap",
    ],
    module = "google.golang.org/grpc",
    version = "v1.34.0",
    deps = [
        ":cmp",
        ":genproto_rpc",
        ":oauth2",
        ":protobuf",
        ":uuid",
        ":xsys",
    ],
)

go_mod_download(
    name = "genproto_download",
    module = "google.golang.org/genproto",
    version = "2b5a72b8730b0b16380010cfe5286c42108d88e7",
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
        ":protobuf",
    ],
)

go_module(
    name = "genproto_api",
    download = ":genproto_download",
    install = [
        "googleapis/api/annotations",
        "googleapis/longrunning",
        "googleapis/bytestream",
    ],
    module = "google.golang.org/genproto",
    deps = [
        ":grpc",
        ":net",
        ":protobuf",
    ],
)

go_mod_download(
    name = "protobuf_download",
    module = "github.com/golang/protobuf",
    version = "v1.4.3",
)

go_module(
    name = "protobuf",
    download = ":protobuf_download",
    exported_deps = [
        ":protobuf-go",
    ],
    install = ["..."],
    module = "github.com/golang/protobuf",
    strip = [
        "proto/proto3_proto",
        "conformance",
    ],
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
    name = "protoc-gen-go",
    binary = True,
    download = ":protobuf_download",
    install = ["protoc-gen-go"],
    module = "github.com/golang/protobuf",
    deps = [":protobuf-go"],
)

go_module(
    name = "protoc-gen-go-grpc",
    binary = True,
    module = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    version = "v1.1.0",
    deps = [
        ":grpc",
        ":protobuf",
    ],
)

go_get(
    name = "spew",
    get = "github.com/davecgh/go-spew/spew",
    licences = ["ISC"],
    patch = "spew_omit_empty.patch",
    revision = "ecdeabc65495df2dec95d7c4a4c3e021903035e5",
)

go_module(
    name = "testify",
    install = [
        "assert",
        "require",
    ],
    licences = ["MIT"],
    module = "github.com/stretchr/testify",
    version = "v1.4.0",
    deps = [
        ":difflib",
        ":objx",
        ":spew",
        ":yaml.v2",
    ],
)

go_module(
    name = "difflib",
    install = ["difflib"],
    licences = ["BSD 3-Clause"],
    module = "github.com/pmezard/go-difflib",
    version = "792786c7400a136282c1664665ae0a8db921c6c2",
)

go_module(
    name = "objx",
    licences = ["MIT"],
    module = "github.com/stretchr/objx",
    version = "1a9d0bb9f541897e62256577b352fdbc1fb4fd94",
)

go_module(
    name = "yaml.v2",
    licences = ["Apache 2.0"],
    module = "gopkg.in/yaml.v2",
    version = "v2.2.4",
)

go_module(
    name = "go-isatty",
    module = "github.com/mattn/go-isatty",
    version = "6ca4dbf54d38eea1a992b3c722a76a5d1c4cb25c",
)

go_module(
    name = "levenshtein",
    install = ["levenshtein"],
    module = "github.com/texttheater/golang-levenshtein",
    version = "14026fface0cb806188c85e792a93d625dc37d0f",
)

go_module(
    name = "queue",
    install = ["queue"],
    module = "github.com/Workiva/go-datastructures",
    version = "064f3ea06ab2337e6eaf42052392d14f779e91cb",
)

go_module(
    name = "fsnotify",
    module = "github.com/fsnotify/fsnotify",
    version = "a8a77c9133d2d6fd8334f3260d06f60e8d80a5fb",
    deps = [":xsys"],
)

go_module(
    name = "concurrent-map",
    module = "github.com/streamrail/concurrent-map",
    version = "5fc745307dc80a1883243b978f7e7c0fd5ce7206",
)

go_module(
    name = "grpc-middleware",
    install = [
        "retry",
        "util/metautils",
        "util/backoffutils",
    ],
    module = "github.com/grpc-ecosystem/go-grpc-middleware",
    version = "fa8fef87dcecac0bda02d36abb3c790ab9e0030b",
    deps = [
        ":grpc",
        ":net",
        ":protobuf",
    ],
)

go_module(
    name = "shlex",
    module = "github.com/google/shlex",
    version = "6f45313302b9c56850fc17f99e40caebce98c716",
)

go_module(
    name = "semver",
    install = ["semver"],
    module = "github.com/coreos/go-semver",
    version = "8ab6407b697782a06568d4b7f1db25550ec2e4c6",
)

go_module(
    name = "go-multierror",
    module = "github.com/hashicorp/go-multierror",
    version = "b7773ae218740a7be65057fc60b366a49b538a44",
    deps = [":errwrap"],
)

go_module(
    name = "errwrap",
    module = "github.com/hashicorp/errwrap",
    version = "7554cd9344cec97297fa6649b055a8c98c2a1e55",
)

go_module(
    name = "atime",
    module = "github.com/djherbis/atime",
    version = "89517e96e10b93292169a79fd4523807bdc5d5fa",
)

go_module(
    name = "errgroup",
    install = ["errgroup"],
    module = "golang.org/x/sync",
    version = "457c5828408160d6a47e17645169cf8fa20218c4",
    deps = [":net"],
)

go_module(
    name = "psutil",
    install = [
        "cpu",
        "internal/common",
        "mem",
    ],
    module = "github.com/shirou/gopsutil",
    version = "v2.17.09",
    deps = [":xsys"],
)

go_module(
    name = "xsys",
    install = ["..."],
    module = "golang.org/x/sys",
    version = "765f4ea38db36397e827c4153018aa272eed7835",
)

go_mod_download(
    name = "go-fuzz-download",
    module = "github.com/dvyukov/go-fuzz",
    version = "445b00a1141b27425541ee8d7dc2f524faf202a9",
)

go_module(
    name = "go-fuzz-build",
    binary = True,
    download = ":go-fuzz-download",
    install = ["go-fuzz-build"],
    module = "github.com/dvyukov/go-fuzz",
)

go_module(
    name = "go-fuzz",
    download = ":go-fuzz-download",
    install = [
        "go-fuzz",
        "go-fuzz-dep",
        "go-fuzz-defs",
    ],
    module = "github.com/dvyukov/go-fuzz",
    please_go_install = True,
    deps = [
        ":go-bindata-assetfs",
        ":writerset",
    ],
)

go_module(
    name = "writerset",
    module = "github.com/stephens2424/writerset",
    version = "v1.0.2",
)

go_module(
    name = "go-bindata-assetfs",
    module = "github.com/elazarl/go-bindata-assetfs",
    version = "v1.0.1",
)

go_module(
    name = "godirwalk",
    module = "github.com/karrick/godirwalk",
    strip = [
        "examples",
        "testdata",
        "vendor",
    ],
    version = "v1.7.0",
    deps = [":errors"],
)

go_module(
    name = "errors",
    module = "github.com/pkg/errors",
    version = "30136e27e2ac8d167177e8a583aa4c3fea5be833",
)

go_module(
    name = "gogo-protobuf",
    install = ["proto"],
    module = "github.com/gogo/protobuf",
    version = "v1.3.2",
)

go_module(
    name = "skylark",
    install = ["syntax"],
    module = "go.starlark.net",
    version = "v0.0.0-20210126161401-bc864be25151",
)

go_mod_download(
    name = "buildtools_download",
    module = "github.com/peterebden/buildtools",
    please_go_install = True,
    version = "f7a36c689cc9e038956d51005332c67ade7aa9c6",
)

go_module(
    name = "buildtools",
    download = ":buildtools_download",
    install = ["..."],
    module = "github.com/bazelbuild/buildtools",
    please_go_install = True,
    deps = [
        ":protobuf",
        ":skylark",
    ],
)

go_module(
    name = "jsonrpc2",
    module = "github.com/sourcegraph/jsonrpc2",
    version = "549eb959f029d014d623104d40ab966d159a92de",
)

go_module(
    name = "xz",
    install = [
        ".",
        "lzma",
        "internal/...",
    ],
    module = "github.com/ulikunitz/xz",
    version = "v0.5.6",
)

go_module(
    name = "ar",
    module = "github.com/peterebden/ar",
    version = "a0ae3a11a518fb9559c95ed151ec07865b31980b",
)

go_module(
    name = "cli-init",
    module = "github.com/peterebden/go-cli-init/v3",
    version = "v3.0.0",
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
    module = "github.com/sourcegraph/go-diff",
    version = "v0.5.1",
    deps = [
        ":gogo-protobuf",
        ":sqs-pbtypes",
    ],
)

go_module(
    name = "sqs-pbtypes",
    module = "sourcegraph.com/sqs/pbtypes",
    version = "v1.0.0",
    deps = [":gogo-protobuf"],
)

go_module(
    name = "remote-apis",
    install = ["build/..."],
    module = "github.com/bazelbuild/remote-apis",
    version = "2846a67ac8feb5001e9f704b66f5acc1e90f1ade",
    deps = [
        ":genproto_api",
        ":grpc",
        ":protobuf",
    ],
)

go_mod_download(
    name = "remote-apis-sdks-download",
    module = "github.com/peterebden/remote-apis-sdks",
    please_go_install = True,
    version = "271eb5b72e7dca1dda1d4dee87d99a17194cf146",
)

go_module(
    name = "remote-apis-sdks",
    download = ":remote-apis-sdks-download",
    install = ["go/..."],
    module = "github.com/bazelbuild/remote-apis-sdks",
    please_go_install = True,
    deps = [
        ":cmp",
        ":errgroup",
        ":errors",
        ":genproto_api",
        ":glog",
        ":grpc",
        ":oauth2",
        ":pborman_uuid",
        ":protobuf",
        ":remote-apis",
    ],
)

go_module(
    name = "cmp",
    install = [
        "cmp",
        "cmp/...",
    ],
    module = "github.com/google/go-cmp",
    version = "b1c9c4891a6525d98001fea424c8926c6d77bb56",
)

go_get(
    name = "glog",
    get = "github.com/golang/glog",
    patch = "glog_disable.patch",
    revision = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
)

go_module(
    name = "cloud-metadata",
    install = ["compute/metadata"],
    module = "cloud.google.com/go",
    version = "v0.40.0",
)

go_module(
    name = "oauth2",
    install = [
        ".",
        "google",
        "internal",
        "jws",
        "jwt",
    ],
    module = "golang.org/x/oauth2",
    version = "d668ce993890a79bda886613ee587a69dd5da7a6",
    visibility = ["//third_party/go/..."],
    deps = [
        ":cloud-metadata",
        "//third_party/go:net",
    ],
)

go_module(
    name = "uuid",
    module = "github.com/google/uuid",
    version = "v1.1.1",
)

go_module(
    name = "pborman_uuid",
    module = "github.com/pborman/uuid",
    version = "v1.2.0",
    deps = [":uuid"],
)

go_module(
    name = "lsp",
    module = "github.com/sourcegraph/go-lsp",
    version = "0c7d621186c1c8684f3d97868f6970b326609cc7",
)

go_module(
    name = "promptui",
    install = [
        ".",
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
    module = "github.com/chzyer/readline",
    version = "2972be24d48e78746da79ba8e24e8b488c9880de",
)

go_module(
    name = "ansiterm",
    install = [
        ".",
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
    module = "github.com/lunixbochs/vtclean",
    version = "v1.0.0",
)

go_module(
    name = "colorable",
    module = "github.com/mattn/go-colorable",
    version = "v0.1.2",
    deps = [
        ":go-isatty",
    ],
)

go_module(
    name = "sri",
    module = "github.com/peterebden/go-sri",
    test_only = True,
    version = "v1.0.0",
)

go_module(
    name = "go-cleanhttp",
    module = "github.com/hashicorp/go-cleanhttp",
    version = "v0.5.1",
)

go_module(
    name = "go-retryablehttp",
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.6.7",
    deps = [
        ":go-cleanhttp",
    ],
)

go_module(
    name = "goldmark",
    install = [
        "...",
        ".",
    ],
    module = "github.com/yuin/goldmark",
    version = "v1.2.1",
)

go_module(
    name = "csslex",
    module = "github.com/x1ddos/csslex",
    version = "7894d8ab8bfebee68b6c32bacec8f5189b45c67f",
)

go_module(
    name = "blackfriday",
    module = "gopkg.in/russross/blackfriday.v2",
    version = "v2.0.1",
    deps = [":sanitized_anchor_name"],
)

go_module(
    name = "sanitized_anchor_name",
    module = "github.com/shurcooL/sanitized_anchor_name",
    version = "v1.0.0",
)

genrule(
    name = "claat_download",
    out = "claat_download",
    cmd = "git clone https://github.com/googlecodelabs/tools $OUT && cd $OUT && git checkout -q v2.2.4",
)

go_module(
    name = "claat",
    binary = True,
    download = ":claat_download",
    install = ["claat"],
    module = "github.com/googlecodelabs/tools",
    deps = [
        ":blackfriday",
        ":csslex",
        ":goldmark",
        ":oauth2",
    ],
)

go_module(
    name = "blake3",
    module = "lukechampine.com/blake3",
    version = "v1.1.4",
    deps = [":cpuid"],
)

go_module(
    name = "cpuid",
    module = "github.com/klauspost/cpuid",
    version = "v1.3.1",
)

go_toolchain(
    name = "toolchain",
    version = "1.15.7",
)
