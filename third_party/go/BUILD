package(default_visibility = ["PUBLIC"])

# This is needed to break a circular dependency.
package(jarcat_tool = "//:jarcat_unzip")

go_get(
    name = "logging",
    get = "gopkg.in/op/go-logging.v1",
    revision = "b2cb9fa56473e98db8caba80237377e83fe44db5",
)

go_get(
    name = "terminal",
    get = "golang.org/x/crypto/ssh/terminal",
    revision = "7b85b097bf7527677d54d3220065e966a0e3b613",
)

go_get(
    name = "warnings",
    get = "gopkg.in/warnings.v0",
    revision = "v0.1.2",
)

go_get(
    name = "gcfg",
    get = "github.com/peterebden/gcfg/...",
    revision = "v1.3.0",
    deps = [
        ":warnings",
    ],
)

go_get(
    name = "xattr",
    get = "github.com/pkg/xattr",
    revision = "d15dbc2bb0b5da267362b5e066e2c44c1fcff6c7",
    deps = [":unix"],
)

go_get(
    name = "go-bindata",
    binary = True,
    get = "github.com/kevinburke/go-bindata/...",
    revision = "46eb4c183bfc1ebb527d9d19bcded39476302eb8",
    strip = ["testdata"],
)

go_get(
    name = "go-flags",
    get = "github.com/jessevdk/go-flags",
    patch = "flags_subcommand.patch",
    revision = "v1.4.0",
)

go_get(
    name = "humanize",
    get = "github.com/dustin/go-humanize",
    revision = "8929fe90cee4b2cb9deb468b51fb34eba64d1bf0",
)

go_get(
    name = "net",
    get = "golang.org/x/net/...",
    revision = "136a25c244d3019482a795d728110278d6ba09a4",
    deps = [
        ":terminal",
        ":text",
    ],
)

go_get(
    name = "tools",
    get = "golang.org/x/tools",
    install = [
        "cover",
    ],
    revision = "2ae76fd1560b622911f444c1e66b70a857e1de67",
)

go_get(
    name = "text",
    get = "golang.org/x/text/...",
    revision = "4e4a3210bb54bb31f6ab2cdca2edcc0b50c420c1",
    deps = [
        ":tools",
    ],
)

go_get(
    name = "cover",
    get = "github.com/peterebden/tools/cover",
    revision = "b2a0db951d2a49fe30dfd9a5f4aad49df75fc12f",
)

go_get(
    name = "grpc",
    exported_deps = [":net"],
    get = "google.golang.org/grpc",
    install = [
        "",
        "attributes",
        "backoff",
        "balancer",
        "balancer/base",
        "balancer/grpclb/...",
        "balancer/roundrobin",
        "binarylog/...",
        "codes",
        "connectivity",
        "credentials/...",
        "encoding",
        "encoding/gzip",
        "encoding/proto",
        "grpclog",
        "health",
        "health/grpc_health_v1",
        "internal/...",
        "keepalive",
        "metadata",
        "naming",
        "peer",
        "resolver",
        "resolver/dns",
        "resolver/passthrough",
        "serviceconfig",
        "stats",
        "status",
        "tap",
    ],
    repo = "github.com/grpc/grpc-go",
    revision = "v1.26.0",
    deps = [
        ":cmp",
        ":oauth2",
        ":protobuf",
        ":rpccode",
        ":rpcstatus",
        ":unix",
    ],
)

#TODO: build from the actual proto.
go_get(
    name = "rpcstatus",
    get = "google.golang.org/genproto/googleapis/rpc/status",
    revision = "2b5a72b8730b0b16380010cfe5286c42108d88e7",
    deps = [":protobuf"],
)

go_get(
    name = "rpccode",
    get = "google.golang.org/genproto/googleapis/rpc/code",
    revision = "2b5a72b8730b0b16380010cfe5286c42108d88e7",
    deps = [":protobuf"],
)

go_get(
    name = "protobuf",
    get = "github.com/golang/protobuf/...",
    revision = "v1.3.2",
    strip = [
        "proto/proto3_proto",
        "conformance",
    ],
)

go_get(
    name = "protoc-gen-go",
    binary = True,
    get = [],
    install = ["github.com/golang/protobuf/protoc-gen-go"],
    deps = [
        ":protobuf",
    ],
)

go_get(
    name = "spew",
    get = "github.com/davecgh/go-spew/spew",
    patch = "spew_omit_empty.patch",
    revision = "ecdeabc65495df2dec95d7c4a4c3e021903035e5",
)

go_get(
    name = "testify",
    get = "github.com/stretchr/testify",
    install = [
        "assert",
        "require",
        "vendor/github.com/davecgh/go-spew/spew",
        "vendor/github.com/pmezard/go-difflib/difflib",
    ],
    revision = "f390dcf405f7b83c997eac1b06768bb9f44dec18",
    deps = [":spew"],
)

go_get(
    name = "go-isatty",
    get = "github.com/mattn/go-isatty",
    revision = "6ca4dbf54d38eea1a992b3c722a76a5d1c4cb25c",
)

