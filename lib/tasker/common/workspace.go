package common

import (
	"inference-tasker/lib"
	"inference-tasker/lib/defs"
	"inference-tasker/lib/state"
	"os"

	log "github.com/sirupsen/logrus"
)

// mut: true
type Workspace struct {
	// Describes the workspaces configuration
	// Is stateless after initialized/read
	// Between tasker runs it is stored in project.yaml and workspace.yaml files
	Definition defs.WorkspaceDefinition // mut: false
	// Describes the workspaces state
	// Is stateful and stored in .tasker/ folders in project and workspace paths
	State state.WorkspacePersistentState // mut: true
}

func NewWorkspace(workspaceDef defs.WorkspaceDefinition) Workspace {
	state, err := state.NewWorkspacePersistentState(
		state.RefToDefns{
			Wsp: workspaceDef,
			Prj: nil,
			Tsk: nil,
		},
	)
	if err != nil {
		// todo: propagate error
		log.Fatal(err)
	}
	return Workspace{
		Definition: workspaceDef,
		State:      state,
	}
}

// GetEnv returns the contents of the env file for pre-pending to commands
// BROKEN
func (ws Workspace) GetEnv() string {
	defaultEnv := "# default env\n"
	defaultEnv += "export ws_root_path=" + lib.WsRootPath + "\n"

	// check if env file exists
	if _, err := os.Stat(ws.Definition.EnvFilePath); os.IsNotExist(err) {
		return defaultEnv + "# no workspace .tasker/.env file found, no env to load\n"
	}

	env, err := os.ReadFile(ws.Definition.EnvFilePath)
	if err != nil {
		log.Fatal(err)
	}

	return defaultEnv + "# workspace env from .tasker/.env \n" + string(env)
}

func (ws Workspace) GetProjectState(projectId defs.ProjectId) state.ProjectPersistentState {
	for _, projectState := range ws.State.ProjectPersistentStates {
		if projectState.RefToDefns.Prj.Id == projectId {
			return projectState
		}
	}

	log.Fatal("Project not found: ", projectId)
	return state.ProjectPersistentState{}
}
