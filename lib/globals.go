package lib

import (
	ignore "github.com/sabhiram/go-gitignore"
	log "github.com/sirupsen/logrus"
)

// Needed?
var DefaultLogger = log.WithFields(log.Fields{
	"extra": "defaultLogger",
})

// TODO: Read this in from a workspace.conf file in the global context
var WsRootPath = "/workspaces/inference"
var WsTaskerPath = WsRootPath + TaskerDir
var WorkspaceIgnoreMatcher = getWorkspaceIgnoreMatcher()

// Pick up the top level .gitignore file
// Each project task handles its own .gitignore file but the global one is always applied
func getWorkspaceIgnoreMatcher() *ignore.GitIgnore {
	ignoreMatcher, err := ignore.CompileIgnoreFile(WsRootPath + "/.gitignore")
	if err != nil {
		log.Fatalf("ignore.CompileIgnoreFile: %v", err)
	}
	return ignoreMatcher
}

var stdBashEnv = NewScriptHeaderSection(
	"globals.go",
	"set -Eeuo pipefail",
)

func StdBashHeader() string {
	return stdBashEnv.ToRawScript()
}
