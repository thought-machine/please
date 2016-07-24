package(default_visibility = ['PUBLIC'])

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
    name = 'cover',
    get = 'golang.org/x/tools/cover',
    revision = 'c0008c5889c0d5091cdfefd2bfb08bff96527879',
)

go_get(
    name = 'gcfg',
    get = 'gopkg.in/gcfg.v1',
    patch = 'gcfg_dynamic_fields.patch',
    revision = 'a3bc0742e3bbfd16123e6cee742cadec9a3301d2',
)

go_get(
    name = 'go-bindata',
    binary = True,
    get = 'github.com/jteeuwen/go-bindata/...',
    revision = 'a0ff2567cfb70903282db057e799fd826784d41d',
)

go_get(
    name = 'go-flags',
    get = 'github.com/jessevdk/go-flags',
    revision = '0a28dbe50f23d8fce6b016975b964cfe7b97a20a',
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
    name = 'grpc',
    get = 'google.golang.org/grpc',
    install = ['google.golang.org/grpc/health'],
    revision = '461dac99975b211ed3eda7eb45b997d82da4345a',
    deps = [
        ':protobuf',
    ],
)

go_get(
    name = 'protoc-gen-go',
    binary = True,
    get = 'github.com/golang/protobuf/protoc-gen-go',
    revision = '3852dcfda249c2097355a6aabb199a28d97b30df',
)

go_get(
    name = 'protobuf',
    get = 'github.com/golang/protobuf/proto',
    revision = '3852dcfda249c2097355a6aabb199a28d97b30df',
)

go_get(
    name = 'testify',
    get = 'github.com/stretchr/testify',
    revision = 'f390dcf405f7b83c997eac1b06768bb9f44dec18',
)

go_get(
    name = 'osext',
    get = 'github.com/kardianos/osext',
    revision = '29ae4ffbc9a6fe9fb2bc5029050ce6996ea1d3bc',
)

go_get(
    name = 'prompter',
    get = 'github.com/Songmu/prompter',
    revision = 'f49666b0047d12850875d771340e1d862d9e7a0c',
    deps = [
        ':terminal',
    ],
)

go_get(
    name = 'stringer',
    binary = True,
    get = 'golang.org/x/tools/cmd/stringer',
    revision = 'f8ecfdb6b66ec67ca930b39806519a2b72d1eaca',
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
)

go_get(
    name = 'concurrent-map',
    get = 'github.com/streamrail/concurrent-map',
    revision = '5fc745307dc80a1883243b978f7e7c0fd5ce7206',
)
