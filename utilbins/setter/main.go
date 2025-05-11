package main

import (
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"inference-tasker/lib/state"
	"inference-tasker/lib/tasker/common"

	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var ctxLog *log.Entry

func main() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		PadLevelText:  true,
		FullTimestamp: false,
	})

	prjId := os.Getenv(lib.CurrTskrProject)
	tskId := os.Getenv(lib.CurrTskrTask)
	args := os.Args[1:]
	validateInput(args, prjId)
	ws := defs.InitWorkspace(ctxLog)
	ctxLog = log.WithFields(
		log.Fields{
			lib.CurrTskrProject: prjId,
			"bin":               os.Args[0],
		},
	)
	ctx := common.NewContext(ctxLog, ws)

	// old
	mm, err := lib.LockFile(ws.EnvFilePath)
	if err != nil {
		ctx.Logger.Fatal("Failed to lock env file: %v", err)
	}
	defer lib.UnlockFile(mm)

	for i := 0; i < len(args); i += 2 {
		key := args[i]
		val := args[i+1]
		setEnvGlobal(ws, key, val, prjId)
	}

	// new
	kvs := []state.EnvKeyVal{}
	for i := 0; i < len(args); i += 2 {
		kvs = append(kvs, state.EnvKeyVal{
			Key: args[i],
			Val: args[i+1],
		})
	}
	projectState := ctx.GetProjectState(defs.ProjectId(prjId))
	err = projectState.SetInProjectEnv(state.SetInProjectEnvParams{
		Tsk: defs.TaskId(tskId),
		Kvs: kvs,
	})
	if err != nil {
		ctx.Logger.
			Fatal("Failed to set project env: %v", err)
	}
}

func validateInput(args []string, envv string) {
	if envv == "" {
		log.Fatal("no project env var set, should always be set by tasker!")
	}
	if len(args) < 2 || len(args)%2 != 0 {
		log.Fatal("incorrect arguments to setter! Should have 2+ and an even number of arguments")
	}
}

func setEnvGlobal(ws defs.WorkspaceDefinition, key string, val string, projectId string) {
	// New export expr
	exportExpr := "# added by project: " + string(projectId) + "\n"
	exportExpr = exportExpr + "export " + key + "=" + val + "\n"

	// Read in the current env file
	log.Debug("opening env file: ", ws.EnvFilePath, " for appending")
	envFile, err := os.OpenFile(ws.EnvFilePath, os.O_RDWR, 0644)
	if err != nil {
		ctxLog.Fatal(err)
	}
	defer envFile.Close()
	currEnv, err := io.ReadAll(envFile)
	if err != nil {
		ctxLog.Fatal(err)
	}

	// If new export expr is already in currEnv, don't add it again
	if !strings.Contains(string(currEnv), exportExpr) {
		_, err = envFile.WriteString(exportExpr)
		if err != nil {
			ctxLog.Fatal(err)
		}
		ctxLog.Info("exported ", key, "=", val)
	}
}
