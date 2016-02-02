// +build proto

// RPC-based remote cache. Similar to HTTP but likely higher performance.
package cache

import (
	"bytes"
	"core"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "cache/proto/rpc_cache"
)

type rpcCache struct {
	connection *grpc.ClientConn
	client     pb.RpcCacheClient
	Writeable  bool
	OSName     string
}

func (cache *rpcCache) Store(target *core.BuildTarget, key []byte) {
	if cache.Writeable {
		artifacts := []*pb.Artifact{}
		for out := range cacheArtifacts(target) {
			artifacts2, err := cache.loadArtifacts(target, out)
			if err != nil {
				log.Warning("RPC cache failed to load artifact %s: %s", out, err)
				return
			}
			artifacts = append(artifacts, artifacts2...)
		}
		cache.sendArtifacts(target, key, artifacts)
	}
}

func (cache *rpcCache) StoreExtra(target *core.BuildTarget, key []byte, file string) {
	if cache.Writeable {
		artifacts, err := cache.loadArtifacts(target, file)
		if err != nil {
			log.Warning("RPC cache failed to load artifact %s: %s", file, err)
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
	resp, err := cache.client.Store(context.Background(), &req)
	if err != nil {
		log.Warning("Error communicating with RPC cache server: %s", err)
	} else if !resp.Success {
		log.Warning("Failed to store artifacts in RPC cache for %s", target.Label)
	}
}

func (cache *rpcCache) Retrieve(target *core.BuildTarget, key []byte) bool {
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
	artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name, File: file}
	artifacts := []*pb.Artifact{&artifact}
	req := pb.RetrieveRequest{Hash: key, Os: runtime.GOOS, Arch: runtime.GOARCH, Artifacts: artifacts}
	return cache.retrieveArtifacts(target, &req)
}

func (cache *rpcCache) retrieveArtifacts(target *core.BuildTarget, req *pb.RetrieveRequest) bool {
	response, err := cache.client.Retrieve(context.Background(), req)
	if err != nil {
		log.Warning("Failed to retrieve artifacts for %s", target.Label)
		return false
	} else if !response.Success {
		// Quiet, this is almost certainly just a 'not found'
		log.Debug("Couldn't retrieve artifacts for %s", target.Label)
		return false
	}
	for _, artifact := range response.Artifacts {
		if !cache.writeFile(target, artifact.File, artifact.Body) {
			return false
		}
	}
	return true
}

func (cache *rpcCache) writeFile(target *core.BuildTarget, file string, body []byte) bool {
	out := path.Join(target.OutDir(), file)
	if err := core.WriteFile(bytes.NewReader(body), out, fileMode(target)); err != nil {
		log.Warning("RPC cache failed to write file %s", err)
		return false
	}
	log.Debug("Retrieved %s from RPC cache", target.Label)
	return true
}

func (cache *rpcCache) Clean(target *core.BuildTarget) {
	if cache.Writeable {
		req := pb.DeleteRequest{Os: runtime.GOOS, Arch: runtime.GOARCH}
		artifact := pb.Artifact{Package: target.Label.PackageName, Target: target.Label.Name}
		req.Artifacts = []*pb.Artifact{&artifact}
		response, err := cache.client.Delete(context.Background(), &req)
		if err != nil || !response.Success {
			log.Error("Failed to remove %s from RPC cache", target.Label)
		}
	}
}

func newRpcCache(config core.Configuration) (*rpcCache, error) {
	// Change grpc to log using our implementation
	grpclog.SetLogger(&grpcLogMabob{})
	log.Info("Connecting to RPC cache at %s", config.Cache.RpcUrl)
	// TODO(pebers): Add support for communicating over https.
	connection, err := grpc.Dial(config.Cache.RpcUrl,
		grpc.WithBlock(), grpc.WithInsecure(), grpc.WithTimeout(time.Duration(config.Cache.RpcTimeout)*time.Second))
	if err != nil {
		return nil, err
	}
	return &rpcCache{
		connection: connection,
		client:     pb.NewRpcCacheClient(connection),
		Writeable:  config.Cache.RpcWriteable,
	}, nil
}

// grpcLogMabob is an implementation of grpc's logging interface using our backend.
type grpcLogMabob struct{}

func (g *grpcLogMabob) Fatal(args ...interface{})                 { log.Fatal(args...) }
func (g *grpcLogMabob) Fatalf(format string, args ...interface{}) { log.Fatalf(format, args...) }
func (g *grpcLogMabob) Fatalln(args ...interface{})               { log.Fatal(args...) }
func (g *grpcLogMabob) Print(args ...interface{})                 { log.Info("%s", args) }
func (g *grpcLogMabob) Printf(format string, args ...interface{}) { log.Info(format, args...) }
func (g *grpcLogMabob) Println(args ...interface{})               { log.Info("%s", args) }
