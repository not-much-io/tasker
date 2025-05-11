package main

import (
	"fmt"
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker/common"
	"os"
	"path"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// TODO: Brittle.. as space after \ in bash is taken as end of args?
func main() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		PadLevelText:  true,
		FullTimestamp: false,
	})
	proj := os.Getenv(lib.CurrTskrProject)
	ctxLog := log.WithFields(log.Fields{
		lib.CurrTskrProject: proj,
		"bin":               os.Args[0],
	})
	wsd := defs.InitWorkspace(ctxLog)
	ctx := common.NewContext(ctxLog, wsd)
	args := os.Args[1:]
	args = cleanArgs(args)
	validateInput(ctx, args, proj)

	wg := sync.WaitGroup{}
	for i := 0; i < len(args); i += 2 {
		log.WithFields(log.Fields{
			"src": args[i],
			"dst": args[i+1],
		}).Debug("next linker args")
		src := prepCliVals(ctx, args[i])
		dst := prepCliVals(ctx, args[i+1])

		wg.Add(1)
		go func() {
			defer wg.Done()
			link(ctx, src, dst, proj)
		}()
	}
	wg.Wait()
}

func cleanArgs(args []string) []string {
	newArgs := []string{}
	for i := 0; i < len(args); i++ {
		trimmed := strings.Trim(args[i], " ")
		if trimmed == "" {
			continue
		}
		newArgs = append(newArgs, trimmed)
	}
	return newArgs
}

func validateInput(ctx common.Context, args []string, proj string) {
	if proj == "" {
		ctx.Logger.Fatal("no project env var set, should always be set by tasker!")
	}
	if len(args) < 2 || len(args)%2 != 0 {
		argsList := ""
		for i, arg := range args {
			argsList += "\n\t" + fmt.Sprint(i+1) + ") " + arg
		}
		ctx.Logger.Fatal(
			"incorrect arguments to linker! Should have 2+ and an even number of arguments. Real:",
			argsList,
		)
	}
}

func link(ctx common.Context, src string, dst string, projectId string) {
	dst = prepCliVals(ctx, ctx.GetProjectDef(projectId).Path+"/"+dst)

	if !isExisting(ctx, src) {
		ctx.Logger.Fatal("src does not exist: ", wrapPathInQuotes(src))
	}

	if isExisting(ctx, dst) {
		if isSymlink(ctx, dst) {
			removePath(ctx, dst)
		} else {
			ctx.Logger.
				WithFields(log.Fields{
					"dst":          wrapPathInQuotes(dst),
					"dstExists":    isExisting(ctx, dst),
					"dstIsSymLink": isSymlink(ctx, dst),
				}).
				Fatal("dst already exists and is not a symlink, not overwriting.")
		}
	}

	assurePathExists(ctx, dst)

	ctx.Logger.Info("linking: \n\tsrc: ", src, "\n\t↪dst: ", dst)
	err := os.Symlink(src, dst)
	if err != nil {
		ctx.Logger.
			WithFields(log.Fields{
				"src":       wrapPathInQuotes(src),
				"srcExists": isExisting(ctx, src),
				"dst":       wrapPathInQuotes(dst),
				"dstExists": isExisting(ctx, dst),
			}).
			Fatal(err)
	}
}

func isExisting(ctx common.Context, path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		ctx.Logger.
			WithField("os.Stat err", err).
			Debug("path does not exist: ", wrapPathInQuotes(path))
		return false
	}
	if err != nil {
		ctx.Logger.Fatal("An unexpected error happened while checking for file existence: ", err)
	}
	ctx.Logger.Debug("path exists: ", wrapPathInQuotes(path))
	return true
}

func isSymlink(ctx common.Context, path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		ctx.Logger.Fatal(err)
	}
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		ctx.Logger.Debug("path is a symlink: ", wrapPathInQuotes(path))
		return true
	}
	ctx.Logger.Debug("path is not a symlink: ", wrapPathInQuotes(path))
	return false
}

func removePath(ctx common.Context, path string) {
	ctx.Logger.Info("removing existing symlink: ", wrapPathInQuotes(path))
	err := os.Remove(path)
	if err != nil {
		ctx.Logger.Fatal(err)
	}
}

func assurePathExists(ctx common.Context, path_ string) {
	pathDir := path.Dir(path_)
	err := os.MkdirAll(pathDir, 0755) // If exist, does nothing
	if err != nil {
		ctx.Logger.Fatal(err)
	}
}

func prepCliVals(ctx common.Context, val string) string {
	// Trim whitespace
	newVal := strings.Trim(val, " ")

	if newVal == "" {
		ctx.Logger.Fatal("empty string passed as cli argument!")
	}

	return newVal
}

func wrapPathInQuotes(path string) string {
	return "\"" + path + "\""
}
