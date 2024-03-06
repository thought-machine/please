subinclude("///go//build_defs:go")

package(default_visibility = ["PUBLIC"])

go_toolchain(
    name = "toolchain",
    hashes = [
        "ebca81df938d2d1047cc992be6c6c759543cf309d401b86af38a6aed3d4090f4",  # darwin-amd64
        "bf8e388b09134164717cd52d3285a4ab3b68691b80515212da0e9f56f518fb1e",  # darwin-arm64
        "50f421c7f217083ac94aab1e09400cb9c2fea7d337679ec11f1638a11460da30",  # freebsd-amd64
        "f6c8a87aa03b92c4b0bf3d558e28ea03006eb29db78917daec5cfb6ec1046265",  # linux-amd64
        "6a63fef0e050146f275bf02a0896badfe77c11b6f05499bb647e7bd613a45a10",  # linux-arm64
    ],
    install_std = True,
    version = "1.22.0",
)

go_stdlib(
    name = "std",
)

go_repo(
    module = "github.com/please-build/buildtools",
    version = "v0.0.0-20240110130546-746673d48318",
)

go_repo(
    module = "gopkg.in/op/go-logging.v1",
    version = "v1.0.0-20160211212156-b2cb9fa56473",
)

go_repo(
    module = "golang.org/x/text",
    version = "v0.14.0",
)

go_repo(
    module = "golang.org/x/tools",
    version = "v0.18.0",
)

go_repo(
    module = "github.com/yuin/goldmark",
    version = "v1.2.1",
)

go_repo(
    module = "github.com/x1ddos/csslex",
    version = "v0.0.0-20160125172232-7894d8ab8bfe",
)

go_repo(
    module = "github.com/matttproud/golang_protobuf_extensions",
    requirements = ["github.com/golang/protobuf"],
    version = "v1.0.1",
)

go_repo(
    module = "github.com/shirou/gopsutil/v3",
    version = "v3.21.8",
)

go_repo(
    module = "github.com/chzyer/readline",
    version = "v0.0.0-20180603132655-2972be24d48e",
)

go_repo(
    module = "github.com/coreos/go-semver",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/Masterminds/semver/v3",
    version = "v3.1.1",
)

go_repo(
    module = "github.com/stretchr/testify",
    version = "v1.8.1",
)

go_repo(
    module = "github.com/peterebden/go-cli-init/v5",
    version = "v5.2.0",
)

go_repo(
    module = "gopkg.in/yaml.v3",
    version = "v3.0.1",
)

go_repo(
    module = "github.com/klauspost/compress",
    version = "v1.15.11",
)

go_repo(
    module = "github.com/jellydator/ttlcache/v2",
    version = "v2.11.1",
)

go_repo(
    module = "github.com/please-build/gcfg",
    version = "v1.6.1-0.20220308170049-10ea9d657afb",
)

go_repo(
    module = "github.com/ProtonMail/go-crypto",
    version = "v0.0.0-20210920135941-2c5829bbf927",
)

go_repo(
    module = "golang.org/x/net",
    version = "v0.21.0",
)

go_repo(
    module = "github.com/prometheus/client_model",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/peterebden/go-sri",
    version = "v1.1.1",
)

go_repo(
    module = "github.com/zeebo/blake3",
    version = "v0.2.3",
)

go_repo(
    module = "github.com/thought-machine/go-flags",
    version = "v1.6.2",
)

go_repo(
    module = "github.com/theupdateframework/go-tuf",
    version = "v0.5.2-0.20220930112810-3890c1e7ace4",
)

go_repo(
    module = "golang.org/x/oauth2",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/google/shlex",
    version = "v0.0.0-20181106134648-c34317bd91bf",
)

go_repo(
    module = "github.com/mostynb/zstdpool-syncpool",
    version = "v0.0.7",
)

go_repo(
    module = "google.golang.org/genproto",
    version = "v0.0.0-20230410155749-daa745c078e1",
)

go_repo(
    module = "github.com/djherbis/atime",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.7.2",
)

go_repo(
    module = "github.com/texttheater/golang-levenshtein",
    version = "v1.0.1",
)

