package remote

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"testing"
	"time"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
	bs "google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/genproto/googleapis/longrunning"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/thought-machine/please/src/core"
)

func TestInit(t *testing.T) {
	c := newClient()
	assert.NoError(t, c.CheckInitialised())
}

func TestBadAPIVersion(t *testing.T) {
	// We specify a required API version of v2.0.0, so should fail initialisation if the server
	// specifies something incompatible with that.
	defer server.Reset()
	server.HighApiVersion.Major = 1
	server.LowApiVersion.Major = 1
	c := newClient()
	assert.Error(t, c.CheckInitialised())
	assert.Contains(t, c.CheckInitialised().Error(), "1.0.0 - 1.1.0")
}

func TestUnsupportedDigest(t *testing.T) {
	defer server.Reset()
	server.DigestFunction = []pb.DigestFunction_Value{
		pb.DigestFunction_SHA256,
		pb.DigestFunction_SHA384,
		pb.DigestFunction_SHA512,
	}
	c := newClient()
	assert.Error(t, c.CheckInitialised())
}

func TestStoreAndRetrieve(t *testing.T) {
	c := newClient()
	c.CheckInitialised()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target1"})
	target.AddSource(core.FileLabel{File: "src1.txt", Package: "package"})
	target.AddSource(core.FileLabel{File: "src2.txt", Package: "package"})
	target.AddOutput("out1.txt")
	target.PostBuildFunction = testFunction{}
	// The hash is arbitrary as far as this package is concerned.
	key, _ := hex.DecodeString("2dd283abb148ebabcd894b306e3d86d0390c82a7")
	metadata := &core.BuildMetadata{
		Stdout:    []byte("test stdout"),
		StartTime: time.Now().UTC(),
		EndTime:   time.Now().UTC(),
	}
	err := c.Store(target, key, metadata, []string{"out1.txt"})
	assert.NoError(t, err)
	// Remove the old file, but remember its contents so we can compare later.
	contents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	err = os.Remove("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	// Now retrieve back the output of this thing.
	retrievedMetadata, err := c.Retrieve(target, key)
	assert.NoError(t, err)
	cachedContents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	assert.Equal(t, contents, cachedContents)
	assert.Equal(t, metadata, retrievedMetadata)
}

func TestExecuteBuild(t *testing.T) {
	c := newClient()
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target2"})
	target.AddSource(core.FileLabel{File: "src1.txt", Package: "package"})
	target.AddSource(core.FileLabel{File: "src2.txt", Package: "package"})
	target.AddOutput("out2.txt")
	target.BuildTimeout = time.Minute
	// We need to set this to force stdout to be retrieved (it is otherwise unnecessary
	// on success).
	target.PostBuildFunction = testFunction{}
	target.Command = "echo hello && echo test > $OUT"
	metadata, err := c.Build(0, target, []byte("stampystampystamp"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello\n"), metadata.Stdout)
}

func TestExecuteTest(t *testing.T) {
	c := newClientInstance("test")
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target3"})
	target.AddOutput("remote_test")
	target.TestTimeout = time.Minute
	target.TestCommand = "$TEST"
	target.IsTest = true
	target.IsBinary = true
	target.SetState(core.Building)
	_, results, coverage, err := c.Test(0, target)
	assert.NoError(t, err)
	assert.Equal(t, testResults, results)
	assert.Equal(t, 0, len(coverage)) // Wasn't requested
}

func TestExecuteTestWithCoverage(t *testing.T) {
	c := newClientInstance("test")
	c.state.NeedCoverage = true // bit of a hack but we need to turn this on somehow
	target := core.NewBuildTarget(core.BuildLabel{PackageName: "package", Name: "target4"})
	target.AddOutput("remote_test")
	target.TestTimeout = time.Minute
	target.TestCommand = "$TEST"
	target.IsTest = true
	target.IsBinary = true
	target.SetState(core.Built)
	_, results, coverage, err := c.Test(0, target)
	assert.NoError(t, err)
	assert.Equal(t, testResults, results)
	assert.Equal(t, coverageData, coverage)
}

var testResults = []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<testcase name="//src/remote:remote_test">
  <test name="testResults" success="true" time="172" type="SUCCESS"/>
</testcase>
`)

var coverageData = []byte(`mode: set
src/core/build_target.go:134.54,143.2 7 0
src/core/build_target.go:159.52,172.2 12 0
src/core/build_target.go:177.44,179.2 1 0
`)

func newClient() *Client {
	return newClientInstance("")
}

func newClientInstance(name string) *Client {
	config := core.DefaultConfiguration()
	config.Build.Path = []string{"/usr/local/bin", "/usr/bin", "/bin"}
	config.Remote.NumExecutors = 1
	config.Remote.Instance = name
	state := core.NewBuildState(config)
	state.Config.Remote.URL = "127.0.0.1:9987"
	return New(state)
}

// A testServer implements the server interface for the various servers we test against.
type testServer struct {
	DigestFunction                []pb.DigestFunction_Value
	LowApiVersion, HighApiVersion semver.SemVer
	actionResults                 map[string]*pb.ActionResult
	blobs                         map[string][]byte
	bytestreams                   map[string][]byte
}

func (s *testServer) GetCapabilities(ctx context.Context, req *pb.GetCapabilitiesRequest) (*pb.ServerCapabilities, error) {
	return &pb.ServerCapabilities{
		CacheCapabilities: &pb.CacheCapabilities{
			DigestFunction: s.DigestFunction,
			ActionCacheUpdateCapabilities: &pb.ActionCacheUpdateCapabilities{
				UpdateEnabled: true,
			},
			MaxBatchTotalSizeBytes: 2048,
		},
		ExecutionCapabilities: &pb.ExecutionCapabilities{
			// TODO(peterebden): this should probably be SHA256 to mimic what we will
			//                   likely find in the wild.
			DigestFunction: pb.DigestFunction_SHA1,
			ExecEnabled:    true,
		},
		LowApiVersion:  &s.LowApiVersion,
		HighApiVersion: &s.HighApiVersion,
	}, nil
}

func (s *testServer) Reset() {
	s.DigestFunction = []pb.DigestFunction_Value{
		pb.DigestFunction_SHA1,
		pb.DigestFunction_SHA256,
	}
	s.LowApiVersion = semver.SemVer{Major: 2}
	s.HighApiVersion = semver.SemVer{Major: 2, Minor: 1}
	s.actionResults = map[string]*pb.ActionResult{}
	s.blobs = map[string][]byte{}
	s.bytestreams = map[string][]byte{}
}

func (s *testServer) GetActionResult(ctx context.Context, req *pb.GetActionResultRequest) (*pb.ActionResult, error) {
	ar, present := s.actionResults[req.ActionDigest.Hash]
	if !present {
		return nil, status.Errorf(codes.NotFound, "action result not found")
	}
	if req.InlineStdout && ar.StdoutDigest != nil {
		ar.StdoutRaw = s.blobs[ar.StdoutDigest.Hash]
	}
	return ar, nil
}

func (s *testServer) UpdateActionResult(ctx context.Context, req *pb.UpdateActionResultRequest) (*pb.ActionResult, error) {
	s.actionResults[req.ActionDigest.Hash] = req.ActionResult
	return req.ActionResult, nil
}

func (s *testServer) FindMissingBlobs(ctx context.Context, req *pb.FindMissingBlobsRequest) (*pb.FindMissingBlobsResponse, error) {
	resp := &pb.FindMissingBlobsResponse{}
	for _, d := range req.BlobDigests {
		if _, present := s.blobs[d.Hash]; !present {
			resp.MissingBlobDigests = append(resp.MissingBlobDigests, d)
		}
	}
	return resp, nil
}

func (s *testServer) BatchUpdateBlobs(ctx context.Context, req *pb.BatchUpdateBlobsRequest) (*pb.BatchUpdateBlobsResponse, error) {
	resp := &pb.BatchUpdateBlobsResponse{
		Responses: make([]*pb.BatchUpdateBlobsResponse_Response, len(req.Requests)),
	}
	for i, r := range req.Requests {
		resp.Responses[i] = &pb.BatchUpdateBlobsResponse_Response{
			Status: &rpcstatus.Status{},
		}
		if len(r.Data) != int(r.Digest.SizeBytes) {
			resp.Responses[i].Status.Code = int32(codes.InvalidArgument)
			resp.Responses[i].Status.Message = fmt.Sprintf("Blob sizes do not match (%d / %d)", len(r.Data), r.Digest.SizeBytes)
		} else {
			s.blobs[r.Digest.Hash] = r.Data
		}
	}
	return resp, nil
}

func (s *testServer) BatchReadBlobs(ctx context.Context, req *pb.BatchReadBlobsRequest) (*pb.BatchReadBlobsResponse, error) {
	resp := &pb.BatchReadBlobsResponse{
		Responses: make([]*pb.BatchReadBlobsResponse_Response, len(req.Digests)),
	}
	for i, d := range req.Digests {
		resp.Responses[i] = &pb.BatchReadBlobsResponse_Response{
			Status: &rpcstatus.Status{},
			Digest: d,
		}
		if data, present := s.blobs[d.Hash]; present {
			resp.Responses[i].Data = data
		} else {
			resp.Responses[i].Status.Code = int32(codes.NotFound)
			resp.Responses[i].Status.Message = fmt.Sprintf("Blob %s not found", d.Hash)
		}
	}
	return resp, nil
}

func (s *testServer) GetTree(*pb.GetTreeRequest, pb.ContentAddressableStorage_GetTreeServer) error {
	return status.Errorf(codes.Unimplemented, "GetTree not implemented for test")
}

func (s *testServer) Read(req *bs.ReadRequest, srv bs.ByteStream_ReadServer) error {
	blobName, err := s.bytestreamBlobName(req.ResourceName)
	if err != nil {
		return err
	}
	b, present := s.blobs[blobName]
	if !present {
		return status.Errorf(codes.NotFound, "bytestream %s not found", blobName)
	} else if req.ReadOffset < 0 || req.ReadOffset > int64(len(b)) {
		return status.Errorf(codes.OutOfRange, "invalid offset for bytestream %s, was %d, must be [0-%d]", req.ResourceName, req.ReadOffset, len(b))
	} else if req.ReadLimit < 0 {
		return status.Errorf(codes.OutOfRange, "negative ReadLimit")
	}
	b = b[req.ReadOffset:]
	if req.ReadLimit != 0 && req.ReadLimit < int64(len(b)) {
		b = b[:req.ReadLimit]
	}
	// Now stream these back a bit at a time.
	for i := 0; i < len(b); i += 1024 {
		n := i + 1024
		if n > len(b) {
			n = len(b)
		}
		if err := srv.Send(&bs.ReadResponse{Data: b[i:n]}); err != nil {
			return fmt.Errorf("ByteStream::Read error: %s", err)
		}
	}
	return nil
}

func (s *testServer) Write(srv bs.ByteStream_WriteServer) error {
	req, err := srv.Recv()
	if err != nil {
		return fmt.Errorf("ByteStream::Write error: %s", err)
	} else if req.ResourceName == "" {
		return status.Errorf(codes.InvalidArgument, "missing ResourceName")
	}
	name := req.ResourceName
	blobName, err := s.bytestreamBlobName(name)
	if err != nil {
		return err
	}
	b := s.bytestreams[name]
	for {
		if req.WriteOffset != int64(len(b)) {
			return status.Errorf(codes.InvalidArgument, "incorrect WriteOffset (was %d, should be %d)", req.WriteOffset, len(b))
		}
		b = append(b, req.Data...)
		if req.FinishWrite {
			s.blobs[blobName] = b
			delete(s.bytestreams, name)
			break
		}
		s.bytestreams[name] = b
		req, err = srv.Recv()
		if err != nil {
			return fmt.Errorf("ByteStream::Write error: %s", err)
		}
	}
	return srv.SendAndClose(&bs.WriteResponse{
		CommittedSize: int64(len(b)),
	})
}

func (s *testServer) bytestreamBlobName(bytestream string) (string, error) {
	r := regexp.MustCompile("(?:uploads/[0-9a-f-]+/)?blobs/([0-9a-f]+)/[0-9]+")
	matches := r.FindStringSubmatch(bytestream)
	if matches == nil {
		return "", status.Errorf(codes.InvalidArgument, "invalid ResourceName: %s", bytestream)
	}
	return matches[1], nil
}

func (s *testServer) QueryWriteStatus(ctx context.Context, req *bs.QueryWriteStatusRequest) (*bs.QueryWriteStatusResponse, error) {
	if b, present := s.blobs[req.ResourceName]; present {
		return &bs.QueryWriteStatusResponse{
			CommittedSize: int64(len(b)),
			Complete:      true,
		}, nil
	} else if b, present := s.bytestreams[req.ResourceName]; present {
		return &bs.QueryWriteStatusResponse{
			CommittedSize: int64(len(b)),
		}, nil
	}
	return nil, status.Errorf(codes.NotFound, "resource %s not found", req.ResourceName)
}

func (s *testServer) Execute(req *pb.ExecuteRequest, srv pb.Execution_ExecuteServer) error {
	mm := func(msg proto.Message) *any.Any {
		a, _ := ptypes.MarshalAny(msg)
		return a
	}
	srv.Send(&longrunning.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_CACHE_CHECK,
		}),
	})
	queued := toTimestamp(time.Now())
	srv.Send(&longrunning.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_QUEUED,
		}),
	})
	start := toTimestamp(time.Now())
	srv.Send(&longrunning.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_EXECUTING,
		}),
	})
	completed := toTimestamp(time.Now())

	// Keep stdout as a blob to force the client to download it.
	s.blobs["5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"] = []byte("hello\n")

	// We use this to conveniently identify whether the request was a test or not.
	if req.InstanceName == "test" {
		s.blobs["a4226cbd3e94a835ffcb5832ddd07eafe29e99494105b01d0df236bd8e9a15c3"] = testResults
		s.blobs["a7f899acaabeaeecea132f782a5ebdddccd76fa1041f3e6d4a6e0d58638ffa0a"] = coverageData
		srv.Send(&longrunning.Operation{
			Name: "geoff",
			Metadata: mm(&pb.ExecuteOperationMetadata{
				Stage: pb.ExecutionStage_COMPLETED,
			}),
			Done: true,
			Result: &longrunning.Operation_Response{
				Response: mm(&pb.ExecuteResponse{
					Result: &pb.ActionResult{
						OutputFiles: []*pb.OutputFile{{
							Path: "test.results",
							Digest: &pb.Digest{
								Hash:      "a4226cbd3e94a835ffcb5832ddd07eafe29e99494105b01d0df236bd8e9a15c3",
								SizeBytes: 181,
							},
						}, {
							Path: "test.coverage",
							Digest: &pb.Digest{
								Hash:      "a7f899acaabeaeecea132f782a5ebdddccd76fa1041f3e6d4a6e0d58638ffa0a",
								SizeBytes: 137,
							},
						}},
						ExitCode: 0,
						StdoutDigest: &pb.Digest{
							Hash:      "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
							SizeBytes: 6,
						},
						ExecutionMetadata: &pb.ExecutedActionMetadata{
							Worker:                      "kev",
							QueuedTimestamp:             queued,
							ExecutionStartTimestamp:     start,
							ExecutionCompletedTimestamp: completed,
						},
					},
					Status: &rpcstatus.Status{
						Code: int32(codes.OK),
					},
				}),
			},
		})
	} else {
		srv.Send(&longrunning.Operation{
			Name: "geoff",
			Metadata: mm(&pb.ExecuteOperationMetadata{
				Stage: pb.ExecutionStage_COMPLETED,
			}),
			Done: true,
			Result: &longrunning.Operation_Response{
				Response: mm(&pb.ExecuteResponse{
					Result: &pb.ActionResult{
						OutputFiles: []*pb.OutputFile{{
							Path: "out2.txt",
							Digest: &pb.Digest{
								Hash:      "5fb3d47e893061ea6627334a8582c37398cfdc68fe7fa59c16912e4a3ab7a5d6",
								SizeBytes: 19,
							},
						}},
						ExitCode: 0,
						StdoutDigest: &pb.Digest{
							Hash:      "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
							SizeBytes: 6,
						},
						ExecutionMetadata: &pb.ExecutedActionMetadata{
							Worker:                      "kev",
							QueuedTimestamp:             queued,
							ExecutionStartTimestamp:     start,
							ExecutionCompletedTimestamp: completed,
						},
					},
					Status: &rpcstatus.Status{
						Code: int32(codes.OK),
					},
					ServerLogs: map[string]*pb.LogFile{
						"test": {
							Digest: &pb.Digest{
								Hash:      "e70c151d26f755cea2162b627151416f4407ebc8502cea8e68f0d95a3950ea16",
								SizeBytes: 42,
							},
						},
					},
				}),
			},
		})
	}
	return nil
}

func (s *testServer) WaitExecution(*pb.WaitExecutionRequest, pb.Execution_WaitExecutionServer) error {
	return fmt.Errorf("not implemented")
}

var server = &testServer{}

// A testFunction is something we can assign to a target's PostBuildFunction; it will
// not be called but affects whether we bother trying to restore stdout or not.
type testFunction struct{}

func (f testFunction) String() string                           { return "test post-build function" }
func (f testFunction) Call(t *core.BuildTarget, o string) error { return nil }

func TestMain(m *testing.M) {
	server.Reset()
	lis, err := net.Listen("tcp", ":9987")
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", lis.Addr(), err)
	}
	s := grpc.NewServer()
	pb.RegisterCapabilitiesServer(s, server)
	pb.RegisterActionCacheServer(s, server)
	pb.RegisterContentAddressableStorageServer(s, server)
	bs.RegisterByteStreamServer(s, server)
	pb.RegisterExecutionServer(s, server)
	go s.Serve(lis)
	if err := os.Chdir("src/remote/test_data"); err != nil {
		log.Fatalf("Failed to chdir: %s", err)
	}
	code := m.Run()
	s.Stop()
	os.Exit(code)
}
