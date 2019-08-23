// +build !bootstrap

// RPC-based remote cache. Similar to HTTP but likely higher performance.

package cache

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/thought-machine/please/src/core"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // Registers the gzip compressor at init
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"gopkg.in/op/go-logging.v1"

	pb "github.com/thought-machine/please/src/cache/proto/rpc_cache"
	"github.com/thought-machine/please/src/cache/tools"
	"github.com/thought-machine/please/src/fs"
)

func init() {
	// Change grpc to log using our implementation
	grpclog.SetLoggerV2(&grpcLogMabob{})
}

const maxErrors = 5

// We use zeroKey in cases where we need to supply a hash but it actually doesn't matter.
var zeroKey = []byte{0, 0, 0, 0}

type rpcCache struct {
	client     pb.RpcCacheClient
	Writeable  bool
	Connected  bool
	Connecting bool
	OSName     string
	numErrors  int32
	timeout    time.Duration
	startTime  time.Time
	maxMsgSize int
	nodes      []cacheNode
	hostname   string
	compressor grpc.CallOption
}

type cacheNode struct {
	cache     *rpcCache
	hashStart uint32
	hashEnd   uint32
}

func (cache *rpcCache) Store(target *core.BuildTarget, key []byte, metadata *core.BuildMetadata, files []string) {
	if cache.isConnected() && cache.Writeable {
		log.Debug("Storing %s in RPC cache...", target.Label)
		artifacts := []*pb.Artifact{}
		totalSize := 0
		if needsPostBuildFile(target) {
			files = append(files, target.PostBuildOutputFileName())
		}
		for _, out := range files {
			artifacts2, size, err := cache.loadArtifacts(target, out)
			if err != nil {
				log.Warning("RPC cache failed to load artifact %s: %s", out, err)
				cache.error()
				return
			}
			totalSize += size
			artifacts = append(artifacts, artifacts2...)
		}
		if totalSize > cache.maxMsgSize {
			log.Info("Artifacts for %s exceed maximum message size of %s bytes", target.Label, cache.maxMsgSize)
			return
		}
		cache.sendArtifacts(target, key, artifacts)
	}
}

func (cache *rpcCache) loadArtifacts(target *core.BuildTarget, file string) ([]*pb.Artifact, int, error) {
	outDir := target.OutDir()
	root := path.Join(outDir, file)
	totalSize := 1000 // Allow a little space for encoding overhead.
	if info, err := os.Lstat(root); err == nil && (info.Mode()&os.ModeSymlink) != 0 {
		// Artifact is a symlink.
		dest, err := os.Readlink(root)
		if err != nil {
			return nil, totalSize, err
		}
		return []*pb.Artifact{{
			Package: target.Label.PackageName,
			Target:  target.Label.Name,
			File:    file,
			Symlink: dest,
		}}, totalSize, nil
	}
	artifacts := []*pb.Artifact{}
	err := fs.Walk(root, func(name string, isDir bool) error {
		if !isDir {
			content, err := ioutil.ReadFile(name)
			if err != nil {
				return err
			}
			artifacts = append(artifacts, &pb.Artifact{
				Package: target.Label.PackageName,
				Target:  target.Label.Name,
				File:    name[len(outDir)+1:],
				Body:    content,
			})
			totalSize += len(content)
		}
		return nil
	})
	return artifacts, totalSize, err
}

func (cache *rpcCache) sendArtifacts(target *core.BuildTarget, key []byte, artifacts []*pb.Artifact) {
	req := pb.StoreRequest{
		Artifacts: artifacts,
		Hash:      key,
		Os:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		Hostname:  cache.hostname,
	}
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	cache.runRPC(key, func(cache *rpcCache) (bool, []*pb.Artifact) {
		_, err := cache.client.Store(ctx, &req, cache.compressor)
		if err != nil {
			log.Warning("Error communicating with RPC cache server: %s", err)
			cache.error()
		}
		return err != nil, nil
	})
}

