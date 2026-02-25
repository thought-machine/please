module github.com/thought-machine/please

go 1.25.0

require (
	cloud.google.com/go/longrunning v0.5.5
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/ProtonMail/go-crypto v0.0.0-20210920135941-2c5829bbf927
	github.com/alessio/shellescape v1.4.2
	github.com/bazelbuild/remote-apis v0.0.0-20240409135018-1f36c310b28d
	github.com/bazelbuild/remote-apis-sdks v0.0.0-20221114180157-e62cf9b8696a
	github.com/cespare/xxhash/v2 v2.3.0
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
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.62.0
	github.com/shirou/gopsutil/v3 v3.24.2
	github.com/sigstore/sigstore v1.10.4
	github.com/sigstore/sigstore/pkg/signature/kms/gcp v1.8.2
	github.com/sourcegraph/go-diff v0.7.0
	github.com/sourcegraph/go-lsp v0.0.0-20240223163137-f80c5dd31dfd
	github.com/sourcegraph/jsonrpc2 v0.2.0
	github.com/stretchr/testify v1.11.1
	github.com/texttheater/golang-levenshtein v1.0.1
	github.com/thought-machine/go-flags v1.6.3
	github.com/ulikunitz/xz v0.5.14
	github.com/zeebo/blake3 v0.2.3
	go.uber.org/automaxprocs v1.5.3
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225
	golang.org/x/net v0.47.0
	golang.org/x/sync v0.18.0
	golang.org/x/sys v0.38.0
	golang.org/x/term v0.37.0
	golang.org/x/tools v0.39.0
	google.golang.org/genproto/googleapis/bytestream v0.0.0-20240304212257-790db918fca8
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250825161204-c5933d9347a5
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/op/go-logging.v1 v1.0.0-20160211212156-b2cb9fa56473
)

require (
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/kms v1.15.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-containerregistry v0.20.7 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/jellydator/ttlcache/v3 v3.2.0 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/lufia/plan9stats v0.0.0-20240226150601-1dcf7310316a // indirect
	github.com/mostynb/zstdpool-syncpool v0.0.13 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.9.1 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sigstore/protobuf-specs v0.5.0 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/oauth2 v0.33.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	google.golang.org/api v0.168.0 // indirect
	google.golang.org/genproto v0.0.0-20240304212257-790db918fca8 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250825161204-c5933d9347a5 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
