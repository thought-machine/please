subinclude("///go//build_defs:go")

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

go_mod_download(
    name = "build_tools_dl",
    module = "github.com/peterebden/buildtools",
    version = "1bf812c34357cd58ee72892491ddb20b3cab4fce",
)

go_repo(
    download = ":build_tools_dl",
    module = "github.com/bazelbuild/buildtools",
)

go_repo(
    module = "gopkg.in/op/go-logging.v1",
    version = "v1.0.0-20160211212156-b2cb9fa56473",
)

go_repo(
    module = "golang.org/x/text",
    version = "v0.9.0",
)

go_repo(
    module = "golang.org/x/tools",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/yuin/goldmark",
    version = "v1.4.13",
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
    version = "v3.23.4",
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
    version = "v1.8.4",
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
    version = "v1.15.9",
)

go_repo(
    module = "github.com/jellydator/ttlcache/v2",
    version = "v2.11.1",
)

go_repo(
    module = "github.com/please-build/gcfg",
    version = "9b4d14cddd6148920e11c8bb7ee19369473b1019",
)

go_repo(
    module = "github.com/ProtonMail/go-crypto",
    version = "3900d675f39ba9686203d137886d74fad620ae87",
)

go_repo(
    module = "golang.org/x/net",
    version = "v0.10.0",
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
    version = "v0.5.1",
)

go_repo(
    module = "golang.org/x/oauth2",
    version = "v0.6.0",
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
    module = "github.com/peterebden/buildtools",
    version = "f7a36c689cc9e038956d51005332c67ade7aa9c6",
)

go_repo(
    module = "google.golang.org/genproto",
    version = "v0.0.0-20230530153820-e85fd2cbaebc",
)

go_repo(
    module = "github.com/djherbis/atime",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/hashicorp/go-retryablehttp",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/texttheater/golang-levenshtein",
    version = "v1.0.1",
)

go_repo(
    module = "github.com/golang/snappy",
    version = "v0.0.4",
)

go_repo(
    module = "github.com/tklauser/go-sysconf",
    version = "v0.3.11",
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
    version = "v0.12.0",
)

go_repo(
    module = "github.com/tklauser/numcpus",
    version = "v0.6.0",
)

go_repo(
    module = "github.com/prometheus/client_golang",
    version = "v1.11.0",
)

go_repo(
    module = "github.com/lunixbochs/vtclean",
    version = "v1.0.0",
)

go_repo(
    module = "golang.org/x/term",
    version = "v0.8.0",
)

go_repo(
    module = "github.com/google/go-cmp",
    version = "v0.5.9",
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
    version = "v0.0.0-20200121045136-8c9f03a8e57e",
)

go_repo(
    module = "google.golang.org/grpc/cmd/protoc-gen-go-grpc",
    version = "v1.1.0",
)

go_repo(
    module = "github.com/golang/protobuf",
    version = "v1.5.3",
)

go_repo(
    module = "golang.org/x/sync",
    version = "v0.1.0",
)

go_repo(
    module = "github.com/dustin/go-humanize",
    version = "v1.0.0",
)

go_repo(
    module = "golang.org/x/mod",
    version = "v0.9.0",
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
    version = "v0.110.0",
)

go_repo(
    module = "cloud.google.com/go/compute/metadata",
    version = "v0.2.3",
)

go_repo(
    module = "github.com/mattn/go-colorable",
    version = "v0.1.8",
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
    version = "720a0952cc2ac777afc295d9861263e2a4cf96a1",
)

go_repo(
    module = "github.com/davecgh/go-spew",
    version = "v1.1.1",
)

go_repo(
    module = "golang.org/x/sys",
    version = "v0.8.0",
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
    version = "v1.55.0",
)

go_repo(
    module = "github.com/sourcegraph/jsonrpc2",
    version = "v0.1.0",
)

go_repo(
    module = "golang.org/x/xerrors",
    version = "v0.0.0-20220907171357-04be3eba64a2",
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
    version = "v2.2.1",
)

go_repo(
    module = "github.com/gogo/protobuf",
    version = "v1.3.2",
)

go_repo(
    module = "github.com/stretchr/objx",
    version = "v0.5.0",
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
    version = "v0.5.0",
)

go_repo(
    module = "google.golang.org/protobuf",
    version = "v1.30.0",
)

go_repo(
    module = "github.com/golang/glog",
    patch = ["glog_disable.patch"],
    version = "v0.0.0-20160126235308-23def4e6c14b",
)

