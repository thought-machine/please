package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"regexp"
	"testing"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	fpb "github.com/bazelbuild/remote-apis/build/bazel/remote/asset/v1"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/peterebden/go-sri"
	bs "google.golang.org/genproto/googleapis/bytestream"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/thought-machine/please/src/cache"
	"github.com/thought-machine/please/src/core"
)

func newClient() *Client {
	return newClientInstance("wibble")
}

func newClientInstance(name string) *Client {
	config := core.DefaultConfiguration()
	config.Build.Path = []string{"/usr/local/bin", "/usr/bin", "/bin"}
	config.Build.HashFunction = "sha256"
	config.Remote.NumExecutors = 1
	config.Remote.Instance = name
	config.Remote.Secure = false
	config.Remote.Platform = []string{"OSFamily=linux"}
	state := core.NewBuildState(config)
	state.Config.Remote.URL = "127.0.0.1:9987"
	state.Config.Remote.AssetURL = state.Config.Remote.URL
	state.Cache = cache.NewCache(state)
	return New(state)
}

// A testServer implements the server interface for the various servers we test against.
type testServer struct {
	DigestFunction                []pb.DigestFunction_Value
	LowAPIVersion, HighAPIVersion semver.SemVer
	actionResults                 map[string]*pb.ActionResult
	blobs                         map[string][]byte
	bytestreams                   map[string][]byte
	mockActionResult              *pb.ActionResult
}

func (s *testServer) GetCapabilities(ctx context.Context, req *pb.GetCapabilitiesRequest) (*pb.ServerCapabilities, error) {
	return &pb.ServerCapabilities{
		CacheCapabilities: &pb.CacheCapabilities{
			DigestFunctions: s.DigestFunction,
			ActionCacheUpdateCapabilities: &pb.ActionCacheUpdateCapabilities{
				UpdateEnabled: true,
			},
			MaxBatchTotalSizeBytes: 2048,
		},
		ExecutionCapabilities: &pb.ExecutionCapabilities{
			DigestFunction: pb.DigestFunction_SHA256,
			ExecEnabled:    true,
		},
		LowApiVersion:  &s.LowAPIVersion,
		HighApiVersion: &s.HighAPIVersion,
	}, nil
}

func (s *testServer) Reset() {
	s.DigestFunction = []pb.DigestFunction_Value{
		pb.DigestFunction_SHA1,
		pb.DigestFunction_SHA256,
	}
	s.LowAPIVersion = semver.SemVer{Major: 2}
	s.HighAPIVersion = semver.SemVer{Major: 2, Minor: 1}
	s.actionResults = map[string]*pb.ActionResult{}
	s.blobs = map[string][]byte{}
	s.bytestreams = map[string][]byte{}
	s.mockActionResult = nil
}

func (s *testServer) GetActionResult(ctx context.Context, req *pb.GetActionResultRequest) (*pb.ActionResult, error) {
	s.checkDigest(req.ActionDigest)
	ar, present := s.actionResults[req.ActionDigest.Hash]
	if !present {
		return nil, status.Errorf(codes.NotFound, "action result not found")
	}
	if req.InlineStdout && ar.StdoutDigest != nil {
		s.checkDigest(ar.StdoutDigest)
		ar.StdoutRaw = s.blobs[ar.StdoutDigest.Hash]
	}
	return ar, nil
}

func (s *testServer) UpdateActionResult(ctx context.Context, req *pb.UpdateActionResultRequest) (*pb.ActionResult, error) {
	s.checkDigest(req.ActionDigest)
	s.actionResults[req.ActionDigest.Hash] = req.ActionResult
	return req.ActionResult, nil
}

