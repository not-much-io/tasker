package state

// mut: false
type TaskPersistentState struct {
	RefToDefns RefToDefns // mut: false
}

func NewTaskPersistentState(refToDefns RefToDefns) (TaskPersistentState, error) {
	newState := TaskPersistentState{}
	newState.Load(refToDefns)
	return newState, nil
}

func (tps TaskPersistentState) Load(refToWsDefn RefToDefns) (TaskPersistentState, error) {
	return TaskPersistentState{
		refToWsDefn,
	}, nil
}

func (tps TaskPersistentState) Dump() error {
	return nil
}