func (cache *rpcCache) Retrieve(target *core.BuildTarget, key []byte, files []string) *core.BuildMetadata {
	if !cache.isConnected() {
		return nil
	}
	req := pb.RetrieveRequest{Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH}
	for _, out := range files {
		req.Artifacts = append(req.Artifacts, &pb.Artifact{
			Package: target.Label.PackageName,
			Target:  target.Label.Name,
			File:    out,
		})
	}
	// We can't tell from here if retrieval has been successful for a target with no outputs.
	// This is kind of weird but not actually disallowed, and we already have a test case for it,
	// so might as well try to get it right here.
	if len(req.Artifacts) == 0 {
		return nil
	}
	if !cache.retrieveArtifacts(target, &req, true, files) {
		return nil
	} else if needsPostBuildFile(target) {
		return loadPostBuildFile(target)
	}
	return &core.BuildMetadata{}
}

func (cache *rpcCache) retrieveArtifacts(target *core.BuildTarget, req *pb.RetrieveRequest, remove bool, files []string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	success, artifacts := cache.runRPC(req.Hash, func(cache *rpcCache) (bool, []*pb.Artifact) {
		response, err := cache.client.Retrieve(ctx, req, cache.compressor)
		if err != nil {
			log.Warning("Failed to retrieve artifacts for %s: %s", target.Label, err)
			cache.error()
			return false, nil
		} else if !response.Success {
			// Quiet, this is almost certainly just a 'not found'
			log.Debug("Couldn't retrieve artifacts for %s [key %s] from RPC cache", target.Label, base64.RawURLEncoding.EncodeToString(req.Hash))
		}
		// This always counts as "success" in this context, i.e. do not bother retrying on the
		// alternate if we were told that the artifact is not there.
		return true, response.Artifacts
	})
	if !success {
		return false
	}
	// Remove any existing outputs first; this is important for cases where the output is a
	// directory, because we get back individual artifacts, and we need to make sure that
	// only the retrieved artifacts are present in the output.
	if remove {
		for _, out := range files {
			out := path.Join(target.OutDir(), out)
			if err := os.RemoveAll(out); err != nil {
				log.Error("Failed to remove artifact %s: %s", out, err)
				return false
			}
		}
	}
	for _, artifact := range artifacts {
		if !cache.writeFile(target, artifact.File, artifact.Body, artifact.Symlink) {
			return false
		}
	}
	// Sanity check: if we don't get anything back, assume it probably wasn't really a success.
	return len(artifacts) > 0
}

func (cache *rpcCache) writeFile(target *core.BuildTarget, file string, body []byte, symlink string) bool {
	out := path.Join(target.OutDir(), file)
	if err := os.MkdirAll(path.Dir(out), core.DirPermissions); err != nil {
		log.Warning("Failed to create directory for artifacts: %s", err)
		return false
	}
	if symlink != "" {
		if err := os.Symlink(symlink, out); err != nil {
			log.Warning("RPC cache failed to create symlink %s", err)
			return false
		}
	} else if err := fs.WriteFile(bytes.NewReader(body), out, target.OutMode()); err != nil {
		log.Warning("RPC cache failed to write file %s", err)
		return false
	}
	log.Debug("Retrieved %s - %s from RPC cache", target.Label, file)
	return true
}

func (cache *rpcCache) Clean(target *core.BuildTarget) {
	if cache.isConnected() && cache.Writeable {
		req := pb.DeleteRequest{Os: runtime.GOOS, Arch: runtime.GOARCH}
		artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name}
		req.Artifacts = []*pb.Artifact{&artifact}
		cache.runRPC(zeroKey, func(cache *rpcCache) (bool, []*pb.Artifact) {
			response, err := cache.client.Delete(context.Background(), &req, cache.compressor)
			if err != nil || !response.Success {
				log.Errorf("Failed to remove %s from RPC cache", target.Label)
				return false, nil
			}
			return true, nil
		})
	}
}