go_repo(
    module = "github.com/sourcegraph/go-lsp",
    version = "v0.0.0-20200429204803-219e11d77f5d",
)

go_repo(
    module = "github.com/pkg/xattr",
    version = "v0.4.3",
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
    version = "v0.30.0",
)

go_repo(
    module = "github.com/titanous/rocacheck",
    version = "afe73141d399b0c79c3d6412d4bdcf4c672c496a",
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
    version = "v0.6.0",
)

go_repo(
    module = "github.com/manifoldco/promptui",
    version = "v0.7.0",
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
    version = "cfa524a7a139bd4cf21c4c09ab47b994a96fd11f",
)

go_repo(
    module = "github.com/fsnotify/fsnotify",
    version = "v1.5.1",
)

go_repo(
    module = "github.com/bazelbuild/remote-apis",
    version = "3e816456ee28f01ab2e0abf72306c1f340c7b229",
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
    version = "v0.0.0-20220827204233-334a2380cb91",
)

go_repo(
    module = "github.com/BurntSushi/toml",
    version = "v0.3.1",
)

go_repo(
    module = "modernc.org/httpfs",
    version = "v1.0.6",
)

go_repo(
    module = "cloud.google.com/go/aiplatform",
    version = "v1.37.0",
)

go_repo(
    module = "github.com/JohnCGriffin/overflow",
    version = "v0.0.0-20211019200055-46fa312c352c",
)

go_repo(
    module = "github.com/mattn/go-sqlite3",
    version = "v1.14.14",
)

go_repo(
    module = "github.com/rogpeppe/fastuuid",
    version = "v1.2.0",
)

go_repo(
    module = "github.com/spf13/afero",
    version = "v1.9.2",
)

go_repo(
    module = "cloud.google.com/go/iot",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/shell",
    version = "v1.6.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/sdk/metric",
    version = "v0.39.0",
)

go_repo(
    module = "gopkg.in/check.v1",
    version = "v1.0.0-20201130134442-10cb98267c6c",
)

go_repo(
    module = "cloud.google.com/go/binaryauthorization",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/pkg/sftp",
    version = "v1.13.1",
)

go_repo(
    module = "cloud.google.com/go/billing",
    version = "v1.13.0",
)

go_repo(
    module = "cloud.google.com/go/spanner",
    version = "v1.45.0",
)

go_repo(
    module = "github.com/kr/pty",
    version = "v1.1.1",
)

go_repo(
    module = "github.com/remyoudompheng/bigfft",
    version = "v0.0.0-20200410134404-eec4a21b6bb0",
)

go_repo(
    module = "cloud.google.com/go/asset",
    version = "v1.13.0",
)

go_repo(
    module = "cloud.google.com/go/lifesciences",
    version = "v0.8.0",
)

go_repo(
    module = "cloud.google.com/go/speech",
    version = "v1.15.0",
)

go_repo(
    module = "github.com/antihax/optional",
    version = "v1.0.0",
)

go_repo(
    module = "cloud.google.com/go/storage",
    version = "v1.29.0",
)

go_repo(
    module = "cloud.google.com/go/osconfig",
    version = "v1.11.0",
)

go_repo(
    module = "cloud.google.com/go/domains",
    version = "v0.8.0",
)

go_repo(
    module = "cloud.google.com/go/gkehub",
    version = "v0.12.0",
)

go_repo(
    module = "github.com/cenkalti/backoff/v4",
    version = "v4.2.1",
)

go_repo(
    module = "cloud.google.com/go/policytroubleshooter",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/recommender",
    version = "v1.9.0",
)

go_repo(
    module = "github.com/phpdave11/gofpdi",
    version = "v1.0.13",
)

go_repo(
    module = "github.com/go-latex/latex",
    version = "v0.0.0-20210823091927-c0d11ff05a81",
)

go_repo(
    module = "modernc.org/strutil",
    version = "v1.1.3",
)

go_repo(
    module = "cloud.google.com/go/servicemanagement",
    version = "v1.6.0",
)

go_repo(
    module = "git.sr.ht/~sbinet/gg",
    version = "v0.3.1",
)

go_repo(
    module = "cloud.google.com/go/documentai",
    version = "v1.18.0",
)

go_repo(
    module = "github.com/lyft/protoc-gen-star",
    version = "v0.6.1",
)

go_repo(
    module = "github.com/honeycombio/honeycomb-opentelemetry-go",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/apikeys",
    version = "v0.5.0",
)

