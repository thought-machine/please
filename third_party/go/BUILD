package(default_visibility = ['PUBLIC'])

# This is needed to break a circular dependency.
package(jarcat_tool = '//tools/jarcat:jarcat_unzip')

go_get(
    name = 'logging',
    get = 'gopkg.in/op/go-logging.v1',
    revision = 'b2cb9fa56473e98db8caba80237377e83fe44db5',
)

go_get(
    name = 'terminal',
    get = 'golang.org/x/crypto/ssh/terminal',
    revision = '7b85b097bf7527677d54d3220065e966a0e3b613',
)

go_get(
    name = 'warnings',
    get = 'gopkg.in/warnings.v0',
    revision = 'v0.1.2',
)

go_get(
    name = 'gcfg',
    get = 'gopkg.in/gcfg.v1',
    install = [
        '',
        'scanner',
        'token',
        'types',
    ],
    patch = 'gcfg_dynamic_fields.patch',
    revision = '27e4946190b4a327b539185f2b5b1f7c84730728',
    deps = [
        ':warnings',
    ],
)

go_get(
    name = 'go-bindata',
    binary = True,
    get = 'github.com/kevinburke/go-bindata/...',
    revision = '46eb4c183bfc1ebb527d9d19bcded39476302eb8',
    strip = ['testdata'],
)

go_get(
    name = 'go-flags',
    get = 'github.com/jessevdk/go-flags',
    patch = 'flags_subcommand.patch',
    revision = 'v1.3.0',
)

go_get(
    name = 'humanize',
    get = 'github.com/dustin/go-humanize',
    revision = '8929fe90cee4b2cb9deb468b51fb34eba64d1bf0',
)

go_get(
    name = 'mux',
    get = 'github.com/gorilla/mux',
    revision = '9c068cf16d982f8bd444b8c352acbeec34c4fe5b',
    deps = [
        ':gorilla_context',
    ],
)

go_get(
    name = 'gorilla_context',
    get = 'github.com/gorilla/context',
    revision = '1c83b3eabd45b6d76072b66b746c20815fb2872d',
)

go_get(
    name = 'net',
    get = 'golang.org/x/net/...',
    revision = '136a25c244d3019482a795d728110278d6ba09a4',
    deps = [
        ':terminal',
        ':text',
    ],
)

go_get(
    name = 'text',
    get = 'golang.org/x/text/...',
    revision = '4e4a3210bb54bb31f6ab2cdca2edcc0b50c420c1',
    deps = [
        ':tools',
    ],
)

go_get(
    name = 'tools',
    get = 'golang.org/x/tools',
    install = [
        'cover',
    ],
    revision = '2ae76fd1560b622911f444c1e66b70a857e1de67',
)

go_get(
    name = 'grpc',
    exported_deps = [':net'],
    get = 'google.golang.org/grpc',
    install = [
        '',
        'balancer',
        'balancer/base',
        'balancer/roundrobin',
        'codes',
        'connectivity',
        'credentials',
        'encoding',
        'encoding/gzip',
        'encoding/proto',
        'grpclb/grpc_lb_v1/messages',
        'grpclog',
        'health',
        'health/grpc_health_v1',
        'internal',
        'keepalive',
        'metadata',
        'naming',
        'peer',
        'resolver',
        'resolver/dns',
        'resolver/passthrough',
        'stats',
        'status',
        'tap',
        'transport',
    ],
    revision = 'v1.10.0',
    deps = [
        ':protobuf',
        ':rpcstatus',
    ],
)

#TODO: build from the actual proto.
go_get(
    name = 'rpcstatus',
    get = 'google.golang.org/genproto/googleapis/rpc/status',
    revision = '2b5a72b8730b0b16380010cfe5286c42108d88e7',
    deps = [':protobuf'],
)

go_get(
    name = 'protobuf',
    get = 'github.com/golang/protobuf/...',
    revision = '130e6b02ab059e7b717a096f397c5b60111cae74',
)

go_get(
    name = 'protoc-gen-go',
    binary = True,
    get = 'github.com/golang/protobuf/protoc-gen-go',
    revision = '130e6b02ab059e7b717a096f397c5b60111cae74',
    deps = [
        ':protobuf',
    ],
)

go_get(
    name = 'spew',
    get = 'github.com/davecgh/go-spew/spew',
    patch = 'spew_omit_empty.patch',
    revision = 'ecdeabc65495df2dec95d7c4a4c3e021903035e5',
)

