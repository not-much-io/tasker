package state

import (
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"log"
)

// mut: true
type WorkspacePersistentState struct {
	RefToDefns              RefToDefns               // mut: false
	ProjectPersistentStates []ProjectPersistentState // mut: true
}

func NewWorkspacePersistentState(refToDefns RefToDefns) (WorkspacePersistentState, error) {
	newState := WorkspacePersistentState{}
	err := newState.Load(refToDefns)
	if err != nil {
		return newState, err
	}
	return newState, nil
}

func (s *WorkspacePersistentState) Load(refToWsDefn RefToDefns) error {
	states := []ProjectPersistentState{}
	for _, prjDefn := range refToWsDefn.Wsp.Projects {
		refToWsDefnCopy := refToWsDefn
		refToWsDefnCopy.Prj = &prjDefn
		newState, err := NewProjectPersistentState(refToWsDefnCopy)
		if err != nil {
			return err
		}
		states = append(states, newState)
	}
	s.RefToDefns = refToWsDefn
	s.ProjectPersistentStates = states
	return nil
}

func (s WorkspacePersistentState) Dump() error {
	return nil
}

// TODO: Implement in fs when needed
func (s WorkspacePersistentState) ReadScriptHeader() string {
	workspaceHeader := "export ws_root_path=" + lib.WsRootPath + "\n"
	return lib.NewScriptHeaderSection(
		"workspace",
		workspaceHeader,
	).ToRawScript()
}

func (wsps WorkspacePersistentState) GetProjectState(projectId defs.ProjectId) ProjectPersistentState {
	for _, projectState := range wsps.ProjectPersistentStates {
		if projectState.RefToDefns.Prj.Id == projectId {
			return projectState
		}
	}

	log.Fatal("Project not found: ", projectId)
	return ProjectPersistentState{}
}
