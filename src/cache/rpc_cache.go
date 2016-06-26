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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb "cache/proto/rpc_cache"
)

const maxErrors = 5

type rpcCache struct {
	connection *grpc.ClientConn
	client     pb.RpcCacheClient
	Writeable  bool
	Connected  bool
	Connecting bool
	OSName     string
	numErrors  int32
	timeout    time.Duration
	startTime  time.Time
}

func (cache *rpcCache) Store(target *core.BuildTarget, key []byte) {
	if cache.isConnected() && cache.Writeable {
		log.Debug("Storing %s in RPC cache...", target.Label)
		artifacts := []*pb.Artifact{}
		for out := range cacheArtifacts(target) {
			artifacts2, err := cache.loadArtifacts(target, out)
			if err != nil {
				log.Warning("RPC cache failed to load artifact %s: %s", out, err)
				cache.error()
				return
			}
			artifacts = append(artifacts, artifacts2...)
		}
		cache.sendArtifacts(target, key, artifacts)
	}
}

func (cache *rpcCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	if cache.isConnected() && cache.Writeable {
		log.Debug("Storing %s : %s in RPC cache...", target.Label, file)
		artifacts, err := cache.loadArtifacts(target, file)
		if err != nil {
			log.Warning("RPC cache failed to load artifact %s: %s", file, err)
			cache.error()
			return
		}
		cache.sendArtifacts(target, key, artifacts)
	}
}

func (cache *rpcCache) loadArtifacts(target *core.BuildTarget, file string) ([]*pb.Artifact, error) {
	artifacts := []*pb.Artifact{}
	outDir := target.OutDir()
	root := path.Join(outDir, file)
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
				File:    name[len(outDir)+1 : len(name)],
				Body:    content,
			})
		}
		return nil
	})
	return artifacts, err
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
	return cache.retrieveArtifacts(target, &req)
}

func (cache *rpcCache) RetrieveExtra(target *core.BuildTarget, key []byte, file string) bool {
	if !cache.isConnected() {
		return false
	}
	artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name, File: file}
	artifacts := []*pb.Artifact{&artifact}
	req := pb.RetrieveRequest{Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH, Artifacts: artifacts}
	return cache.retrieveArtifacts(target, &req)
}

func (cache *rpcCache) retrieveArtifacts(target *core.BuildTarget, req *pb.RetrieveRequest) bool {
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
	// Note that we have to actually send it a message here to validate the connection;
	// Dial() only seems to return errors for superficial failures like syntactically invalid addresses,
	// it will return essentially immediately even if the server doesn't exist.
	healthclient := healthpb.NewHealthClient(connection)
	ctx, cancel := context.WithTimeout(context.Background(), cache.timeout)
	defer cancel()
	resp, err := healthclient.Check(ctx, &healthpb.HealthCheckRequest{Service: "plz-rpc-cache"})
	if err != nil {
		cache.Connecting = false
		log.Warning("Failed to contact RPC cache: %s", err)
	} else if resp.Status != healthpb.HealthCheckResponse_SERVING {
		cache.Connecting = false
		log.Warning("RPC cache says it is not serving (%d)", resp.Status)
	} else {
		cache.connection = connection
		cache.client = pb.NewRpcCacheClient(connection)
		cache.Connected = true
		cache.Connecting = false
		log.Info("RPC cache connected after %0.2fs", time.Since(cache.startTime).Seconds())
	}
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

// error increments the error counter on the cache, and disables it if it gets too high.
// Note that after this it won't reconnect; we could try that but it probably isn't worth it
// (it's unlikely to restart in time if it's got a nontrivial set of artifacts to scan) and
// the user has probably been pestered by enough messages already.
func (cache *rpcCache) error() {
	if atomic.AddInt32(&cache.numErrors, 1) >= maxErrors {
		log.Warning("Disabling RPC cache, looks like the connection has been lost")
		cache.Connected = false
	}
}

func newRpcCache(config *core.Configuration) (*rpcCache, error) {
	cache := &rpcCache{
		Writeable:  config.Cache.RpcWriteable,
		Connecting: true,
		timeout:    time.Duration(config.Cache.RpcTimeout) * time.Second,
		startTime:  time.Now(),
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