func (s *testServer) FindMissingBlobs(ctx context.Context, req *pb.FindMissingBlobsRequest) (*pb.FindMissingBlobsResponse, error) {
	resp := &pb.FindMissingBlobsResponse{}
	for _, d := range req.BlobDigests {
		s.checkDigest(d)
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
		s.checkDigest(r.Digest)
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
		s.checkDigest(d)
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
	mm := func(msg protoreflect.ProtoMessage) *anypb.Any {
		a := &anypb.Any{}
		a.MarshalFrom(msg)
		return a
	}
	srv.Send(&longrunningpb.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_CACHE_CHECK,
		}),
	})
	queued := timestamppb.Now()
	srv.Send(&longrunningpb.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_QUEUED,
		}),
	})
	start := timestamppb.Now()
	srv.Send(&longrunningpb.Operation{
		Name: "geoff",
		Metadata: mm(&pb.ExecuteOperationMetadata{
			Stage: pb.ExecutionStage_EXECUTING,
		}),
	})
	completed := timestamppb.Now()

	// Keep stdout as a blob to force the client to download it.
	s.blobs["5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"] = []byte("hello\n")

	// We use this to conveniently identify whether the request was a test or not.
	if req.InstanceName == "test" {
		s.blobs["a4226cbd3e94a835ffcb5832ddd07eafe29e99494105b01d0df236bd8e9a15c3"] = testResults
		s.blobs["a7f899acaabeaeecea132f782a5ebdddccd76fa1041f3e6d4a6e0d58638ffa0a"] = coverageData
		srv.Send(&longrunningpb.Operation{
			Name: "geoff",
			Metadata: mm(&pb.ExecuteOperationMetadata{
				Stage: pb.ExecutionStage_COMPLETED,
			}),
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: mm(&pb.ExecuteResponse{
					Result: &pb.ActionResult{
						OutputFiles: []*pb.OutputFile{{
							Path: "test.results",
							Digest: &pb.Digest{
								Hash:      "a4226cbd3e94a835ffcb5832ddd07eafe29e99494105b01d0df236bd8e9a15c3",
								SizeBytes: 180,
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
	} else if req.InstanceName == "mock" {
		srv.Send(&longrunningpb.Operation{
			Name: "geoff",
			Metadata: mm(&pb.ExecuteOperationMetadata{
				Stage: pb.ExecutionStage_COMPLETED,
			}),
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: mm(&pb.ExecuteResponse{
					Result: server.mockActionResult,
					Status: &rpcstatus.Status{
						Code: int32(codes.OK),
					},
				}),
			},
		})
	} else {
		s.blobs["aaaf60fab1ff6b3d8147bafa3d29cb3e985cf0265cbf53705372eaabcd76c06b"] = []byte("what is the meaning of life, the universe, and everything?\n")
		srv.Send(&longrunningpb.Operation{
			Name: "geoff",
			Metadata: mm(&pb.ExecuteOperationMetadata{
				Stage: pb.ExecutionStage_COMPLETED,
			}),
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: mm(&pb.ExecuteResponse{
					Result: &pb.ActionResult{
						OutputFiles: []*pb.OutputFile{{
							Path: "out2.txt",
							Digest: &pb.Digest{
								Hash:      "aaaf60fab1ff6b3d8147bafa3d29cb3e985cf0265cbf53705372eaabcd76c06b",
								SizeBytes: 60,
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

// checkDigest checks a digest is structurally valid and panics if not.
func (s *testServer) checkDigest(digest *pb.Digest) {
	const length = sha256.Size * 2 // times 2 for the hex encoding
	if len(digest.Hash) != length {
		panic(fmt.Errorf("Incorrect digest length; was %d, should be %d", len(digest.Hash), length))
	} else if _, err := hex.DecodeString(digest.Hash); err != nil {
		panic(fmt.Errorf("Invalid hex encoding for digest: %s", err))
	}
}

func (s *testServer) RecoverUnaryPanics(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic in handler for %s: %s", info.FullMethod, r)
			err = status.Errorf(codes.Unknown, "handler failed: %s", r)
		}
	}()
	return handler(ctx, req)
}

func (s *testServer) RecoverStreamPanics(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Panic in handler for %s: %s", info.FullMethod, r)
			err = status.Errorf(codes.Unknown, "handler failed: %s", r)
		}
	}()
	return handler(srv, ss)
}

func (s *testServer) FetchBlob(ctx context.Context, req *fpb.FetchBlobRequest) (*fpb.FetchBlobResponse, error) {
	// This is a little overly specific but wevs
	if len(req.Qualifiers) != 1 {
		return nil, fmt.Errorf("Expected exactly one qualifier, got %s", req.Qualifiers)
	} else if req.Qualifiers[0].Name != "checksum.sri" {
		return nil, fmt.Errorf("Missing checksum.sri qualifier")
	}
	sri, err := sri.NewChecker(req.Qualifiers[0].Value)
	if err != nil {
		return nil, err
	}
	sri.Write([]byte("abc"))
	if err := sri.Check(); err != nil {
		return nil, err
	}
	return &fpb.FetchBlobResponse{
		BlobDigest: &pb.Digest{
			Hash:      "edeaaff3f1774ad2888673770c6d64097e391bc362d7d6fb34982ddf0efd18cb",
			SizeBytes: 3,
		},
	}, nil
}

func (s *testServer) FetchDirectory(ctx context.Context, req *fpb.FetchDirectoryRequest) (*fpb.FetchDirectoryResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

var server = &testServer{}

// A testFunction is something we can assign to a target's PostBuildFunction; it will
// not be called but affects whether we bother trying to restore stdout or not.
type testFunction struct{}

func (f testFunction) String() string                           { return "test post-build function" }
func (f testFunction) Call(t *core.BuildTarget, o string) error { return nil }

func TestMain(m *testing.M) {
	server.Reset()
	addr := "127.0.0.1:9987"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(server.RecoverUnaryPanics), grpc.StreamInterceptor(server.RecoverStreamPanics))
	pb.RegisterCapabilitiesServer(s, server)
	pb.RegisterActionCacheServer(s, server)
	pb.RegisterContentAddressableStorageServer(s, server)
	bs.RegisterByteStreamServer(s, server)
	pb.RegisterExecutionServer(s, server)
	fpb.RegisterFetchServer(s, server)
	go s.Serve(lis)
	if err := os.Chdir("src/remote/test_data"); err != nil {
		log.Fatalf("Failed to chdir: %s", err)
	}
	code := m.Run()
	s.Stop()
	os.Exit(code)
}