go_repo(
    module = "cloud.google.com/go/cloudtasks",
    version = "v1.10.0",
)

go_repo(
    module = "cloud.google.com/go/gkemulticloud",
    version = "v0.5.0",
)

go_repo(
    module = "golang.org/x/image",
    version = "v0.0.0-20220302094943-723b81ca9867",
)

go_repo(
    module = "github.com/go-fonts/liberation",
    version = "v0.2.0",
)

go_repo(
    module = "cloud.google.com/go/secretmanager",
    version = "v1.10.0",
)

go_repo(
    module = "google.golang.org/genproto/googleapis/api",
    version = "v0.0.0-20230530153820-e85fd2cbaebc",
)

go_repo(
    module = "golang.org/x/lint",
    version = "v0.0.0-20210508222113-6edffad5e616",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc",
    version = "v0.39.0",
)

go_repo(
    module = "github.com/zeebo/xxh3",
    version = "v1.0.2",
)

go_repo(
    module = "modernc.org/z",
    version = "v1.5.1",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlptrace",
    version = "v1.16.0",
)

go_repo(
    module = "github.com/kr/fs",
    version = "v0.1.0",
)

go_repo(
    module = "cloud.google.com/go/security",
    version = "v1.13.0",
)

go_repo(
    module = "lukechampine.com/uint128",
    version = "v1.2.0",
)

go_repo(
    module = "cloud.google.com/go/accessapproval",
    version = "v1.6.0",
)

go_repo(
    module = "github.com/envoyproxy/go-control-plane",
    version = "v0.11.0",
)

go_repo(
    module = "cloud.google.com/go/area120",
    version = "v0.7.1",
)

go_repo(
    module = "cloud.google.com/go/baremetalsolution",
    version = "v0.5.0",
)

go_repo(
    module = "cloud.google.com/go/datalabeling",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/functions",
    version = "v1.13.0",
)

go_repo(
    module = "cloud.google.com/go/serviceusage",
    version = "v1.5.0",
)

go_repo(
    module = "modernc.org/mathutil",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/datastore",
    version = "v1.11.0",
)

go_repo(
    module = "cloud.google.com/go/ids",
    version = "v1.3.0",
)

go_repo(
    module = "cloud.google.com/go/apigateway",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/servicecontrol",
    version = "v1.11.0",
)

go_repo(
    module = "github.com/kr/text",
    version = "v0.2.0",
)

go_repo(
    module = "github.com/shoenig/go-m1cpu",
    version = "v0.1.6",
)

go_repo(
    module = "modernc.org/opt",
    version = "v0.1.3",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/stdout/stdouttrace",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/video",
    version = "v1.15.0",
)

go_repo(
    module = "github.com/golang/freetype",
    version = "v0.0.0-20170609003504-e2365dfdc4a0",
)

go_repo(
    module = "github.com/ajstarks/deck/generate",
    version = "v0.0.0-20210309230005-c3f852c02e19",
)

go_repo(
    module = "cloud.google.com/go/automl",
    version = "v1.12.0",
)

go_repo(
    module = "cloud.google.com/go/deploy",
    version = "v1.8.0",
)

go_repo(
    module = "cloud.google.com/go/apigeeconnect",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/grafeas",
    version = "v0.2.0",
)

go_repo(
    module = "cloud.google.com/go/gaming",
    version = "v1.9.0",
)

go_repo(
    module = "gioui.org",
    version = "v0.0.0-20210308172011-57750fc8a0a6",
)

go_repo(
    module = "github.com/google/martian/v3",
    version = "v3.3.2",
)

go_repo(
    module = "cloud.google.com/go/clouddms",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/dataproc",
    version = "v1.12.0",
)

go_repo(
    module = "cloud.google.com/go/dlp",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/eventarc",
    version = "v1.11.0",
)

go_repo(
    module = "gopkg.in/errgo.v2",
    version = "v2.1.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/orgpolicy",
    version = "v1.10.0",
)

go_repo(
    module = "github.com/goccy/go-json",
    version = "v0.9.11",
)

go_repo(
    module = "cloud.google.com/go/servicedirectory",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/talent",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/census-instrumentation/opencensus-proto",
    version = "v0.4.1",
)

go_repo(
    module = "cloud.google.com/go/firestore",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/tpu",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/kisielk/gotool",
    version = "v1.0.0",
)

go_repo(
    module = "cloud.google.com/go/vision",
    version = "v1.2.0",
)