go_get(
    name = "levenshtein",
    get = "github.com/texttheater/golang-levenshtein/levenshtein",
    revision = "14026fface0cb806188c85e792a93d625dc37d0f",
)

go_get(
    name = "queue",
    get = "github.com/Workiva/go-datastructures/queue",
    revision = "064f3ea06ab2337e6eaf42052392d14f779e91cb",
)

go_get(
    name = "fsnotify",
    get = "github.com/fsnotify/fsnotify",
    revision = "a8a77c9133d2d6fd8334f3260d06f60e8d80a5fb",
    deps = [":unix"],
)

go_get(
    name = "concurrent-map",
    get = "github.com/streamrail/concurrent-map",
    revision = "5fc745307dc80a1883243b978f7e7c0fd5ce7206",
)

go_get(
    name = "grpc-middleware",
    get = "github.com/grpc-ecosystem/go-grpc-middleware",
    install = [
        "retry",
        "util/metautils",
        "util/backoffutils",
    ],
    revision = "fa8fef87dcecac0bda02d36abb3c790ab9e0030b",
    deps = [
        ":grpc",
        ":net",
        ":protobuf",
    ],
)

go_get(
    name = "shlex",
    get = "github.com/google/shlex",
    revision = "6f45313302b9c56850fc17f99e40caebce98c716",
)

go_get(
    name = "semver",
    get = "github.com/coreos/go-semver/semver",
    revision = "8ab6407b697782a06568d4b7f1db25550ec2e4c6",
)

go_get(
    name = "go-multierror",
    get = "github.com/hashicorp/go-multierror",
    install = [
        "",
        "vendor/github.com/hashicorp/errwrap",
    ],
    revision = "b7773ae218740a7be65057fc60b366a49b538a44",
)

go_get(
    name = "atime",
    get = "github.com/djherbis/atime",
    revision = "89517e96e10b93292169a79fd4523807bdc5d5fa",
)

go_get(
    name = "errgroup",
    get = "golang.org/x/sync/errgroup",
    revision = "457c5828408160d6a47e17645169cf8fa20218c4",
    deps = [":net"],
)

go_get(
    name = "psutil",
    get = "github.com/shirou/gopsutil",
    install = [
        "cpu",
        "internal/common",
        "mem",
    ],
    revision = "v2.17.09",
    deps = [":unix"],
)

go_get(
    name = "unix",
    get = "golang.org/x/sys/unix",
    revision = "1b2967e3c290b7c545b3db0deeda16e9be4f98a2",
)

go_get(
    name = "openpgp",
    get = "golang.org/x/crypto/openpgp/...",
    revision = "077efaa604f994162e3307fafe5954640763fc08",
    deps = [":cast5"],
)

go_get(
    name = "cast5",
    get = "golang.org/x/crypto/cast5",
    revision = "077efaa604f994162e3307fafe5954640763fc08",
)

go_get(
    name = "go-fuzz-build",
    binary = True,
    get = [],
    install = ["github.com/dvyukov/go-fuzz/go-fuzz-build"],
    deps = [":go-fuzz-dep"],
)

go_get(
    name = "go-fuzz",
    binary = True,
    get = [],
    install = ["github.com/dvyukov/go-fuzz/go-fuzz"],
    deps = [":go-fuzz-dep"],
)

go_get(
    name = "go-fuzz-dep",
    get = "github.com/dvyukov/go-fuzz",
    install = [
        "go-fuzz-dep",
        "go-fuzz-defs",
    ],
    revision = "445b00a1141b27425541ee8d7dc2f524faf202a9",
)

go_get(
    name = "godirwalk",
    get = "github.com/karrick/godirwalk",
    revision = "v1.7.0",
    strip = [
        "examples",
        "testdata",
        "vendor",
    ],
    deps = [":errors"],
)

go_get(
    name = "errors",
    get = "github.com/pkg/errors",
    revision = "30136e27e2ac8d167177e8a583aa4c3fea5be833",
)

go_get(
    name = "gogo-protobuf",
    get = "github.com/gogo/protobuf",
    install = ["proto"],
    revision = "v0.4",
)

go_get(
    name = "skylark",
    get = "github.com/google/skylark/syntax",
    revision = "a5f7082aabed29c0e429c722292c66ec8ecf9591",
)

go_get(
    name = "buildtools",
    get = "github.com/bazelbuild/buildtools/...",
    revision = "0.19.2",
    deps = [
        ":protobuf",
        ":skylark",
    ],
)

go_get(
    name = "buildifier",
    binary = True,
    get = [],
    install = ["github.com/bazelbuild/buildtools/buildifier"],
    deps = [
        ":buildtools",
    ],
)

go_get(
    name = "jsonrpc2",
    get = "github.com/sourcegraph/jsonrpc2",
    revision = "549eb959f029d014d623104d40ab966d159a92de",
)

go_get(
    name = "xz",
    get = "github.com/ulikunitz/xz",
    install = [
        "",
        "lzma",
        "internal/...",
    ],
    revision = "v0.5.6",
)

