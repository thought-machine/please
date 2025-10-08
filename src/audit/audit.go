package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

type auditLog struct {
	enabled                   bool
	baseDir                   string
	uuid                      string
	pleaseInvocationAuditFile *auditFile
	remoteFilesAuditFile      *auditFile
	buildCommandsAuditFile    *auditFile
}

func (a *auditLog) getDir() string {
	return filepath.Join(a.baseDir, a.uuid)
}

var globalAuditLog auditLog

type auditFile struct {
	*os.File
	sync.Mutex
}

const pleaseInvocationAuditFilename = "please_invocation_audit_file.jsonl"
const remoteFilesAuditFilename = "remote_files_audit_file.jsonl"
const buildCommandsAuditFilename = "build_commands_audit_file.jsonl"

func InitAuditLogging(auditLogDir string) {
	id, err := uuid.NewRandom()
	if err != nil || id.String() == "" {
		log.Fatalf("Unable to create uuid for subdir within %s: %s", auditLogDir, err)
	}
	globalAuditLog.enabled = true
	globalAuditLog.baseDir = auditLogDir
	globalAuditLog.uuid = id.String()
	openAuditFiles()
	writePleaseInvocation()
}

func openAuditFiles() {
	if !globalAuditLog.enabled {
		return
	}
	dir := globalAuditLog.getDir()
	if err := os.MkdirAll(dir, fs.DirPermissions); err != nil {
		log.Fatalf("Unable to create audit directory %s: %s", dir, err)
	}
	globalAuditLog.pleaseInvocationAuditFile = openAuditFile(pleaseInvocationAuditFilename)
	globalAuditLog.remoteFilesAuditFile = openAuditFile(remoteFilesAuditFilename)
	globalAuditLog.buildCommandsAuditFile = openAuditFile(buildCommandsAuditFilename)
}

func openAuditFile(filename string) *auditFile {
	fullpath := filepath.Join(globalAuditLog.getDir(), filename)
	file, err := fs.OpenDirFile(fullpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Unable to open %s: %s", fullpath, err)
	}
	return &auditFile{File: file}
}

func Shutdown() {
	if !globalAuditLog.enabled {
		return
	}
	closeAuditFile(globalAuditLog.pleaseInvocationAuditFile)
	closeAuditFile(globalAuditLog.remoteFilesAuditFile)
	closeAuditFile(globalAuditLog.buildCommandsAuditFile)
}

func closeAuditFile(file *auditFile) {
	if err := file.Close(); err != nil {
		log.Errorf("Unable to close file %s: %s", file, err)
	}
}

func WriteBuildCommand(buildLabel string, env []string, command string) {
	if !globalAuditLog.enabled {
		return
	}
	writeObjectToFile(globalAuditLog.buildCommandsAuditFile, struct {
		BuildLabel string   `json:"build_label"`
		Env        []string `json:"env"`
		Command    string   `json:"command"`
	}{
		BuildLabel: buildLabel,
		Env:        env,
		Command:    command,
	})
}

func WriteFetchRemoteFile(buildLabel string, url string, success bool, errorMsg string) {
	if !globalAuditLog.enabled {
		return
	}
	writeObjectToFile(globalAuditLog.remoteFilesAuditFile, struct {
		BuildLabel string `json:"build_label"`
		Url        string `json:"url"`
		Success    bool   `json:"success"`
		ErrorMsg   string `json:"error_message"`
	}{
		BuildLabel: buildLabel,
		Url:        url,
		Success:    success,
		ErrorMsg:   errorMsg,
	})
}

func writePleaseInvocation() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Errorf("Unable to get cwd: %s", err)
		cwd = "ERROR"
	}

	writeObjectToFile(globalAuditLog.pleaseInvocationAuditFile, struct {
		Args      []string  `json:"args"`
		Envs      []string  `json:"envs"`
		Cwd       string    `json:"cwd"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Args:      os.Args,
		Envs:      os.Environ(),
		Cwd:       cwd,
		Timestamp: time.Now(),
	})
}

func writeObjectToFile(file *auditFile, obj any) {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("Unable to marshal obj %s to json: %s", obj, err)
		return
	}
	file.Lock()
	defer file.Unlock()
	if _, err := file.Write(data); err != nil {
		log.Errorf("Unable to write object to audit file: %s", err)
	}
}