go_repo(
    module = "cloud.google.com/go/vpcaccess",
    version = "v1.6.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/container",
    version = "v1.15.0",
)

go_repo(
    module = "cloud.google.com/go/iap",
    version = "v1.7.1",
)

go_repo(
    module = "cloud.google.com/go/appengine",
    version = "v1.7.1",
)

go_repo(
    module = "go.opentelemetry.io/contrib/instrumentation/host",
    version = "v0.42.0",
)

go_repo(
    module = "github.com/go-fonts/latin-modern",
    version = "v0.2.0",
)

go_repo(
    module = "go.opentelemetry.io/proto/otlp",
    version = "v0.19.0",
)

go_repo(
    module = "github.com/go-fonts/dejavu",
    version = "v0.1.0",
)

go_repo(
    module = "github.com/ruudk/golang-pdf417",
    version = "v0.0.0-20201230142125-a7e3863a1245",
)

go_repo(
    module = "github.com/envoyproxy/protoc-gen-validate",
    version = "v0.10.0",
)

go_repo(
    module = "cloud.google.com/go/gsuiteaddons",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/redis",
    version = "v1.11.0",
)

go_repo(
    module = "github.com/grpc-ecosystem/grpc-gateway/v2",
    version = "v2.15.2",
)

go_repo(
    module = "github.com/minio/asm2plan9s",
    version = "v0.0.0-20200509001527-cdd76441f9d8",
)

go_repo(
    module = "gonum.org/v1/plot",
    version = "v0.10.1",
)

go_repo(
    module = "cloud.google.com/go/cloudbuild",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/networkconnectivity",
    version = "v1.11.0",
)

go_repo(
    module = "cloud.google.com/go/phishingprotection",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/shoenig/test",
    version = "v0.6.4",
)

go_repo(
    module = "modernc.org/libc",
    version = "v1.17.1",
)

go_repo(
    module = "github.com/lyft/protoc-gen-star/v2",
    version = "v2.0.1",
)

go_repo(
    module = "go.opentelemetry.io/otel",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/apigeeregistry",
    version = "v0.6.0",
)

go_repo(
    module = "modernc.org/ccorpus",
    version = "v1.11.6",
)

go_repo(
    module = "cloud.google.com/go/translate",
    version = "v1.7.0",
)

go_repo(
    module = "honnef.co/go/tools",
    version = "v0.1.3",
)

go_repo(
    module = "cloud.google.com/go/memcache",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/dialogflow",
    version = "v1.32.0",
)

go_repo(
    module = "cloud.google.com/go/datafusion",
    version = "v1.6.0",
)

go_repo(
    module = "github.com/ajstarks/svgo",
    version = "v0.0.0-20211024235047-1546f124cd8b",
)

go_repo(
    module = "github.com/creack/pty",
    version = "v1.1.9",
)

go_repo(
    module = "cloud.google.com/go/edgecontainer",
    version = "v1.0.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/metric",
    version = "v1.16.0",
)

go_repo(
    module = "github.com/apache/thrift",
    version = "v0.16.0",
)

go_repo(
    module = "modernc.org/tcl",
    version = "v1.13.1",
)

go_repo(
    module = "cloud.google.com/go/vmmigration",
    version = "v1.6.0",
)

go_repo(
    module = "github.com/power-devops/perfstat",
    version = "v0.0.0-20221212215047-62379fc7944b",
)

go_repo(
    module = "modernc.org/token",
    version = "v1.0.0",
)

go_repo(
    module = "cloud.google.com/go/mediatranslation",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/resourcesettings",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/webrisk",
    version = "v1.8.0",
)

go_repo(
    module = "github.com/andybalholm/brotli",
    version = "v1.0.4",
)

go_repo(
    module = "modernc.org/cc/v3",
    version = "v3.36.3",
)

go_repo(
    module = "github.com/apache/arrow/go/v11",
    version = "v11.0.0",
)

go_repo(
    module = "github.com/googleapis/google-cloud-go-testing",
    version = "v0.0.0-20200911160855-bcd43fbb19e8",
)

go_repo(
    module = "golang.org/x/time",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/kballard/go-shellquote",
    version = "v0.0.0-20180428030007-95032a82bc51",
)

go_repo(
    module = "cloud.google.com/go/compute",
    version = "v1.19.0",
)

go_repo(
    module = "cloud.google.com/go/texttospeech",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/batch",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/managedidentities",
    version = "v1.5.0",
)

