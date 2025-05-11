package defs

import (
	"inference-tasker/lib"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// TODO: Set this up in init or something
const WS_ROOT_PATH = "/workspaces/inference"
const WS_TASKER_PATH = WS_ROOT_PATH + "/.tasker"
const WS_FILE_PATH = WS_TASKER_PATH + "/workspace.yaml"
const WS_ENV_FILE = WS_TASKER_PATH + "/.env"
const WS_PROJECT_FILE = "project.yaml"

// WorkspaceDefinition contains all information about the workspace definitions in the fs
// It can be dumped to file and reloaded or alternatively inited from scratch
// mut: false
type WorkspaceDefinition struct {
	RootPath    string              `yaml:"root"`
	TaskerPath  string              `yaml:"taskerPath"`
	DefnPath    string              `yaml:"defnPath"`
	EnvFilePath string              `yaml:"envFilePath"`
	Projects    []ProjectDefinition `yaml:"projects"`
}

// InitWorkspace inits the workspace from scratch - existing workspace.yaml overwritten
// TODO:
// Specifically init only with `tasker init` and not on every run.
// This is faster and safer - as in parallel execution fs might change as looking for project.yaml files.
func InitWorkspace(ctxLogger *log.Entry) WorkspaceDefinition {
	err := initTaskerPath()
	if err != nil {
		ctxLogger.Fatal("Error initializing tasker path: ", err)
	}
	err = initWorkspaceFile()
	if err != nil {
		ctxLogger.Fatal("Error initializing workspace file: ", err)
	}
	err = initEnvFile()
	if err != nil {
		ctxLogger.Fatal("Error initializing env file: ", err)
	}

	ws := WorkspaceDefinition{}
	ws.RootPath = WS_ROOT_PATH
	ws.TaskerPath = WS_TASKER_PATH
	ws.DefnPath = WS_FILE_PATH
	ws.EnvFilePath = WS_ENV_FILE

	// Find all the project.yaml files in the workspace
	projectDefs, err := findProjectDefs(ctxLogger)
	if err != nil {
		log.Fatal("Error finding project.yaml files: ", err)
	}

	// Read all the project.yaml files into Project structs
	ws.Projects = []ProjectDefinition{}
	for _, projectDef := range projectDefs {
		ws.Projects = append(ws.Projects, InitProject(projectDef))
	}

	// Validate that all deps actually exist
	for _, project := range ws.Projects {
		for _, task := range project.TaskDefs {
			for _, dep := range task.Deps {
				if !ws.containsTask(dep) {
					log.
						WithFields(log.Fields{
							"project": project.Id,
							"task":    task.Id,
							"dep":     dep,
						}).
						Fatal("Invalid dep!")
				}
			}
		}
	}

	return ws
}

// Dump dumps the workspace to the workspace.yaml file in the workspace ./tasker dir
func (ws WorkspaceDefinition) Dump() {
	wsFile, err := os.OpenFile(WS_FILE_PATH, os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	err = wsFile.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
	defer wsFile.Close()

	wsYaml, err := yaml.Marshal(ws)
	if err != nil {
		log.Fatal(err)
	}

	_, err = wsFile.Write(wsYaml)
	if err != nil {
		log.Fatal(err)
	}
}

//
// Finders
//

//
// Utils
//

func initTaskerPath() error {
	return lib.InitPath(WS_TASKER_PATH)
}

func initWorkspaceFile() error {
	return lib.InitFile(WS_FILE_PATH)
}

func initEnvFile() error {
	return lib.InitFile(WS_ENV_FILE)
}

func findProjectDefs(ctxLogger *log.Entry) ([]string, error) {
	return lib.FindFiles(ctxLogger, WS_ROOT_PATH, WS_PROJECT_FILE, nil)
}

func (wsd WorkspaceDefinition) containsTask(taskId TaskId) bool {
	for _, project := range wsd.Projects {
		for _, task := range project.TaskDefs {
			if task.Id == taskId {
				return true
			}
		}
	}
	return false
}

func (wsd WorkspaceDefinition) MapTaskToProject(taskId TaskId) ProjectDefinition {
	for _, project := range wsd.Projects {
		for _, task := range project.TaskDefs {
			if task.Id == taskId {
				return project
			}
		}
	}
	log.Fatal("Project not found for task: ", taskId)
	return ProjectDefinition{}
}
