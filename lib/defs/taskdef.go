package defs

import "inference-tasker/lib"

type TaskId string

type TaskArgs = string

type Condition string

// default
// unless-already-run-once
// unless-explicitly-called
// unless-fs-changes {arg}

const (
	// Run task only if:
	// * It's deps ran
	// * Something in the projects dir changed
	// * It is called explicitly
	//   NOTE: this includes the project.yaml itself!
	// Usecases: This should cover most cases and is set implicitly
	DefaultCondition Condition = "default"
	// Run task only if:
	// * It has not been run before
	// * It is called explicitly
	// Usecases: Installing "project external" dependencies like OS level packages
	OnceCondition Condition = "once"
	// Run task only if it is called explicitly
	// Usecases: Like cleaning outputs to start from scratch to shake loose an undefined state
	ExplicitCondition Condition = "explicit"
)

// mut: false
type TaskDefinition struct {
	// ex. "assets::install"
	Id TaskId `yaml:"id"` // TODO: This currently can conflict with std tasks
	// ex. "once"
	Cond Condition `yaml:"cond"`
	// ex. ["assets::build"]
	Deps []TaskId `yaml:"deps"`
	// ex. "echo 'hello world'" for a bash task
	Task TaskArgs `yaml:"task"`
}

func (task TaskDefinition) GetEnv() string {
	return "# Prepend task env\n" + "export " + lib.CurrTskrTask + "=\"" + string(task.Id) + "\"\n"
}