go_repo(
    module = "cloud.google.com/go/beyondcorp",
    version = "v0.5.0",
)

go_repo(
    module = "cloud.google.com/go/essentialcontacts",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/boombuler/barcode",
    version = "v1.0.1",
)

go_repo(
    module = "cloud.google.com/go/assuredworkloads",
    version = "v1.10.0",
)

go_repo(
    module = "go.uber.org/goleak",
    version = "v1.2.1",
)

go_repo(
    module = "cloud.google.com/go/dataplex",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/accesscontextmanager",
    version = "v1.7.0",
)

go_repo(
    module = "cloud.google.com/go/certificatemanager",
    version = "v1.6.0",
)

go_repo(
    module = "github.com/lufia/plan9stats",
    version = "v0.0.0-20230326075908-cb1d2100619a",
)

go_repo(
    module = "github.com/pkg/diff",
    version = "v0.0.0-20210226163009-20ebb0f2a09e",
)

go_repo(
    module = "github.com/go-logr/logr",
    version = "v1.2.4",
)

go_repo(
    module = "github.com/go-pdf/fpdf",
    version = "v0.6.0",
)

go_repo(
    module = "cloud.google.com/go/recommendationengine",
    version = "v0.7.0",
)

go_repo(
    module = "go.opentelemetry.io/contrib/propagators/ot",
    version = "v1.17.0",
)

go_repo(
    module = "github.com/golang/mock",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/cncf/udpa/go",
    version = "v0.0.0-20220112060539-c52dc94e7fbe",
)

go_repo(
    module = "cloud.google.com/go/datacatalog",
    version = "v1.13.0",
)

go_repo(
    module = "cloud.google.com/go/filestore",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/notebooks",
    version = "v1.8.0",
)

go_repo(
    module = "cloud.google.com/go/securitycenter",
    version = "v1.19.0",
)

go_repo(
    module = "cloud.google.com/go/analytics",
    version = "v0.19.0",
)

go_repo(
    module = "cloud.google.com/go/run",
    version = "v0.9.0",
)

go_repo(
    module = "github.com/go-logr/stdr",
    version = "v1.2.2",
)

go_repo(
    module = "cloud.google.com/go/resourcemanager",
    version = "v1.7.0",
)

go_repo(
    module = "cloud.google.com/go/websecurityscanner",
    version = "v1.5.0",
)

go_repo(
    module = "github.com/docopt/docopt-go",
    version = "v0.0.0-20180111231733-ee0de3bc6815",
)

go_repo(
    module = "modernc.org/sqlite",
    version = "v1.18.1",
)

go_repo(
    module = "cloud.google.com/go/monitoring",
    version = "v1.13.0",
)

go_repo(
    module = "github.com/google/flatbuffers",
    version = "v2.0.8+incompatible",
)

go_repo(
    module = "github.com/klauspost/asmfmt",
    version = "v1.3.2",
)

go_repo(
    module = "gonum.org/v1/gonum",
    version = "v0.11.0",
)

go_repo(
    module = "github.com/fogleman/gg",
    version = "v1.3.0",
)

go_repo(
    module = "cloud.google.com/go/storagetransfer",
    version = "v1.8.0",
)

go_repo(
    module = "github.com/apache/arrow/go/v10",
    version = "v10.0.1",
)

go_repo(
    module = "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
    version = "v0.42.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/trace",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/errorreporting",
    version = "v0.3.0",
)

go_repo(
    module = "cloud.google.com/go/networkmanagement",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/pubsub",
    version = "v1.30.0",
)

go_repo(
    module = "cloud.google.com/go/artifactregistry",
    version = "v1.13.0",
)

go_repo(
    module = "cloud.google.com/go/gkeconnect",
    version = "v0.7.0",
)

go_repo(
    module = "github.com/minio/c2goasm",
    version = "v0.0.0-20190812172519-36a3d3bbc4f3",
)

go_repo(
    module = "go.opentelemetry.io/contrib/propagators/b3",
    version = "v1.17.0",
)

go_repo(
    module = "cloud.google.com/go/videointelligence",
    version = "v1.10.0",
)

go_repo(
    module = "github.com/sethvargo/go-envconfig",
    version = "v0.9.0",
)

go_repo(
    module = "rsc.io/pdf",
    version = "v0.1.1",
)

go_repo(
    module = "cloud.google.com/go/datastream",
    version = "v1.7.0",
)

go_repo(
    module = "cloud.google.com/go/oslogin",
    version = "v1.9.0",
)

