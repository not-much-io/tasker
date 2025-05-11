package skipper

import (
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker/common"
	"inference-tasker/lib/tasker/tasks"
)

type Skipper struct {
	ctx *common.Context
}

func NewSkipper(ctx *common.Context, args common.TaskerArgs) Skipper {
	return Skipper{ctx: ctx}
}

func (c Skipper) ShouldSkip(task tasks.Task) bool {
	if task.TaskDef.Cond == defs.OnceCondition {
		return c.checkConditionOnce()
	}
	return false
}

func (c Skipper) checkConditionOnce() bool {

	return false
}
