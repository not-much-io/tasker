package state

import (
	"fmt"
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"strings"
)

// mut: true
type ProjectPersistentState struct {
	RefToDefns           RefToDefns            // mut: false
	TaskPersistentStates []TaskPersistentState // mut: true
}

func NewProjectPersistentState(refToDefns RefToDefns) (ProjectPersistentState, error) {
	prj := refToDefns.Prj
	if prj == nil {
		return ProjectPersistentState{}, fmt.Errorf("NewProjectPersistentState: refToDefns.Prj is nil")
	}
	newState := ProjectPersistentState{}
	err := newState.Load(refToDefns)
	if err != nil {
		return newState, err
	}
	return newState, nil
}

func (pps *ProjectPersistentState) Load(refToWsDefn RefToDefns) error {
	states := []TaskPersistentState{}
	for _, taskDefn := range refToWsDefn.Prj.TaskDefs {
		refToWsDefnCopy := refToWsDefn
		refToWsDefnCopy.Tsk = &taskDefn
		newState, err := NewTaskPersistentState(refToWsDefnCopy)
		if err != nil {
			return err
		}
		states = append(states, newState)
	}
	pps.RefToDefns = refToWsDefn
	pps.TaskPersistentStates = states
	return nil
}

// Init the project .tasker dir to hold state if it doesn't already exist
func (pps ProjectPersistentState) Init() error {
	err := lib.InitPath(pps.RefToDefns.Prj.Path + lib.TaskerDir)
	if err != nil {
		return err
	}
	err = lib.InitFile(pps.EnvPath())
	if err != nil {
		return err
	}
	err = lib.InitFile(pps.RefToDefns.Prj.Path + lib.TaskerDir + lib.LastRunsFile)
	if err != nil {
		return err
	}
	return nil
}

type SetInProjectEnvParams struct {
	Tsk defs.TaskId
	Kvs []EnvKeyVal
}

// SetInProjectEnv sets the given key value pairs in the project env file
// lock: r/w (on the projcets env file)
func (pps ProjectPersistentState) SetInProjectEnv(params SetInProjectEnvParams) error {
	envPath := pps.EnvPath()

	// Any parallel process could potentianlly try to access the same env file.
	// So we need to lock it with exclusive access until we are done with this invocation of setter.
	mm, err := lib.LockFile(envPath)
	if err != nil {
		return fmt.Errorf("SetInProjectEnv: %w", err)
	}
	defer lib.UnlockFile(mm)

	content, err := lib.ReadScriptHeader(envPath)
	if err != nil {
		return err
	}

	// Only add exports that are not already in the file
	for _, kv := range params.Kvs {
		exportExpr := kv.AsExportExpr(params.Tsk)
		if !strings.Contains(string(content), exportExpr) {
			content += exportExpr
		}
	}

	err = lib.WriteScriptHeader(envPath, content)
	if err != nil {
		return err
	}

	return nil
}

// GetProjectEnv returns the project env file content + static export of the current project id
// lock: r (on the projcts env file)
func (pps ProjectPersistentState) GetProjectEnv() (string, error) {
	mm, err := lib.LockFile(pps.EnvPath())
	if err != nil {
		return "", fmt.Errorf("GetProjectEnv: %w", err)
	}
	defer lib.UnlockFile(mm)
	content, err := lib.ReadScriptHeader(pps.EnvPath())
	if err != nil {
		return "", err
	}
	content += "export " + lib.CurrTskrProject + "=\"" + pps.RefToDefns.Prj.Id + "\"\n"
	return lib.NewScriptHeaderSection(
		lib.PrependedEnv,
		content,
	).ToRawScript(), nil
}

func (pps ProjectPersistentState) EnvPath() string {
	return pps.RefToDefns.Prj.Path + lib.TaskerDir + lib.EnvFile
}