go_repo(
    module = "github.com/kr/pretty",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/pierrec/lz4/v4",
    version = "v4.1.15",
)

go_repo(
    module = "github.com/zeebo/assert",
    version = "v1.3.0",
)

go_repo(
    module = "cloud.google.com/go/gkebackup",
    version = "v0.4.0",
)

go_repo(
    module = "cloud.google.com/go/language",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/privatecatalog",
    version = "v0.8.0",
)

go_repo(
    module = "cloud.google.com/go/scheduler",
    version = "v1.9.0",
)

go_repo(
    module = "github.com/go-ole/go-ole",
    version = "v1.2.6",
)

go_repo(
    module = "cloud.google.com/go/bigquery",
    version = "v1.50.0",
)

go_repo(
    module = "cloud.google.com/go/contactcenterinsights",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/containeranalysis",
    version = "v0.9.0",
)

go_repo(
    module = "cloud.google.com/go/trace",
    version = "v1.9.0",
)

go_repo(
    module = "cloud.google.com/go/vmwareengine",
    version = "v0.3.0",
)

go_repo(
    module = "github.com/honeycombio/otel-config-go",
    version = "v1.10.0",
)

go_repo(
    module = "google.golang.org/appengine",
    version = "v1.6.7",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp",
    version = "v0.39.0",
)

go_repo(
    module = "cloud.google.com/go/workflows",
    version = "v1.10.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/otlpmetric",
    version = "v0.39.0",
)

go_repo(
    module = "modernc.org/ccgo/v3",
    version = "v3.16.9",
)

go_repo(
    module = "cloud.google.com/go/optimization",
    version = "v1.3.1",
)

go_repo(
    module = "cloud.google.com/go/pubsublite",
    version = "v1.7.0",
)

go_repo(
    module = "cloud.google.com/go/recaptchaenterprise",
    version = "v1.3.1",
)

go_repo(
    module = "github.com/ajstarks/deck",
    version = "v0.0.0-20200831202436-30c9fc6549a9",
)

go_repo(
    module = "google.golang.org/genproto/googleapis/rpc",
    version = "v0.0.0-20230530153820-e85fd2cbaebc",
)

go_repo(
    module = "github.com/iancoleman/strcase",
    version = "v0.2.0",
)

go_repo(
    module = "cloud.google.com/go/dataflow",
    version = "v0.8.0",
)

go_repo(
    module = "cloud.google.com/go/dataform",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/recaptchaenterprise/v2",
    version = "v2.7.0",
)

go_repo(
    module = "cloud.google.com/go/retail",
    version = "v1.12.0",
)

go_repo(
    module = "go.opentelemetry.io/contrib/instrumentation/runtime",
    version = "v0.42.0",
)

go_repo(
    module = "modernc.org/memory",
    version = "v1.2.1",
)

go_repo(
    module = "cloud.google.com/go/dataqna",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/metastore",
    version = "v1.10.0",
)

go_repo(
    module = "github.com/phpdave11/gofpdf",
    version = "v1.4.2",
)

go_repo(
    module = "github.com/yusufpapurcu/wmi",
    version = "v1.2.3",
)

go_repo(
    module = "go.uber.org/multierr",
    version = "v1.11.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/exporters/otlp/internal/retry",
    version = "v1.16.0",
)

go_repo(
    module = "cloud.google.com/go/logging",
    version = "v1.7.0",
)

go_repo(
    module = "cloud.google.com/go/maps",
    version = "v0.7.0",
)

go_repo(
    module = "cloud.google.com/go/networksecurity",
    version = "v0.8.0",
)

go_repo(
    module = "cloud.google.com/go/orchestration",
    version = "v1.6.0",
)

go_repo(
    module = "cloud.google.com/go/vision/v2",
    version = "v2.7.0",
)

go_repo(
    module = "github.com/go-fonts/stix",
    version = "v0.1.0",
)

go_repo(
    module = "github.com/rogpeppe/go-internal",
    version = "v1.9.0",
)

go_repo(
    module = "github.com/cncf/xds/go",
    version = "v0.0.0-20230310173818-32f1caf87195",
)

go_repo(
    module = "cloud.google.com/go/channel",
    version = "v1.12.0",
)

go_repo(
    module = "go.opentelemetry.io/otel/sdk",
    version = "v1.16.0",
)

go_repo(
    module = "google.golang.org/genproto/googleapis/bytestream",
    version = "v0.0.0-20230530153820-e85fd2cbaebc",
)