go_repo(
    module = "github.com/golang/snappy",
    version = "v0.0.3",
)

go_repo(
    module = "github.com/tklauser/go-sysconf",
    version = "v0.3.9",
)

go_repo(
    module = "github.com/ulikunitz/xz",
    version = "v0.5.10",
)

go_repo(
    module = "github.com/mattn/go-isatty",
    version = "v0.0.16",
)

go_repo(
    module = "github.com/google/go-containerregistry",
    version = "v0.12.1",
)

go_repo(
    module = "github.com/tklauser/numcpus",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/prometheus/client_golang",
    version = "v1.13.0",
)

go_repo(
    module = "github.com/lunixbochs/vtclean",
    version = "v1.0.0",
)

go_repo(
    module = "golang.org/x/term",
    version = "v0.17.0",
)

go_repo(
    module = "github.com/google/go-cmp",
    version = "v0.6.0",
)

go_repo(
    module = "github.com/hashicorp/go-multierror",
    version = "v1.1.1",
)

go_repo(
    module = "github.com/grpc-ecosystem/go-grpc-middleware",
    version = "v1.3.0",
)

go_repo(
    module = "github.com/pborman/uuid",
    version = "v1.2.0",
)

go_repo(
    module = "github.com/sigstore/sigstore",
    version = "v1.5.0",
)

go_repo(
    module = "gopkg.in/warnings.v0",
    version = "v0.1.2",
)

go_repo(
    module = "github.com/pkg/errors",
    version = "v0.9.1",
)

go_repo(
    module = "github.com/googleapis/gax-go/v2",
    version = "v2.7.1",
)

go_repo(
    module = "github.com/golang/groupcache",
    version = "v0.0.0-20210331224755-41bb18bfe9da",
)

go_repo(
    module = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/golang/protobuf",
    version = "v1.5.4",
)

go_repo(
    module = "golang.org/x/sync",
    version = "v0.6.0",
)

go_repo(
    module = "github.com/dustin/go-humanize",
    version = "v1.0.0",
)

go_repo(
    module = "golang.org/x/mod",
    version = "v0.5.0",
)

go_repo(
    module = "github.com/googleapis/enterprise-certificate-proxy",
    version = "v0.2.3",
)

go_repo(
    module = "github.com/peterebden/ar",
    version = "v0.0.0-20181115090543-a0ae3a11a518",
)

go_repo(
    module = "cloud.google.com/go",
    version = "v0.65.0",
)

go_repo(
    module = "cloud.google.com/go/compute/metadata",
    version = "v0.2.3",
)

go_repo(
    module = "github.com/mattn/go-colorable",
    version = "v0.1.13",
)

go_repo(
    module = "github.com/peterebden/go-deferred-regex",
    version = "v1.0.0",
)

go_repo(
    module = "github.com/hashicorp/errwrap",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/karrick/godirwalk",
    version = "v1.16.1",
)

go_repo(
    module = "github.com/beorn7/perks",
    version = "v1.0.1",
)

go_repo(
    module = "github.com/juju/ansiterm",
    requirements = [
        "github.com/lunixbochs/vtclean",
        "github.com/mattn/go-colorable",
        "github.com/mattn/go-isatty",
    ],
    version = "v0.0.0-20210706145210-9283cdf370b5",
)

go_repo(
    module = "github.com/davecgh/go-spew",
    version = "v1.1.1",
)

go_repo(
    module = "golang.org/x/sys",
    version = "v0.17.0",
)

go_repo(
    module = "github.com/shurcooL/sanitized_anchor_name",
    version = "v1.0.0",
)

go_repo(
    module = "github.com/peterebden/tools",
    version = "v0.0.0-20190805132753-b2a0db951d2a",
)

go_repo(
    module = "github.com/opencontainers/go-digest",
    version = "v1.0.0",
)

go_repo(
    module = "google.golang.org/grpc",
    version = "v1.56.3",
)

go_repo(
    module = "github.com/sourcegraph/jsonrpc2",
    version = "v0.1.0",
)

go_repo(
    module = "golang.org/x/xerrors",
    version = "5ec99f83aff198f5fbd629d6c8d8eb38a04218ca",
)

