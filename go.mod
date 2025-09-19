module github.com/thought-machine/please

go 1.23.0

require (
	cloud.google.com/go/longrunning v0.5.5
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/ProtonMail/go-crypto v0.0.0-20210920135941-2c5829bbf927
	github.com/alessio/shellescape v1.4.2
	github.com/bazelbuild/remote-apis v0.0.0-20240409135018-1f36c310b28d
	github.com/bazelbuild/remote-apis-sdks v0.0.0-20221114180157-e62cf9b8696a
	github.com/cespare/xxhash/v2 v2.2.0
	github.com/coreos/go-semver v0.3.1
	github.com/davecgh/go-spew v1.1.1
	github.com/djherbis/atime v1.1.0
	github.com/dustin/go-humanize v1.0.1
	github.com/fsnotify/fsnotify v1.7.0
	github.com/golang/protobuf v1.5.4
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/go-retryablehttp v0.7.7
	github.com/jstemmer/go-junit-report/v2 v2.1.0
	github.com/karrick/godirwalk v1.17.0
	github.com/manifoldco/promptui v0.9.0
	github.com/peterebden/go-cli-init/v5 v5.2.1
	github.com/peterebden/go-deferred-regex v1.1.0
	github.com/peterebden/go-sri v1.1.1
	github.com/peterebden/tools v0.0.0-20190805132753-b2a0db951d2a
	github.com/pkg/xattr v0.4.9
	github.com/please-build/buildtools v0.0.0-20240111140234-77ffe55926d9
	github.com/please-build/gcfg v1.7.0
	github.com/prometheus/client_golang v1.19.0
	github.com/prometheus/common v0.50.0
	github.com/shirou/gopsutil/v3 v3.24.2
	github.com/sigstore/sigstore v1.8.2
	github.com/sigstore/sigstore/pkg/signature/kms/gcp v1.8.2
	github.com/sourcegraph/go-diff v0.7.0
	github.com/sourcegraph/go-lsp v0.0.0-20240223163137-f80c5dd31dfd
	github.com/sourcegraph/jsonrpc2 v0.2.0
	github.com/stretchr/testify v1.9.0
	github.com/texttheater/golang-levenshtein v1.0.1
	github.com/thought-machine/go-flags v1.6.3
	github.com/ulikunitz/xz v0.5.11
	github.com/zeebo/blake3 v0.2.3
	go.uber.org/automaxprocs v1.5.3
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225
	golang.org/x/net v0.38.0
	golang.org/x/sync v0.12.0
	golang.org/x/sys v0.31.0
	golang.org/x/term v0.30.0
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d
	google.golang.org/genproto/googleapis/bytestream v0.0.0-20240304212257-790db918fca8
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240304212257-790db918fca8
	google.golang.org/grpc v1.62.1
	google.golang.org/protobuf v1.33.0
	gopkg.in/op/go-logging.v1 v1.0.0-20160211212156-b2cb9fa56473
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/kms v1.15.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/golang/glog v1.2.4 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-containerregistry v0.19.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/jellydator/ttlcache/v3 v3.2.0 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/letsencrypt/boulder v0.0.0-20240306190618-9b05c38eb38a // indirect
	github.com/lufia/plan9stats v0.0.0-20240226150601-1dcf7310316a // indirect
	github.com/mostynb/zstdpool-syncpool v0.0.13 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.0 // indirect
	github.com/prometheus/procfs v0.13.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.8.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/titanous/rocacheck v0.0.0-20171023193734-afe73141d399 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/sdk v1.22.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/oauth2 v0.27.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/api v0.168.0 // indirect
	google.golang.org/genproto v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240304212257-790db918fca8 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
