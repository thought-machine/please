go_benchmark(
    name = "benchmark",
    srcs = ["benchmark_test.go"],
)

go_test(
    name = "benchmark_test",
    srcs = ["bench_test.go"],
    data = [":benchmark"],
    deps = ["///third_party/go/github.com_stretchr_testify//assert"],
)