go_get(
    name = 'testify',
    get = 'github.com/stretchr/testify',
    install = [
        'assert',
        'require',
        'vendor/github.com/davecgh/go-spew/spew',
        'vendor/github.com/pmezard/go-difflib/difflib',
    ],
    revision = 'f390dcf405f7b83c997eac1b06768bb9f44dec18',
    deps = [':spew'],
)

go_get(
    name = 'go-isatty',
    get = 'github.com/mattn/go-isatty',
    revision = '6ca4dbf54d38eea1a992b3c722a76a5d1c4cb25c',
)

go_get(
    name = 'prompter',
    get = 'github.com/Songmu/prompter',
    revision = 'f49666b0047d12850875d771340e1d862d9e7a0c',
    deps = [
        ':go-isatty',
        ':terminal',
    ],
)

go_get(
    name = 'stringer',
    binary = True,
    get = 'golang.org/x/tools/cmd/stringer',
    revision = 'f8ecfdb6b66ec67ca930b39806519a2b72d1eaca',
    deps = [':tools'],
)

go_get(
    name = 'levenshtein',
    get = 'github.com/texttheater/golang-levenshtein/levenshtein',
    revision = '14026fface0cb806188c85e792a93d625dc37d0f',
)

go_get(
    name = 'queue',
    get = 'github.com/Workiva/go-datastructures/queue',
    revision = '064f3ea06ab2337e6eaf42052392d14f779e91cb',
)

go_get(
    name = 'fsnotify',
    get = 'github.com/fsnotify/fsnotify',
    revision = 'a8a77c9133d2d6fd8334f3260d06f60e8d80a5fb',
    deps = [':unix'],
)

go_get(
    name = 'concurrent-map',
    get = 'github.com/streamrail/concurrent-map',
    revision = '5fc745307dc80a1883243b978f7e7c0fd5ce7206',
)

go_get(
    name = 'quantile',
    get = 'github.com/beorn7/perks/quantile',
    revision = '4c0e84591b9aa9e6dcfdf3e020114cd81f89d5f9',
)

go_get(
    name = 'pbutil',
    get = 'github.com/matttproud/golang_protobuf_extensions/pbutil',
    revision = 'c12348ce28de40eed0136aa2b644d0ee0650e56c',
    deps = [':protobuf'],
)

go_get(
    name = 'prometheus',
    get = 'github.com/prometheus/client_golang/prometheus/...',
    revision = 'c5b7fccd204277076155f10851dad72b76a49317',
    deps = [
        ':grpc',
        ':pbutil',
        ':procfs',
        ':prometheus_client_model',
        ':prometheus_common',
        ':protobuf',
        ':quantile',
    ],
)

go_get(
    name = 'prometheus_common',
    get = 'github.com/prometheus/common',
    install = [
        'expfmt',
        'model',
        'internal/bitbucket.org/ww/goautoneg',
    ],
    revision = '89604d197083d4781071d3c65855d24ecfb0a563',
    deps = [
        ':pbutil',
        ':prometheus_client_model',
        ':protobuf',
    ],
)

go_get(
    name = 'prometheus_client_model',
    get = 'github.com/prometheus/client_model/go',
    revision = '99fa1f4be8e564e8a6b613da7fa6f46c9edafc6c',
    deps = [
        ':protobuf',
    ],
)

go_get(
    name = 'procfs',
    get = 'github.com/prometheus/procfs',
    revision = 'abf152e5f3e97f2fafac028d2cc06c1feb87ffa5',
    strip = ['fixtures'],  # Test fixture has a symlink to /usr/bin/vim which might not exist
)

go_get(
    name = 'grpc-prometheus',
    get = 'github.com/grpc-ecosystem/go-grpc-prometheus',
    revision = '34abd90a014618f61222a1b0a7b7eb834a2d0dc3',
    deps = [
        ':grpc',
        ':procfs',
        ':prometheus',
        ':protobuf',
    ],
)

go_get(
    name = 'grpc-middleware',
    get = 'github.com/grpc-ecosystem/go-grpc-middleware',
    install = [
        'retry',
        'util/metautils',
        'util/backoffutils',
    ],
    revision = 'fa8fef87dcecac0bda02d36abb3c790ab9e0030b',
    deps = [
        ':grpc',
        ':net',
        ':protobuf',
    ],
)

go_get(
    name = 'shlex',
    get = 'github.com/google/shlex',
    revision = '6f45313302b9c56850fc17f99e40caebce98c716',
)

go_get(
    name = 'semver',
    get = 'github.com/coreos/go-semver/semver',
    revision = '8ab6407b697782a06568d4b7f1db25550ec2e4c6',
)

