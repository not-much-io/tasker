package common

import (
	"inference-tasker/lib/defs"
	"inference-tasker/lib/state"

	log "github.com/sirupsen/logrus"
)

// mut: true
type Context struct {
	Logger    *log.Entry
	Workspace Workspace
}

func NewContext(logger *log.Entry, workspaceDef defs.WorkspaceDefinition) Context {
	return Context{
		Logger:    logger,
		Workspace: NewWorkspace(workspaceDef),
	}
}

//
// START: Utility accessors / helpers
//
// These just cut out chaining structs or/and implement minor utils
//

func (ctx Context) GetAllTaskDefs() []defs.TaskDefinition {
	allTasks := []defs.TaskDefinition{}
	for _, project := range ctx.Workspace.Definition.Projects {
		allTasks = append(allTasks, project.TaskDefs...)
	}
	return allTasks
}

func (ctx Context) GetProjectDef(projectId defs.ProjectId) defs.ProjectDefinition {
	for _, project := range ctx.Workspace.Definition.Projects {
		if project.Id == projectId {
			return project
		}
	}

	log.Fatal("Project not found: ", projectId)
	return defs.ProjectDefinition{}
}

// TODO: Spread the inner structs
func (ctx Context) MapTaskToProject(taskId defs.TaskId) defs.ProjectDefinition {
	return ctx.Workspace.Definition.MapTaskToProject(taskId)
}

func (ctx Context) GetProjectState(projectId defs.ProjectId) state.ProjectPersistentState {
	return ctx.Workspace.State.GetProjectState(projectId)
}

//
// END: Utility accessors / helpers
//