go_get(
    name = "ar",
    get = "github.com/peterebden/ar",
    revision = "a0ae3a11a518fb9559c95ed151ec07865b31980b",
)

go_get(
    name = "cli-init",
    get = "github.com/peterebden/go-cli-init",
    revision = "v1.2.0",
    deps = [
        ":go-flags",
        ":logging",
        ":terminal",
    ],
)

go_get(
    name = "diff",
    get = "github.com/sourcegraph/go-diff/diff",
    revision = "v0.5.1",
    deps = [
        ":gogo-protobuf",
        ":sqs-pbtypes",
    ],
)

go_get(
    name = "sqs-pbtypes",
    get = "sourcegraph.com/sqs/pbtypes",
    repo = "github.com/sqs/pbtypes",
    revision = "v1.0.0",
    deps = [":gogo-protobuf"],
)

go_get(
    name = "remote-apis",
    get = "github.com/bazelbuild/remote-apis/build/...",
    revision = "2846a67ac8feb5001e9f704b66f5acc1e90f1ade",
    deps = [
        ":annotations",
        ":grpc",
        ":longrunning",
        ":protobuf",
        ":rpcstatus",
    ],
)

go_get(
    name = "remote-apis-sdks",
    get = "github.com/bazelbuild/remote-apis-sdks/go/...",
    repo = "github.com/peterebden/remote-apis-sdks",
    revision = "16b8c03a1b365b4c36c6544b5d03bda429cee2e2",
    deps = [
        ":annotations",
        ":bytestream",
        ":cmp",
        ":errgroup",
        ":errors",
        ":glog",
        ":grpc",
        ":longrunning",
        ":oauth2",
        ":pborman_uuid",
        ":protobuf",
        ":remote-apis",
        ":rpcstatus",
    ],
)

go_get(
    name = "cmp",
    get = "github.com/google/go-cmp/cmp",
    revision = "b1c9c4891a6525d98001fea424c8926c6d77bb56",
)

go_get(
    name = "glog",
    get = "github.com/golang/glog",
    revision = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
)

go_get(
    name = "cloud-metadata",
    get = "cloud.google.com/go/compute/metadata",
    revision = "v0.40.0",
)

go_get(
    name = "oauth2",
    get = "golang.org/x/oauth2",
    install = [
        "",
        "google",
        "internal",
        "jws",
        "jwt",
    ],
    revision = "d668ce993890a79bda886613ee587a69dd5da7a6",
    visibility = ["//third_party/go/..."],
    deps = [
        ":cloud-metadata",
        "//third_party/go:net",
    ],
)

# TODO(peterebden): Splitting up this & the other bits of genproto is a bit of
#                   a hack. Replace once we have a better way of describing Go modules.
go_get(
    name = "annotations",
    get = "google.golang.org/genproto/googleapis/api/annotations",
    revision = "3bdd9d9f5532d75d09efb230bd767d265245cfe5",
    deps = [
        ":grpc",
        ":protobuf",
    ],
)

go_get(
    name = "longrunning",
    get = "google.golang.org/genproto/googleapis/longrunning",
    revision = "3bdd9d9f5532d75d09efb230bd767d265245cfe5",
    deps = [
        ":annotations",
        ":grpc",
        ":protobuf",
    ],
)

go_get(
    name = "bytestream",
    get = "google.golang.org/genproto/googleapis/bytestream",
    revision = "3bdd9d9f5532d75d09efb230bd767d265245cfe5",
    deps = [
        ":annotations",
        ":grpc",
        ":protobuf",
    ],
)

go_get(
    name = "uuid",
    get = "github.com/google/uuid",
    revision = "v1.1.1",
)

go_get(
    name = "pborman_uuid",
    get = "github.com/pborman/uuid",
    revision = "v1.2.0",
    deps = [":uuid"],
)

go_get(
    name = "lsp",
    get = "github.com/sourcegraph/go-lsp",
    revision = "0c7d621186c1c8684f3d97868f6970b326609cc7",
)

go_get(
    name = "promptui",
    get = "github.com/manifoldco/promptui/...",
    revision = "v0.3.2",
    deps = [
        ":ansiterm",
        ":readline",
    ],
)

go_get(
    name = "readline",
    get = "github.com/chzyer/readline",
    revision = "2972be24d48e78746da79ba8e24e8b488c9880de",
)

go_get(
    name = "ansiterm",
    get = "github.com/juju/ansiterm/...",
    revision = "720a0952cc2ac777afc295d9861263e2a4cf96a1",
    deps = [
        ":colorable",
        ":go-isatty",
        ":vtclean",
    ],
)

go_get(
    name = "vtclean",
    get = "github.com/lunixbochs/vtclean",
    revision = "v1.0.0",
)

go_get(
    name = "colorable",
    get = "github.com/mattn/go-colorable",
    revision = "v0.1.2",
    deps = [
        ":go-isatty",
    ],
)

go_get(
    name = "sri",
    get = "github.com/peterebden/go-sri",
    revision = "v1.0.0",
    test_only = True,
)

go_get(
    name = "eager_errgroup",
    get = "github.com/peterebden/errgroup",
    revision = "v1.1.0",
)