go_get(
    name = 'memberlist',
    get = 'github.com/hashicorp/memberlist',
    revision = '7ad712f5f34ec40aebe6ca47756d07898486a8d2',
    deps = [
        ':dns',
        ':go-metrics',
        ':go-msgpack',
        ':go-multierror',
    ],
)

go_get(
    name = 'go-msgpack',
    get = 'github.com/hashicorp/go-msgpack/...',
    revision = 'fa3f63826f7c23912c15263591e65d54d080b458',
)

go_get(
    name = 'go-multierror',
    get = 'github.com/hashicorp/go-multierror',
    install = [
        '',
        'vendor/github.com/hashicorp/errwrap',
    ],
    revision = 'b7773ae218740a7be65057fc60b366a49b538a44',
)

go_get(
    name = 'go-metrics',
    get = 'github.com/armon/go-metrics',
    revision = '7aa49fde808223f8dadfdbfd3a20ff6c19e5f9ec',
    deps = [
        ':go-immutable-radix',
        ':verify',
    ],
)

go_get(
    name = 'go-immutable-radix',
    get = 'github.com/hashicorp/go-immutable-radix',
    revision = '7f3cd4390caab3250a57f30efdb2a65dd7649ecf',
    deps = [
        ':go-uuid',
        ':simplelru',
    ],
)

go_get(
    name = 'simplelru',
    get = 'github.com/hashicorp/golang-lru/simplelru',
    revision = '0fb14efe8c47ae851c0034ed7a448854d3d34cf3',
)

go_get(
    name = 'go-uuid',
    get = 'github.com/hashicorp/go-uuid',
    revision = '64130c7a86d732268a38cb04cfbaf0cc987fda98',
)

go_get(
    name = 'verify',
    get = 'github.com/pascaldekloe/goe/verify',
    revision = 'c580d16984352a427218a8e43232313d1c1c6f84',
)

go_get(
    name = 'dns',
    get = 'github.com/miekg/dns/...',
    revision = 'be5ae6ca7ac994584d2e30167737687f1c1ded8e',
)

go_get(
    name = 'atime',
    get = 'github.com/djherbis/atime',
    revision = '89517e96e10b93292169a79fd4523807bdc5d5fa',
)

go_get(
    name = 'errgroup',
    get = 'golang.org/x/sync/errgroup',
    revision = '457c5828408160d6a47e17645169cf8fa20218c4',
    deps = [':net'],
)

go_get(
    name = 'psutil',
    get = 'github.com/shirou/gopsutil',
    install = [
        'cpu',
        'internal/common',
        'mem',
    ],
    revision = 'v2.17.09',
    deps = [':unix'],
)

go_get(
    name = 'unix',
    get = 'golang.org/x/sys/unix',
    revision = 'ebfc5b4631820b793c9010c87fd8fef0f39eb082',
)

go_get(
    name = 'openpgp',
    get = 'golang.org/x/crypto/openpgp/...',
    revision = '077efaa604f994162e3307fafe5954640763fc08',
    deps = [':cast5'],
)

go_get(
    name = 'cast5',
    get = 'golang.org/x/crypto/cast5',
    revision = '077efaa604f994162e3307fafe5954640763fc08',
)

go_get(
    name = 'go-fuzz-build',
    binary = True,
    get = 'github.com/dvyukov/go-fuzz/go-fuzz-build',
    revision = '445b00a1141b27425541ee8d7dc2f524faf202a9',
    deps = [':go-fuzz-dep'],
)

go_get(
    name = 'go-fuzz',
    binary = True,
    get = 'github.com/dvyukov/go-fuzz/go-fuzz',
    revision = '445b00a1141b27425541ee8d7dc2f524faf202a9',
    deps = [':go-fuzz-dep'],
)

go_get(
    name = 'go-fuzz-dep',
    get = 'github.com/dvyukov/go-fuzz',
    install = [
        'go-fuzz-dep',
        'go-fuzz-defs',
    ],
    revision = '445b00a1141b27425541ee8d7dc2f524faf202a9',
)

go_get(
    name = 'godirwalk',
    get = 'github.com/karrick/godirwalk',
    revision = '17c244300a0e2101858b851a8df1ae8e796a0bca',
    strip = [
        'examples',
        'testdata',
        'vendor',
    ],
    deps = [':errors'],
)

go_get(
    name = 'errors',
    get = 'github.com/pkg/errors',
    revision = '30136e27e2ac8d167177e8a583aa4c3fea5be833',
)
