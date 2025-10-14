// Package audit supports producing audit logs when the --audit_log_dir flag is set.
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

type auditLog struct {
	enabled                   bool
	pleaseInvocationAuditFile *auditFile
	remoteFilesAuditFile      *auditFile
	buildCommandsAuditFile    *auditFile
}

var globalAuditLog auditLog

type auditFile struct {
	*os.File
	sync.Mutex
}

func (a *auditFile) closeAuditFile() {
	if err := a.Close(); err != nil {
		log.Errorf("Unable to close file %s: %s", a.Name(), err)
	}
}

const pleaseInvocationFilename = "please_invocation.jsonl"
const remoteFilesFilename = "remote_files.jsonl"
const buildCommandsFilename = "build_commands.jsonl"

const nanoTimeFormat = "20060102_150405.999999999" // we need to use a decimal to enforce nanosecond precision

// InitAuditLogging initialises the audit logging directory and logging files, and writes
// the Please invocation to the Please invocation audit file on startup.
func InitAuditLogging(auditLogDir string) {
	ts := strings.ReplaceAll(time.Now().Format(nanoTimeFormat), ".", "_")
	globalAuditLog.enabled = true
	openAuditFiles(auditLogDir, ts)
	writePleaseInvocation()
}

func openAuditFiles(baseDir string, ts string) {
	dir := filepath.Join(baseDir, ts)
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("Unable to create audit directory %s: %s", dir, err)
	}
	globalAuditLog.pleaseInvocationAuditFile = openAuditFile(dir, pleaseInvocationFilename)
	globalAuditLog.remoteFilesAuditFile = openAuditFile(dir, remoteFilesFilename)
	globalAuditLog.buildCommandsAuditFile = openAuditFile(dir, buildCommandsFilename)
}

func openAuditFile(dir string, filename string) *auditFile {
	fullpath := filepath.Join(dir, filename)
	file, err := fs.OpenDirFile(fullpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Unable to open %s: %s", fullpath, err)
	}
	return &auditFile{File: file}
}

// Shutdown closes all the audit files
func Shutdown() {
	if !globalAuditLog.enabled {
		return
	}
	globalAuditLog.pleaseInvocationAuditFile.closeAuditFile()
	globalAuditLog.remoteFilesAuditFile.closeAuditFile()
	globalAuditLog.buildCommandsAuditFile.closeAuditFile()
}

// WriteBuildCommand writes the build label, environment and build command
// as a line of json to the build command audit file
func WriteBuildCommand(buildLabel string, env []string, command string) {
	if !globalAuditLog.enabled {
		return
	}
	globalAuditLog.buildCommandsAuditFile.writeObjectToFile(struct {
		// BuildLabel is the string representation of the build target of the build command.
		BuildLabel string `json:"build_label"`
		// Environment is the shell environment variables used in the build command.
		Environment []string `json:"environment"`
		// Command is the actual command used to build a target within the build command.
		Command string `json:"command"`
	}{
		BuildLabel:  buildLabel,
		Environment: env,
		Command:     command,
	})
}

// WriteRemoteFile writes the build label, source url, whether the attempt was successful
// and any error message (which may be an empty string) as a line of json to
// the remote files audit file given an attempt to fetch a remote file.
func WriteRemoteFile(buildLabel string, url string, success bool, errorMsg string) {
	if !globalAuditLog.enabled {
		return
	}
	globalAuditLog.remoteFilesAuditFile.writeObjectToFile(struct {
		// BuildLabel is the string representation of the build target of the remote file.
		BuildLabel string `json:"build_label"`
		// URL is the url of the source of the remote file.
		URL string `json:"url"`
		// Success is true if the attempt to fetch the remote file was successful, false otherwise.
		Success bool `json:"success"`
		// ErrorMsg is the error message returned in the event the attempt to fetch the remote file
		// was unsuccessful. It will be empty if Success is true.
		ErrorMsg string `json:"error_message"`
	}{
		BuildLabel: buildLabel,
		URL:        url,
		Success:    success,
		ErrorMsg:   errorMsg,
	})
}

// writePleaseInvocation writes the args, environment, current working directory
// and timestamp of any Please invocation to the Please invocation audit file
func writePleaseInvocation() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Unable to get cwd: %s", err)
		cwd = "ERROR"
	}

	globalAuditLog.pleaseInvocationAuditFile.writeObjectToFile(struct {
		// Args are the command line args used when invoking Please.
		Args []string `json:"args"`
		// Environment contains the environment variables set when Please was invoked, in the format key=value.
		Environment []string `json:"environment"`
		// Cwd is the current working directory when Please was invoked.
		Cwd string `json:"cwd"`
		// Timestamp is the timestamp when Please was invoked.
		Timestamp time.Time `json:"timestamp"`
	}{
		Args:        os.Args,
		Environment: os.Environ(),
		Cwd:         cwd,
		Timestamp:   time.Now(),
	})
}

func (a *auditFile) writeObjectToFile(obj any) {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("Unable to marshal obj %s to json: %s", obj, err)
		return
	}
	data = append(data, byte('\n')) // Audit logs are newline-delimited JSON
	a.Lock()
	defer a.Unlock()
	if _, err := a.Write(data); err != nil {
		log.Errorf("Unable to write object to audit file: %s", err)
	}
}
