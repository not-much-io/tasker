package lib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexflint/go-filemutex"
	ignore "github.com/sabhiram/go-gitignore"
	log "github.com/sirupsen/logrus"
)

// We must always start and end with the more fine grained mutex:
// - Lock process wide mutex first
// - Unlock process wide mutex last
// This is because the systemWideMutex has hard to reason about semantics.
// It is based on flock which does all kinds of "weird" things from a high level perspective.
// For example the same process can lock the same file twice.
// Flock will just allow it silently and change the lock if needed and invalidates the old lock(?).
// Either way better to just not depend on those complicated semantics.
type masterMutex struct {
	processWideMutex *sync.Mutex
	systemWideMutex  *filemutex.FileMutex
}

func NewMasterMutex(path string) (*masterMutex, error) {
	processWideMutex := &sync.Mutex{}
	systemWideMutex, err := filemutex.New(path)
	if err != nil {
		return nil, fmt.Errorf("filemutex.New: %w", err)
	}
	return &masterMutex{
		processWideMutex: processWideMutex,
		systemWideMutex:  systemWideMutex,
	}, nil
}

func (mm *masterMutex) lock() error {
	mm.processWideMutex.Lock()
	err := mm.systemWideMutex.Lock()
	if err != nil {
		mm.processWideMutex.Unlock()
		return fmt.Errorf("outerProcessMutex.Lock: %w", err)
	}
	return nil
}

func (mm *masterMutex) unlock() error {
	err := mm.systemWideMutex.Unlock()
	if err != nil {
		return fmt.Errorf("outerProcessMutex.Unlock: %w", err)
	}
	mm.processWideMutex.Unlock()
	return nil
}

// LockFile locks a file for exlusive access
// lock: r/w
func LockFile(path string) (*masterMutex, error) {
	mm, err := NewMasterMutex(path)
	if err != nil {
		return nil, fmt.Errorf("NewMasterMutex: %w", err)
	}
	return mm, mm.lock()
}

// UnlockFile unlocks a file from exlusive access
// lock: r/w
func UnlockFile(mm *masterMutex) error {
	return mm.unlock()
}

// InitPath creates a directory if it does not exist already
func InitPath(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debug("mkdir ", path)
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// InitFile creates a file if it does not exist already
func InitFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Debug("touch ", path)
		err := os.WriteFile(path, []byte(""), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

// FindFiles returns a list of files that match the given pattern except:
// - files that are ignored by the workspace .gitignore
// - [optional] files that are ignored by the project .gitignore
func FindFiles(ctxLogger *log.Entry, searchRoot string, pattern string, projectIgnoreMatcher *ignore.GitIgnore) ([]string, error) {
	var files []string
	err := filepath.WalkDir(searchRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// We just care about not walking subdirs that are ignored
		if entry.IsDir() {
			if WorkspaceIgnoreMatcher.MatchesPath(path) {
				return filepath.SkipDir
			}
			if projectIgnoreMatcher != nil && projectIgnoreMatcher.MatchesPath(path) {
				return filepath.SkipDir
			}
		}

		match, err := regexp.MatchString(pattern, path)
		if err != nil {
			log.Fatalf("regexp.MatchString: %v", err)
		}
		if match {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// This type represents a "header" to apply to a bash script (prepend to it)
//
// Example:
// #!/bin/bash
// export global_var=1
type ScriptHeaderSection struct {
	Creator string // This is either a project id or "workspace"
	Content string // This is a bash script
}

// ReadScriptHeader reads one full script header from a file (as in one .env file)
// lock: depends on caller read lock
func ReadScriptHeader(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "# no .tasker/.env file in " + path + " so no env to load\n", nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteScriptHeader writes one full script header to a file (as in one .env file)
// lock: depends on caller write lock
func WriteScriptHeader(path string, content string) error {
	_, err := os.Stat(path)
	if err != nil {
		return err
	}

	envFile, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer envFile.Close()

	err = envFile.Truncate(0)
	if err != nil {
		return err
	}

	_, err = envFile.Write([]byte(content))
	if err != nil {
		return err
	}
	return nil
}

func NewScriptHeaderSection(creator string, content string) ScriptHeaderSection {
	return ScriptHeaderSection{
		Creator: creator,
		Content: content,
	}
}

func (sh ScriptHeaderSection) ToRawScript() string {
	return "# creator: " + sh.Creator + "\n" + sh.Content + "\n"
}

//
// A hack way to debug deadlocks, not in use right now
//

// These are only process specific locks
var procFileLockIds = []fileLockId{}
var procFileLockIdsMutex = sync.Mutex{}
var debugOn = true
var running = false
var startDebug = func() {
	if !debugOn || running {
		return
	}
	running = true
	go func() {
		for {
			logDebugInfo()
			time.Sleep(1 * time.Second)
		}
	}()
}

func logDebugInfo() {
	procFileLockIdsMutex.Lock()
	defer procFileLockIdsMutex.Unlock()
	out := "\n"
	for _, lock := range procFileLockIds {
		out += lock.id
	}
	if out == "\n" {
		log.Warn("locks: no files are locked in this process procCall='", strings.Join(os.Args, " ")+"'")
	} else {
		log.Warn("locks: ", out)
	}
}

type fileLockId struct {
	id   string
	file string
}

func newFileLockId(lockType string, filePath string, fileDesc int) fileLockId {
	procCall := strings.Join(os.Args, " ")
	_, file, no, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		no = 0
	}
	funcCall := fmt.Sprintf("%s#%d", file, no)
	return fileLockId{
		id: "\t[lockType='" + lockType + "']\n" +
			"\t[procCall='" + procCall + "']\n" +
			"\t[funcCall='" + funcCall + "']\n" +
			"\t[filePath='" + filePath + "']\n" +
			"\t[fileDesc='" + strconv.Itoa(fileDesc) + "]\n",
		file: filePath,
	}
}

func debugLock(ltype string, file string, fd int) {
	logDebugInfo()
	id := newFileLockId(ltype, file, fd)
	procFileLockIdsMutex.Lock()
	log.Warn("locked: \n", id.id)
	procFileLockIds = append(procFileLockIds, id)
	procFileLockIdsMutex.Unlock()
	logDebugInfo()
}

func debugUnlock(ltype string, file string, fd int) {
	logDebugInfo()
	procFileLockIdsMutex.Lock()
	newLockedList := []fileLockId{}
	for _, lockId := range procFileLockIds {
		// Check only file as callsite and others can be different
		if lockId.file != file {
			newLockedList = append(newLockedList, lockId)
		} else {
			log.Warn("unlocked: \n", lockId.id)
		}
	}
	procFileLockIds = newLockedList
	procFileLockIdsMutex.Unlock()
	logDebugInfo()
}

func readPrivateFieldAsInt[T any, U any](f *T, field string) int {
	structVal := reflect.ValueOf(*f)
	fieldVal := structVal.FieldByName(field)
	return int(fieldVal.Int())
}
