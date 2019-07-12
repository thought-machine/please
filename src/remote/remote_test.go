package remote

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/bazelbuild/remote-apis/build/bazel/semver"
	"github.com/stretchr/testify/assert"
	bs "google.golang.org/genproto/googleapis/bytestream"
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
	// The hash is arbitrary as far as this package is concerned.
	key, _ := hex.DecodeString("2dd283abb148ebabcd894b306e3d86d0390c82a7")
	err := c.Store(target, key, []string{"plz-out/gen/package/out1.txt"})
	assert.NoError(t, err)
	// Remove the old file, but remember its contents so we can compare later.
	contents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	err = os.Remove("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	// Now retrieve back the output of this thing.
	err = c.Retrieve(target, key)
	assert.NoError(t, err)
	cachedContents, err := ioutil.ReadFile("plz-out/gen/package/out1.txt")
	assert.NoError(t, err)
	assert.Equal(t, contents, cachedContents)
}

func newClient() *Client {
	config := core.DefaultConfiguration()
	config.Build.Path = []string{"/usr/local/bin", "/usr/bin", "/bin"}
	// Can't use NewDefaultBuildState since we need to modify the config first.
	state := core.NewBuildState(1, nil, 4, config)
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
	b, present := s.blobs[req.ResourceName]
	if !present {
		return status.Errorf(codes.NotFound, "bytestream %s not found", req.ResourceName)
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
	b := s.bytestreams[name]
	for {
		if req.WriteOffset != int64(len(b)) {
			return status.Errorf(codes.InvalidArgument, "incorrect WriteOffset (was %d, should be %d)", req.WriteOffset, len(b))
		}
		b = append(b, req.Data...)
		if req.FinishWrite {
			s.blobs[name] = b
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

var server = &testServer{}

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
	go s.Serve(lis)
	if err := os.Chdir("src/remote/test_data"); err != nil {
		log.Fatalf("Failed to chdir: %s", err)
	}
	code := m.Run()
	s.Stop()
	os.Exit(code)
}
