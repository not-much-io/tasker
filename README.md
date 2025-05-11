# inference-tasker

## architecture

`tasker` is the main binary that is called once per execution. It handles:
* initing the workspace.yaml file based on all the found project.yaml files
* initing and managing states between executions in .tasker/ dirs
* resolving project interdependencies
* scheduling tasks for execution (as parallel as possible)
* evaluating custom task skipping conditions if applicable

Tasker internally calls out to bash scripts defined in project.yaml files which can use tasker utilsbins like `finder`, `linker` and `setter`. These utilbins can also manage state in .tasker/ dirs.
