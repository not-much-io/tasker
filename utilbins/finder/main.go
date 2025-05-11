package main

import (
	"fmt"
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// No logging on successful run as need to pass result back to bash as output
var ctxLog *log.Entry

func main() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		PadLevelText:  true,
		FullTimestamp: false,
	})

	proj := os.Getenv(lib.CurrTskrProject)
	if proj == "" {
		ctxLog.Fatal("no project env var set, should always be set by tasker!")
	}
	ctxLog = log.WithFields(
		log.Fields{
			lib.CurrTskrProject: proj,
			"bin":               os.Args[0],
		},
	)
	root, quer := parseInput()
	ctxLog = ctxLog.WithFields(log.Fields{
		"root": root,
		"quer": quer,
	})
	if root == "" || quer == "" {
		ctxLog.Fatal("root or query empty!")
	}
	ws := defs.InitWorkspace(ctxLog)
	result := find(ws, root, quer)

	result = strings.Trim(result, " ")
	if result == "" {
		ctxLog.Fatal("result empty!")
	}
	fmt.Println(result)
}

func parseInput() (string, string) {
	root := os.Getenv(lib.FinderRootParam)
	quer := ""
	args := os.Args[1:]
	if len(args) == 0 {
		ctxLog.Fatal("no arguments passed to finder!")
	}
	if len(args) == 1 {
		quer = args[0]
		if root == "" {
			ctxLog.Fatal("no root set by env var and only 1 argument passed to finder!")
		}
		return root, quer
	}
	if len(args) == 2 {
		root = args[0]
		quer = args[1]
		return root, quer
	}
	ctxLog.WithFields(
		log.Fields{
			"args": args,
			"root": root,
		},
	).Fatal("incorrect arguments to finder! Should have 2 arguments or 1 argument and have root set by env var")
	return "", ""
}

// TODO: Also ignore .gitignore like ws
func find(ws defs.WorkspaceDefinition, root string, query string) string {
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			ctxLog.
				WithFields(log.Fields{
					"path":  path,
					"query": query,
					"err":   err,
				}).
				Fatal("hit error while looking for matches to query.")
		}
		if strings.Contains(path, query) {
			matches = append(matches, path)
		}
		return err
	})
	if err != nil {
		ctxLog.Fatal(err)
	}
	if len(matches) == 0 {
		ctxLog.Fatal("no matches found for query: ", query, " in root: ", root)
	}
	if len(matches) > 1 {
		ctxLog.Fatal(
			"multiple matches found for query: ", query, " in root: ", root, " matches:\n\t* ",
			strings.Join(matches, "\n\t* "))
	}
	return matches[0]
}
