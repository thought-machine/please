subinclude("///go//build_defs:go")

package(default_visibility = ["PUBLIC"])

go_toolchain(
    name = "toolchain",
    hashes = [
        "addbfce2056744962e2d7436313ab93486660cf7a2e066d171b9d6f2da7c7abe",  # go1.24.1.darwin-amd64.tar.gz
        "295581b5619acc92f5106e5bcb05c51869337eb19742fdfa6c8346c18e78ff88",  # go1.24.1.darwin-arm64.tar.gz
        "47d7de8bb64d5c3ee7b6723aa62d5ecb11e3568ef2249bbe1d4bbd432d37c00c",  # go1.24.1.freebsd-amd64.tar.gz
        "cb2396bae64183cdccf81a9a6df0aea3bce9511fc21469fb89a0c00470088073",  # go1.24.1.linux-amd64.tar.gz
        "8df5750ffc0281017fb6070fba450f5d22b600a02081dceef47966ffaf36a3af",  # go1.24.1.linux-arm64.tar.gz
    ],
    install_std = False,
    version = "1.24.1",
)

go_stdlib(
    name = "std",
    tags = [
        "netgo",
        "osusergo",
    ],
)

go_repo(
    module = "github.com/please-build/buildtools",
    version = "v0.0.0-20240111140234-77ffe55926d9",
)

go_repo(
    module = "gopkg.in/op/go-logging.v1",
    version = "v1.0.0-20160211212156-b2cb9fa56473",
)

go_repo(
    module = "golang.org/x/text",
    version = "v0.23.0",
)

go_repo(
    module = "golang.org/x/tools",
    version = "v0.21.1-0.20240508182429-e35e4ccd0d2d",
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
    version = "v1.0.4",
)

go_repo(
    module = "github.com/shirou/gopsutil/v3",
    version = "v3.24.2",
)

go_repo(
    module = "github.com/chzyer/readline",
    version = "v1.5.1",
)

go_repo(
    module = "github.com/coreos/go-semver",
    version = "v0.3.1",
)

go_repo(
    module = "github.com/Masterminds/semver/v3",
    version = "v3.2.1",
)

go_repo(
    module = "github.com/stretchr/testify",
    version = "v1.9.0",
)

go_repo(
    module = "github.com/peterebden/go-cli-init/v5",
    version = "v5.2.1",
)

go_repo(
    module = "gopkg.in/yaml.v3",
    version = "v3.0.1",
)

go_repo(
    module = "github.com/klauspost/compress",
    version = "v1.17.7",
)

go_repo(
    module = "github.com/jellydator/ttlcache/v2",
    version = "v2.11.1",
)

go_repo(
    module = "github.com/please-build/gcfg",
    version = "v1.7.0",
)

go_repo(
    module = "github.com/ProtonMail/go-crypto",
    version = "v0.0.0-20210920135941-2c5829bbf927",
)

go_repo(
    module = "golang.org/x/net",
    version = "v0.38.0",
)

go_repo(
    module = "github.com/prometheus/client_model",
    version = "v0.6.0",
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
    version = "v1.6.3",
)

go_repo(
    module = "github.com/theupdateframework/go-tuf",
    version = "v0.7.0",
)

go_repo(
    module = "golang.org/x/oauth2",
    version = "v0.18.0",
)

go_repo(
    module = "github.com/google/shlex",
    version = "v0.0.0-20191202100458-e7afc7fbc510",
)

go_repo(
    module = "github.com/mostynb/zstdpool-syncpool",
    version = "v0.0.13",
)

go_repo(
    module = "google.golang.org/genproto",
    version = "v0.0.0-20240304212257-790db918fca8",
)

go_repo(
    module = "github.com/djherbis/atime",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.7.7",
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
    version = "v0.3.13",
)

go_repo(
    module = "github.com/ulikunitz/xz",
    version = "v0.5.11",
)

go_repo(
    module = "github.com/google/go-containerregistry",
    version = "v0.19.0",
)

go_repo(
    module = "github.com/tklauser/numcpus",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/prometheus/client_golang",
    version = "v1.19.0",
)

go_repo(
    module = "golang.org/x/term",
    version = "v0.30.0",
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
    version = "v1.4.0",
)

go_repo(
    module = "github.com/pborman/uuid",
    version = "v1.2.1",
)

go_repo(
    module = "github.com/sigstore/sigstore",
    version = "v1.8.2",
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
    version = "v2.12.2",
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
    version = "v0.12.0",
)

go_repo(
    module = "github.com/dustin/go-humanize",
    version = "v1.0.1",
)

go_repo(
    module = "golang.org/x/mod",
    version = "v0.5.0",
)

go_repo(
    module = "github.com/googleapis/enterprise-certificate-proxy",
    version = "v0.3.2",
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
    module = "github.com/peterebden/go-deferred-regex",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/hashicorp/errwrap",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/karrick/godirwalk",
    version = "v1.17.0",
)

go_repo(
    module = "github.com/beorn7/perks",
    version = "v1.0.1",
)

