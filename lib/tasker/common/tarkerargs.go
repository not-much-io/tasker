package common

import (
	"inference-tasker/lib/defs"
	"os"
	"strings"
)

type TaskerArgs struct {
	// ex. "assets"
	ProjectId defs.ProjectId
	// ex. "assets::set_env"
	TaskId defs.TaskId
	// ex. "foo=bar"
	Arguments []string
	// ex. "tasker assets::set_env foo=bar"
	AsRawString string
}

func ParseTaskerArgs(ctx Context, cliArgs []string) TaskerArgs {
	if len(cliArgs) == 0 {
		ctx.Logger.Fatal("no args passed to tasker")
	}

	ctx.Logger.Debug("tasker args: ", cliArgs)
	project := strings.Split(cliArgs[0], "::")[0]
	return TaskerArgs{
		ProjectId:   defs.ProjectId(project),
		TaskId:      defs.TaskId(cliArgs[0]),
		Arguments:   cliArgs[1:],
		AsRawString: strings.Join(os.Args, " "), // just put it all back together
	}
}
