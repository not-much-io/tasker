package tasks

import (
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker/common"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Task struct {
	ProjectDef defs.ProjectDefinition
	TaskDef    defs.TaskDefinition
}

func NewTask(ctx common.Context, taskDef defs.TaskDefinition) Task {
	projectDef := ctx.MapTaskToProject(taskDef.Id)
	return Task{
		ProjectDef: projectDef,
		TaskDef:    taskDef,
	}
}

func (task Task) Run(ctx common.Context) (string, error) {
	logPrefix := "[task=" + string(task.TaskDef.Id) + "] "
	log.Info(logPrefix + "starting task")
	res, err := RunBash(ctx, task)
	log.Info(logPrefix + "finished task")
	return res, err
}

func RunBash(ctx common.Context, task Task) (string, error) {
	log.Debug("running RunBashImpl for task: ", task.TaskDef.Id)

	// We use a tmp script file that exec.Command can execute
	scriptId := randSeq(8)
	tmpScriptFilePath := "/tmp/" + scriptId + ".sh"

	prjEnv, err := ctx.GetProjectState(task.ProjectDef.Id).GetProjectEnv()
	if err != nil {
		return "", err
	}

	bashScript := lib.StdBashHeader() +
		ctx.Workspace.GetEnv() +
		prjEnv +
		task.TaskDef.GetEnv() +
		task.TaskDef.Task +
		"\n"
	err = os.WriteFile(tmpScriptFilePath, []byte(bashScript), 0777)
	if err != nil {
		return "", err
	}

	// Clean up tmp script file when done using it
	defer func() {
		_, err := os.Stat(tmpScriptFilePath)
		if err == nil {
			err = os.Remove(tmpScriptFilePath)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()

	// Setup the command as a direct call to /bin/bash in the project dir
	cmd := exec.Command("/bin/bash", tmpScriptFilePath)
	cmd.Dir = task.ProjectDef.Path

	// Tail all output from the script
	allOut := ""
	allOutPipe, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout // pipe both stdout and stderr to the same pipe
	if err != nil {
		return "", err
	}
	go func() {
		defer allOutPipe.Close()
		buf := make([]byte, 1024)
		for {
			n, err := allOutPipe.Read(buf)
			if err != nil {
				// Stop condition for goroutine
				if err == io.EOF {
					break
				}
			}
			newOut := string(buf[:n])

			for _, line := range strings.Split(newOut, "\n") {
				// Sometimes seems like whitespace gets spammer?
				// Unsure, just in case doing this and keeping an eye out
				newOut = strings.Trim(newOut, " ")
				if newOut == "" {
					continue
				}
				log.Infof("[task=%s] %s", string(task.TaskDef.Id), line)
			}
			allOut += newOut
		}
	}()

	log.Debug("running script: ", tmpScriptFilePath)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		log.Error("Failed to run script with error: ", cmdErr)
		scriptContent, readFileErr := os.ReadFile(tmpScriptFilePath)
		if readFileErr != nil {
			return "", readFileErr
		}
		log.Error("failed script content:\n\t", strings.ReplaceAll(string(scriptContent), "\n", "\n\t"))
		return "", cmdErr
	}
	log.Debug("finished running script: ", tmpScriptFilePath)

	return string(allOut), nil
}

//
// Utils
//

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