go_repo(
    module = "github.com/davecgh/go-spew",
    version = "v1.1.1",
)

go_repo(
    module = "golang.org/x/sys",
    version = "v0.31.0",
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
    version = "v1.62.1",
)

go_repo(
    module = "github.com/sourcegraph/jsonrpc2",
    version = "v0.2.0",
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
    version = "v2.2.7",
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
    version = "v1.4.2",
)

go_repo(
    module = "golang.org/x/crypto",
    version = "v0.36.0",
)

go_repo(
    module = "google.golang.org/protobuf",
    version = "v1.33.0",
)

go_repo(
    module = "github.com/golang/glog",
    patch = ["glog_disable.patch"],
    version = "v1.2.4",
)

go_repo(
    module = "github.com/sourcegraph/go-lsp",
    version = "v0.0.0-20240223163137-f80c5dd31dfd",
)

go_repo(
    module = "github.com/pkg/xattr",
    version = "v0.4.9",
)

go_repo(
    module = "cloud.google.com/go/iam",
    version = "v1.1.6",
)

go_repo(
    module = "go.uber.org/automaxprocs",
    version = "v1.5.3",
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
    version = "v0.50.0",
)

go_repo(
    module = "github.com/titanous/rocacheck",
    version = "v0.0.0-20171023193734-afe73141d399",
)

go_repo(
    module = "google.golang.org/api",
    version = "v0.168.0",
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
    version = "v0.5.5",
)

go_repo(
    module = "github.com/prometheus/procfs",
    version = "v0.13.0",
)

go_repo(
    module = "github.com/manifoldco/promptui",
    version = "v0.9.0",
)

go_repo(
    module = "sourcegraph.com/sqs/pbtypes",
    version = "v0.0.0-20180604144634-d3ebe8f20ae4",
)

go_repo(
    module = "github.com/google/uuid",
    version = "v1.6.0",
)

go_repo(
    module = "github.com/letsencrypt/boulder",
    version = "v0.0.0-20240306190618-9b05c38eb38a",
)

go_repo(
    module = "github.com/fsnotify/fsnotify",
    version = "v1.7.0",
)

go_repo(
    module = "github.com/bazelbuild/remote-apis",
    version = "v0.0.0-20240409135018-1f36c310b28d",
)

go_repo(
    module = "go.starlark.net",
    version = "v0.0.0-20210901212718-87f333178d59",
)

go_repo(
    module = "cloud.google.com/go/kms",
    version = "v1.15.7",
)

go_repo(
    module = "github.com/sourcegraph/go-diff",
    version = "v0.7.0",
)

go_repo(
    module = "golang.org/x/exp",
    version = "v0.0.0-20240222234643-814bf88cf225",
)

go_repo(
    module = "github.com/jstemmer/go-junit-report/v2",
    version = "v2.1.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "cloud.google.com/go/compute",
    version = "v1.25.0",
)

go_repo(
    module = "github.com/StackExchange/wmi",
    version = "v1.2.1",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/go-ole/go-ole",
    version = "v1.3.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/appengine",
    version = "v1.6.8",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/genproto/googleapis/rpc",
    version = "v0.0.0-20240304212257-790db918fca8",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/secure-systems-lab/go-securesystemslib",
    version = "v0.8.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "gopkg.in/go-jose/go-jose.v2",
    version = "v2.6.3",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/genproto/googleapis/api",
    version = "v0.0.0-20240304212257-790db918fca8",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "google.golang.org/genproto/googleapis/bytestream",
    version = "v0.0.0-20240304212257-790db918fca8",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/sigstore/sigstore/pkg/signature/kms/gcp",
    version = "v1.8.2",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/felixge/httpsnoop",
    version = "v1.0.4",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/go-logr/logr",
    version = "v1.4.1",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/go-logr/stdr",
    version = "v1.2.2",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/google/s2a-go",
    version = "v0.1.7",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/jellydator/ttlcache/v3",
    version = "v3.2.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
    version = "v0.49.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
    version = "v0.49.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opentelemetry.io/otel",
    version = "v1.24.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opentelemetry.io/otel/metric",
    version = "v1.24.0",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "go.opentelemetry.io/otel/trace",
    version = "v1.24.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "golang.org/x/time",
    version = "v0.5.0",
)

go_repo(
    licences = ["BSD-3-Clause"],
    module = "github.com/lufia/plan9stats",
    version = "v0.0.0-20240226150601-1dcf7310316a",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/power-devops/perfstat",
    version = "v0.0.0-20240221224432-82ca36839d55",
)

go_repo(
    licences = ["MPL-2.0"],
    module = "github.com/shoenig/go-m1cpu",
    version = "v0.1.6",
)

go_repo(
    licences = ["MIT"],
    module = "github.com/yusufpapurcu/wmi",
    version = "v1.2.4",
)

go_repo(
    licences = ["Apache-2.0"],
    module = "github.com/grpc-ecosystem/go-grpc-prometheus",
    version = "v1.2.0",
)
