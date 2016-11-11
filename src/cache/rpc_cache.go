// +build proto

// RPC-based remote cache. Similar to HTTP but likely higher performance.
package cache

import (
	"bytes"
	"core"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	pb "cache/proto/rpc_cache"
	"cache/tools"
)

const maxErrors = 5
const replicas = 2

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
}

type cacheNode struct {
	cache     *rpcCache
	hashStart uint32
	hashEnd   uint32
}

func (cache *rpcCache) Store(target *core.BuildTarget, key []byte) {
	if cache.isConnected() && cache.Writeable {
		log.Debug("Storing %s in RPC cache...", target.Label)
		artifacts := []*pb.Artifact{}
		totalSize := 0
		for out := range cacheArtifacts(target) {
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

func (cache *rpcCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	if cache.isConnected() && cache.Writeable {
		log.Debug("Storing %s : %s in RPC cache...", target.Label, file)
		artifacts, totalSize, err := cache.loadArtifacts(target, file)
		if err != nil {
			log.Warning("RPC cache failed to load artifact %s: %s", file, err)
			cache.error()
			return
		}
		if totalSize > cache.maxMsgSize {
			log.Info("Artifact %s for %s exceeds maximum message size of %s bytes", file, target.Label, cache.maxMsgSize)
			return
		}
		cache.sendArtifacts(target, key, artifacts)
	}
}

func (cache *rpcCache) loadArtifacts(target *core.BuildTarget, file string) ([]*pb.Artifact, int, error) {
	artifacts := []*pb.Artifact{}
	outDir := target.OutDir()
	root := path.Join(outDir, file)
	totalSize := 1000 // Allow a little space for encoding overhead.
	err := filepath.Walk(root, func(name string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if !info.IsDir() {
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
	req := pb.StoreRequest{Artifacts: artifacts, Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH}
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	resp, err := cache.client.Store(ctx, &req)
	if err != nil {
		log.Warning("Error communicating with RPC cache server: %s", err)
		cache.error()
	} else if !resp.Success {
		log.Warning("Failed to store artifacts in RPC cache for %s", target.Label)
	}
}

func (cache *rpcCache) Retrieve(target *core.BuildTarget, key []byte) bool {
	if !cache.isConnected() {
		return false
	}
	req := pb.RetrieveRequest{Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH}
	for out := range cacheArtifacts(target) {
		artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name, File: out}
		req.Artifacts = append(req.Artifacts, &artifact)
	}
	// We can't tell from here if retrieval has been successful for a target with no outputs.
	// This is kind of weird but not actually disallowed, and we already have a test case for it,
	// so might as well try to get it right here.
	if len(req.Artifacts) == 0 {
		return false
	}
	return cache.retrieveArtifacts(target, &req, true)
}

func (cache *rpcCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	if !cache.isConnected() {
		return false
	}
	artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name, File: file}
	artifacts := []*pb.Artifact{&artifact}
	req := pb.RetrieveRequest{Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH, Artifacts: artifacts}
	return cache.retrieveArtifacts(target, &req, false)
}

func (cache *rpcCache) retrieveArtifacts(target *core.BuildTarget, req *pb.RetrieveRequest, remove bool) bool {
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	response, err := cache.client.Retrieve(ctx, req)
	if err != nil {
		log.Warning("Failed to retrieve artifacts for %s", target.Label)
		cache.error()
		return false
	} else if !response.Success {
		// Quiet, this is almost certainly just a 'not found'
		log.Debug("Couldn't retrieve artifacts for %s [key %s] from RPC cache", target.Label, base64.RawURLEncoding.EncodeToString(req.Hash))
		return false
	}
	// Remove any existing outputs first; this is important for cases where the output is a
	// directory, because we get back individual artifacts, and we need to make sure that
	// only the retrieved artifacts are present in the output.
	if remove {
		for _, out := range target.Outputs() {
			out := path.Join(target.OutDir(), out)
			if err := os.RemoveAll(out); err != nil {
				log.Error("Failed to remove artifact %s: %s", out, err)
				return false
			}
		}
	}
	for _, artifact := range response.Artifacts {
		if !cache.writeFile(target, artifact.File, artifact.Body) {
			return false
		}
	}
	// Sanity check: if we don't get anything back, assume it probably wasn't really a success.
	return len(response.Artifacts) > 0
}

func (cache *rpcCache) writeFile(target *core.BuildTarget, file string, body []byte) bool {
	out := path.Join(target.OutDir(), file)
	if err := os.MkdirAll(path.Dir(out), core.DirPermissions); err != nil {
		log.Warning("Failed to create directory for artifacts: %s", err)
		return false
	}
	if err := core.WriteFile(bytes.NewReader(body), out, fileMode(target)); err != nil {
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
		response, err := cache.client.Delete(context.Background(), &req)
		if err != nil || !response.Success {
			log.Errorf("Failed to remove %s from RPC cache", target.Label)
		}
	}
}

func (cache *rpcCache) Shutdown() {}

func (cache *rpcCache) connect(config *core.Configuration) {
	// Change grpc to log using our implementation
	grpclog.SetLogger(&grpcLogMabob{})
	log.Info("Connecting to RPC cache at %s", config.Cache.RpcUrl)
	opts := []grpc.DialOption{grpc.WithTimeout(cache.timeout)}
	if config.Cache.RpcPublicKey != "" || config.Cache.RpcCACert != "" || config.Cache.RpcSecure {
		auth, err := loadAuth(config.Cache.RpcCACert, config.Cache.RpcPublicKey, config.Cache.RpcPrivateKey)
		if err != nil {
			log.Warning("Failed to load RPC cache auth keys: %s", err)
			return
		}
		opts = append(opts, auth)
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	connection, err := grpc.Dial(config.Cache.RpcUrl, opts...)
	if err != nil {
		cache.Connecting = false
		log.Warning("Failed to connect to RPC cache: %s", err)
		return
	}
	// Message the server to get its cluster topology.
	client := pb.NewRpcCacheClient(connection)
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	resp, err := client.ListNodes(ctx, &pb.ListRequest{})
	// For compatibility with older servers, handle an error code of Unimplemented and treat
	// as an unclustered server (because of course they can't be clustered).
	if err != nil && grpc.Code(err) != codes.Unimplemented {
		cache.Connecting = false
		log.Warning("Failed to contact RPC cache: %s", err)
		return
	} else if resp == nil || len(resp.Nodes) == 0 {
		// Server is not clustered, just use this one directly.
		cache.client = client
		cache.Connected = true
		cache.Connecting = false
		log.Info("RPC cache connected after %0.2fs", time.Since(cache.startTime).Seconds())
		return
	}
	// If we get here, we are connected and the cache is clustered.
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

// getInstance grabs a cache instance for the given hash.
func (cache *rpcCache) getInstance(hash []byte, replica int) *rpcCache {
	var h uint32
	if replica == 0 {
		h = tools.Hash(hash)
	} else {
		h = tools.AlternateHash(hash)
	}
	for _, n := range cache.nodes {
		if h >= n.hashStart && h < n.hashEnd {
			return n.cache
		}
	}
	log.Warning("No RPC cache client available for %d", h)
	return nil
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

func newRpcCache(config *core.Configuration) (*rpcCache, error) {
	cache := &rpcCache{
		Writeable:  config.Cache.RpcWriteable,
		Connecting: true,
		timeout:    time.Duration(config.Cache.RpcTimeout),
		startTime:  time.Now(),
		maxMsgSize: int(config.Cache.RpcMaxMsgSize),
	}
	go cache.connect(config)
	return cache, nil
}

// grpcLogMabob is an implementation of grpc's logging interface using our backend.
type grpcLogMabob struct{}

func (g *grpcLogMabob) Fatal(args ...interface{})                 { log.Fatal(args...) }
func (g *grpcLogMabob) Fatalf(format string, args ...interface{}) { log.Fatalf(format, args...) }
func (g *grpcLogMabob) Fatalln(args ...interface{})               { log.Fatal(args...) }
func (g *grpcLogMabob) Print(args ...interface{})                 { log.Warning("%s", args) }
func (g *grpcLogMabob) Printf(format string, args ...interface{}) { log.Warning(format, args...) }
func (g *grpcLogMabob) Println(args ...interface{})               { log.Warning("%s", args) }

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
