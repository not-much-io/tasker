package defs

import (
	"inference-tasker/lib"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const (
	ProjectFile       = "project.yaml"
	ProjectTaskerPath = ".tasker"
)

// todo: track uniqueness of project ids in a global set?
type ProjectId = string

// A ProjectDefinition in the workspace
// mut: false
type ProjectDefinition struct {
	// The project file full path, aka /path/to/project.yaml
	File string `yaml:"file"`
	// The project path, aka /path/to
	Path string `yaml:"path"`
	// The id of the project, aka "writer" or "writer::norrland"
	Id ProjectId `yaml:"id"`
	// All the project tasks, aka "install", "build", "test", "lint", etc.
	TaskDefs []TaskDefinition `yaml:"tasks"`
}

func InitProject(filePath string) ProjectDefinition {
	log.Debug("reading project @ " + filePath)

	project := ProjectDefinition{}
	project.File = filePath
	project.Path = strings.Replace(filePath, "/"+ProjectFile, "", -1)

	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("File read: ", err)
	}

	err = yaml.Unmarshal(yamlFile, &project)
	if err != nil {
		log.Error("Unmarshal: ", err)
	}

	log.Debug("reading project done!")
	return project
}

func (project ProjectDefinition) GetEnv() string {
	return "# Prepend project env\n" + "export " + lib.CurrTskrProject + "=\"" + project.Id + "\"\n"
}