go_repo(
    module = "github.com/cespare/xxhash/v2",
    version = "v2.2.0",
)

go_repo(
    module = "github.com/googlecodelabs/tools/claat",
    version = "v0.0.0-20210914205149-d1177395e3b8",
)

go_repo(
    module = "github.com/klauspost/cpuid/v2",
    version = "v2.0.12",
)

go_repo(
    module = "github.com/gogo/protobuf",
    version = "v1.3.2",
)

go_repo(
    module = "github.com/stretchr/objx",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/bazelbuild/remote-apis-sdks",
    version = "v0.0.0-20221114180157-e62cf9b8696a",
)

go_repo(
    module = "github.com/alessio/shellescape",
    version = "v1.4.1",
)

go_repo(
    module = "golang.org/x/crypto",
    version = "v0.19.0",
)

go_repo(
    module = "google.golang.org/protobuf",
    version = "v1.33.0",
)

go_repo(
    module = "github.com/golang/glog",
    patch = ["glog_disable.patch"],
    version = "v1.2.0",
)

go_repo(
    module = "github.com/sourcegraph/go-lsp",
    version = "v0.0.0-20200429204803-219e11d77f5d",
)

go_repo(
    module = "github.com/pkg/xattr",
    version = "v0.4.4",
)

go_repo(
    module = "cloud.google.com/go/iam",
    version = "v0.13.0",
)

go_repo(
    module = "go.uber.org/automaxprocs",
    version = "v1.4.0",
)

go_repo(
    module = "github.com/pmezard/go-difflib",
    version = "v1.0.0",
)

go_repo(
    module = "gopkg.in/square/go-jose.v2",
    requirements = ["golang.org/x/crypto"],
    version = "v2.6.0",
)

go_repo(
    module = "github.com/prometheus/common",
    version = "v0.37.0",
)

go_repo(
    module = "github.com/titanous/rocacheck",
    version = "v0.0.0-20171023193734-afe73141d399",
)

go_repo(
    module = "google.golang.org/api",
    version = "v0.114.0",
)

go_repo(
    module = "github.com/hashicorp/go-cleanhttp",
    version = "v0.5.2",
)

go_repo(
    module = "go.opencensus.io",
    version = "v0.24.0",
)

go_repo(
    module = "cloud.google.com/go/longrunning",
    version = "v0.4.1",
)

go_repo(
    module = "github.com/prometheus/procfs",
    version = "v0.8.0",
)

go_repo(
    module = "github.com/manifoldco/promptui",
    version = "v0.8.0",
)

go_repo(
    module = "sourcegraph.com/sqs/pbtypes",
    version = "v0.0.0-20180604144634-d3ebe8f20ae4",
)

go_repo(
    module = "github.com/google/uuid",
    version = "v1.3.0",
)

go_repo(
    module = "github.com/letsencrypt/boulder",
    version = "v0.0.0-20221109233200-85aa52084eaf",
)

go_repo(
    module = "github.com/fsnotify/fsnotify",
    version = "v1.5.1",
)

go_repo(
    module = "github.com/bazelbuild/remote-apis",
    version = "v0.0.0-20210718193713-0ecef08215cf",
)

go_repo(
    module = "go.starlark.net",
    version = "v0.0.0-20210901212718-87f333178d59",
)

go_repo(
    module = "cloud.google.com/go/kms",
    version = "v1.10.1",
)

go_repo(
    module = "github.com/sourcegraph/go-diff",
    version = "v0.6.1",
)

go_repo(
    module = "golang.org/x/exp",
    version = "v0.0.0-20240213143201-ec583247a57a",
)

go_repo(
    module = "github.com/jstemmer/go-junit-report/v2",
    version = "v2.1.0",
)

go_repo(
    module = "cloud.google.com/go/compute",
    version = "v1.19.1",
    licences = ["Apache-2.0"],
)

go_repo(
    module = "github.com/StackExchange/wmi",
    version = "v1.2.1",
)

go_repo(
    module = "github.com/go-ole/go-ole",
    version = "v1.2.5",
    licences = ["MIT"],
)

go_repo(
    module = "google.golang.org/appengine",
    version = "v1.6.7",
    licences = ["Apache-2.0"],
)