func (cache *rpcCache) CleanAll() {
	if !cache.isConnected() {
		log.Error("RPC cache is not connected, cannot clean")
	} else if !cache.Writeable {
		log.Error("RPC cache is not writable, will not clean")
	} else {
		log.Debug("Cleaning entire RPC cache")
		req := pb.DeleteRequest{Everything: true}
		cache.runRPC(zeroKey, func(cache *rpcCache) (bool, []*pb.Artifact) {
			if response, err := cache.client.Delete(context.Background(), &req, cache.compressor); err != nil || !response.Success {
				log.Errorf("Failed to clean RPC cache: %s", err)
				return false, nil
			}
			return true, nil
		})
	}
}

func (cache *rpcCache) Shutdown() {}

func (cache *rpcCache) connect(url string, config *core.Configuration, isSubnode bool) {
	log.Info("Connecting to RPC cache at %s", url)
	opts := []grpc.DialOption{
		grpc.WithTimeout(cache.timeout),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(cache.maxMsgSize), grpc.MaxCallSendMsgSize(cache.maxMsgSize)),
	}
	if config.Cache.RPCPublicKey != "" || config.Cache.RPCCACert != "" || config.Cache.RPCSecure {
		auth, err := loadAuth(config.Cache.RPCCACert, config.Cache.RPCPublicKey, config.Cache.RPCPrivateKey)
		if err != nil {
			log.Warning("Failed to load RPC cache auth keys: %s", err)
			return
		}
		opts = append(opts, auth)
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	connection, err := grpc.Dial(url, opts...)
	if err != nil {
		cache.Connecting = false
		log.Warning("Failed to connect to RPC cache: %s", err)
		return
	}
	// Stash hostname for later
	if hostname, err := os.Hostname(); err == nil {
		cache.hostname = hostname
	}
	// Message the server to get its cluster topology.
	client := pb.NewRpcCacheClient(connection)
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	resp, err := client.ListNodes(ctx, &pb.ListRequest{})
	// For compatibility with older servers, handle an error code of Unimplemented and treat
	// as an unclustered server (because of course they can't be clustered).
	if err != nil && status.Code(err) != codes.Unimplemented {
		cache.Connecting = false
		log.Warning("Failed to contact RPC cache: %s", err)
		return
	} else if isSubnode || resp == nil || len(resp.Nodes) == 0 {
		// Server is not clustered, just use this one directly.
		// Or we're one of the sub-nodes and we are meant to connect directly.
		cache.client = client
		cache.Connected = true
		cache.Connecting = false
		log.Info("RPC cache connected after %0.2fs", time.Since(cache.startTime).Seconds())
		return
	}
	// If we get here, we are connected and the cache is clustered.
	cache.nodes = make([]cacheNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		subCache, _ := newRPCCacheInternal(n.Address, config, true)
		cache.nodes[i] = cacheNode{
			cache:     subCache,
			hashStart: n.HashBegin,
			hashEnd:   n.HashEnd,
		}
	}
	// We are now connected, the children aren't necessarily yet but that won't matter.
	cache.Connected = true
	cache.Connecting = false
	log.Info("Top-level RPC cache connected after %0.2fs with %d known nodes", time.Since(cache.startTime).Seconds(), len(resp.Nodes))
}

