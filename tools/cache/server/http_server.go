package server

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("server")

type httpServer struct {
	cache *Cache
}

// The pingHandler will return a 200 Accepted status
// This handler will handle ping endpoint requests, in order to confirm whether the server can be accessed
func (s *httpServer) pingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server connection established successfully.")
}

// The getHandler function handles the GET endpoint for the artifact path.
// It calls the RetrieveArtifact function, and then either returns the found artifact, or logs the error
// returned by RetrieveArtifact.
func (s *httpServer) getHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("GET %s", r.URL.Path)
	artifactPath := strings.TrimPrefix(r.URL.Path, "/artifact/")

	arts, err := s.cache.RetrieveArtifact(artifactPath)
	if err != nil && os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		log.Debug("%s doesn't exist in http cache", artifactPath)
		return
	} else if err != nil {
		log.Errorf("Failed to retrieve artifact %s: %s", artifactPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// In order to handle directories we use multipart encoding.
	// Note that we don't bother on the upload because the client knows all the parts and can
	// send individually; here they don't know what they'll need to expect.
	// We could use it for upload too which might be faster and would be more symmetric, but
	// multipart is a bit fiddly so for now we're not bothering.
	mw := multipart.NewWriter(w)
	defer mw.Close()
	w.Header().Set("Content-Type", mw.FormDataContentType())
	for _, art := range arts {
		if part, err := mw.CreateFormFile(art.File, art.File); err != nil {
			log.Errorf("Failed to create form file %s: %s", art.File, err)
			w.WriteHeader(http.StatusInternalServerError)
		} else if _, err := io.Copy(part, bytes.NewReader(art.Body)); err != nil {
			log.Errorf("Failed to write form file %s: %s", art.File, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// The postHandler function handles the POST endpoint for the artifact path.
// It reads the request body and sends it to the StoreArtifact function, along with the path where it should
// be stored.
// The handler will either return an error or display a message confirming the file has been created.
func (s *httpServer) postHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("POST %s", r.URL.Path)
	artifact, err := ioutil.ReadAll(r.Body)
	filePath, fileName := path.Split(strings.TrimPrefix(r.URL.Path, "/artifact"))
	if err == nil {
		// N.B. We cannot store symlinks here.
		if err := s.cache.StoreArtifact(strings.TrimPrefix(r.URL.Path, "/artifact"), artifact, ""); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Failed to store artifact %s: %s", fileName, err)
			return
		}
		absPath, _ := filepath.Abs(filePath)
		fmt.Fprintf(w, "%s was created in %s.", fileName, absPath)
		log.Notice("%s was stored in the http cache.", fileName)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Failed to store artifact %s: %s", fileName, err)
	}
}

// The deleteAllHandler function handles the DELETE endpoint for the general server path.
// It calls the DeleteAllArtifacts function.
// The handler will either return an error or display a message confirming the files have been removed.
func (s *httpServer) deleteAllHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.cache.DeleteAllArtifacts(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Failed to clean http cache: %s", err)
		return
	}
	log.Notice("The http cache has been cleaned.")
	fmt.Fprintf(w, "The http cache has been cleaned.")
}

// The deleteHandler function handles the DELETE endpoint for the artifact path.
// It calls the DeleteArtifact function, sending the path of the artifact as a parameter.
// The handler will either return an error or display a message confirming the artifact has been removed.
func (s *httpServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	artifactPath := strings.TrimPrefix(r.URL.Path, "/artifact")
	if err := s.cache.DeleteArtifact(artifactPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Failed to remove %s from http cache: %s", artifactPath, err)
		return
	}
	log.Notice("%s was removed from the http cache.", artifactPath)
	fmt.Fprintf(w, "%s artifact was removed from cache.", artifactPath)
}

// BuildRouter creates a router, sets the base FileServer directory and the Handler Functions
// for each endpoint, and then returns the router.
func BuildRouter(cache *Cache) *mux.Router {
	s := &httpServer{cache: cache}
	r := mux.NewRouter()
	r.HandleFunc("/ping", s.pingHandler).Methods("GET")
	r.HandleFunc("/artifact/{os_name}/{artifact:.*}", s.getHandler).Methods("GET")
	r.HandleFunc("/artifact/{os_name}/{artifact:.*}", s.postHandler).Methods("POST")
	r.HandleFunc("/artifact/{artifact:.*}", s.deleteHandler).Methods("DELETE")
	r.HandleFunc("/", s.deleteAllHandler).Methods("DELETE")
	return r
}
