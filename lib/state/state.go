package state

import (
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
)

// PersistentState is a generic interface for all persistent state types
// Persistent state in tasker is implemented as read/write to files in the .tasker directory
// Persistent state is also the *only* way for tasks to communicate state between each other
//
// # NOTE: Scheduler state
// Scheduler also communicates "task state" but not really. That state is considered scheduler state not task state.
// Neither is it persistent, though it can be inferred from persistent state!
type PersistentState[D RefToDefns] interface {
	Load(D) error
	Dump() error
}

type RefToDefns struct {
	Wsp defs.WorkspaceDefinition
	Prj *defs.ProjectDefinition
	Tsk *defs.TaskDefinition
}

type EnvKeyVal struct {
	Key string
	Val string
}

func (kv EnvKeyVal) AsExportExpr(tskId defs.TaskId) string {
	return lib.NewScriptHeaderSection(
		string(tskId),
		"export "+kv.Key+"="+kv.Val,
	).ToRawScript()
}
