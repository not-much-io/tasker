package lib

import "time"

// fs constants
const TaskerDir = "/.tasker"
const EnvFile = "/.env"
const LastRunsFile = "/last_run.yaml"

// bash variables
const PrependedEnv = "env prepend"
const CurrTskrProject = "curr_tskr_project"
const CurrTskrTask = "curr_tskr_task"
const FinderRootParam = "finder_root_param"

// utils
const StdSleepWait = 100 * time.Millisecond