// isConnected checks if the cache is connected. If it's still trying to connect it allows a
// very brief wait to give it a chance to come online.
func (cache *rpcCache) isConnected() bool {
	if cache.Connected {
		return true
	} else if !cache.Connecting {
		return false
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	for i := 0; i < 5 && cache.Connecting; i++ {
		<-ticker.C
	}
	ticker.Stop()
	return cache.Connected
}

// runRPC runs one RPC for a cache, with optional fallback to a replica on RPC failure
// (but not if the RPC completes unsuccessfully).
func (cache *rpcCache) runRPC(hash []byte, f func(*rpcCache) (bool, []*pb.Artifact)) (bool, []*pb.Artifact) {
	if len(cache.nodes) == 0 {
		// No clustering, just call it directly.
		return f(cache)
	}
	try := func(hash uint32) (bool, []*pb.Artifact) {
		for _, n := range cache.nodes {
			if hash >= n.hashStart && hash < n.hashEnd {
				if !n.cache.isConnected() {
					return false, nil
				}
				return f(n.cache)
			}
		}
		log.Warning("No RPC cache client available for %d", hash)
		return false, nil
	}
	h := tools.Hash(hash)
	success, artifacts := try(h)
	if success {
		return success, artifacts
	}
	log.Info("Initial replica failed for %d, will retry on the alternate", h)
	return try(tools.AlternateHash(hash))
}

// error increments the error counter on the cache, and disables it if it gets too high.
// Note that after this it won't reconnect; we could try that but it probably isn't worth it
// (it's unlikely to restart in time if it's got a nontrivial set of artifacts to scan) and
// the user has probably been pestered by enough messages already.
func (cache *rpcCache) error() {
	if atomic.AddInt32(&cache.numErrors, 1) >= maxErrors && cache.Connected {
		log.Warning("Disabling RPC cache, looks like the connection has been lost")
		cache.Connected = false
	}
}

func newRPCCache(config *core.Configuration) (*rpcCache, error) {
	return newRPCCacheInternal(config.Cache.RPCURL.String(), config, false)
}

func newRPCCacheInternal(url string, config *core.Configuration, isSubnode bool) (*rpcCache, error) {
	cache := &rpcCache{
		Writeable:  config.Cache.RPCWriteable,
		Connecting: true,
		timeout:    time.Duration(config.Cache.RPCTimeout),
		startTime:  time.Now(),
		maxMsgSize: int(config.Cache.RPCMaxMsgSize),
		compressor: grpc.UseCompressor("gzip"),
	}
	go cache.connect(url, config, isSubnode)
	return cache, nil
}

// grpcLogMabob is an implementation of grpc's logging interface using our backend.
type grpcLogMabob struct{}

func (g *grpcLogMabob) Info(args ...interface{})                    { log.Info("%s", args) }
func (g *grpcLogMabob) Infof(format string, args ...interface{})    { log.Info(format, args...) }
func (g *grpcLogMabob) Infoln(args ...interface{})                  { log.Info("%s", args) }
func (g *grpcLogMabob) Warning(args ...interface{})                 { log.Warning("%s", args) }
func (g *grpcLogMabob) Warningf(format string, args ...interface{}) { log.Warning(format, args...) }
func (g *grpcLogMabob) Warningln(args ...interface{})               { log.Warning("%s", args) }
func (g *grpcLogMabob) Error(args ...interface{})                   { log.Error("", args...) }
func (g *grpcLogMabob) Errorf(format string, args ...interface{})   { log.Errorf(format, args...) }
func (g *grpcLogMabob) Errorln(args ...interface{})                 { log.Error("", args...) }
func (g *grpcLogMabob) Fatal(args ...interface{})                   { log.Fatal(args...) }
func (g *grpcLogMabob) Fatalf(format string, args ...interface{})   { log.Fatalf(format, args...) }
func (g *grpcLogMabob) Fatalln(args ...interface{})                 { log.Fatal(args...) }
func (g *grpcLogMabob) V(l int) bool                                { return log.IsEnabledFor(logging.Level(l)) }

// loadAuth loads authentication credentials from a given pair of public / private key files.
func loadAuth(caCert, publicKey, privateKey string) (grpc.DialOption, error) {
	config := tls.Config{}
	if publicKey != "" {
		log.Debug("Loading client certificate from %s, key %s", publicKey, privateKey)
		cert, err := tls.LoadX509KeyPair(publicKey, privateKey)
		if err != nil {
			return nil, err
		}
		config.Certificates = []tls.Certificate{cert}
	}
	if caCert != "" {
		log.Debug("Reading CA cert file from %s", caCert)
		cert, err := ioutil.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		config.RootCAs = x509.NewCertPool()
		if !config.RootCAs.AppendCertsFromPEM(cert) {
			return nil, fmt.Errorf("Failed to add any PEM certificates from %s", caCert)
		}
	}
	return grpc.WithTransportCredentials(credentials.NewTLS(&config)), nil
}
