package scheduler

import (
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker/common"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Scheduler is a simple implementation of a tasks scheduler.
//
// Mainly it is "simple" because it doesn't pre-plan a full execution schedule but rather:
// 1) Evaluates the "next runnable task" on demand.
// 2) Requires the caller to poll for the next runnable task.
type Scheduler struct {
	ctx common.Context
	// Using arrays+mutex instead of channels because we need to contantly read to re-evaluate state on polling
	// _ prefix reminder to use mutex when accessing
	_unscheduledTasks []defs.TaskDefinition // tasks that are not yet scheduled for execution
	_scheduledTasks   []defs.TaskDefinition // tasks that are scheduled for execution, effectively "running"
	_completedTasks   []defs.TaskDefinition // tasks that have completed execution whether successfully or not
	_failedTasks      []defs.TaskDefinition // tasks that have failed execution (these tasks also in _completedTasks)
	// Using one mutex for all above just for simplicity sake
	mutex sync.RWMutex
}

func NewScheduler(ctx *common.Context, targs common.TaskerArgs) Scheduler {
	// TODO: Implement filtering ws tasks based on targs
	return Scheduler{
		ctx:               *ctx,
		_unscheduledTasks: ctx.GetAllTaskDefs(),
		_scheduledTasks:   []defs.TaskDefinition{},
		_completedTasks:   []defs.TaskDefinition{},
		mutex:             sync.RWMutex{},
	}
}

//
// START: WRITERS SECTION (use write lock)
//

// MarkScheduled marks a task as scheduled.
// lock: r/w
func (s *Scheduler) MarkScheduled(task defs.TaskDefinition) {
	log.Debug("marking tasks scheduled: ", task.Id)

	// Atomic: must remove from unscheduled and add to scheduled in the same lock
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s._unscheduledTasks = s.removeFromUnscheduled(task)
	s._scheduledTasks = append(s._scheduledTasks, task)
}

// MarkCompleted marks a task as completed.
// lock: r/w
func (s *Scheduler) MarkCompleted(task defs.TaskDefinition) {
	log.Debug("marking task completed: ", task.Id)

	// Atomic: must remove from scheduled and add to completed in the same lock
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s._scheduledTasks = s.removeFromScheduled(task)
	s._completedTasks = append(s._completedTasks, task)
}

// MarkFailed marks a task as failed.
// lock: r/w
func (s *Scheduler) MarkFailed(task defs.TaskDefinition) {
	log.Debug("marking task failed: ", task.Id)

	// Atomic: must remove from scheduled and add to completed in the same lock
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s._scheduledTasks = s.removeFromScheduled(task)
	s._completedTasks = append(s._completedTasks, task)
	s._failedTasks = append(s._failedTasks, task)
}

// AnyFailed returns true if any tasks have failed.
// lock: r
func (s *Scheduler) AnyFailed() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s._failedTasks) != 0
}

//
// END: WRITERS SECTION
//

//
// START: READER SECTION (use read lock)
//

// GetAllUnscheduler returns all tasks that are not yet scheduled.
// lock: r
func (s *Scheduler) GetAllUnscheduled() []defs.TaskDefinition {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	// Return copy to avoid caller modifying internal state
	return append([]defs.TaskDefinition{}, s._unscheduledTasks...)
}

// GetAllSchedulable returns all NEW tasks that are ready to be scheduled.
// lock: r
func (s *Scheduler) GetAllSchedulable() []defs.TaskDefinition {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if !s.anySchedulable() {
		return []defs.TaskDefinition{}
	}

	newSchedulableTasks := []defs.TaskDefinition{}
	for _, task := range s._unscheduledTasks {
		if s.areDepsCompleted(task) {
			newSchedulableTasks = append(newSchedulableTasks, task)
		}
	}
	return newSchedulableTasks
}

// AllComplete returns true if there are no more tasks unsccheduled or scheduled (only completed).
// lock: r
func (s *Scheduler) AllComplete() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s._unscheduledTasks) == 0 && len(s._scheduledTasks) == 0
}

// IsDeadlocked returns true if the scheduler seems to be deadlocked.
// lock: r
func (s *Scheduler) IsDeadlocked() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// * No new tasks can be scheduled
	// * Still tasks left to schedule
	// * No scheduled tasks (so nothing running)
	// Aka: can't proceed and no progress is being made
	isDeadlock := !s.anySchedulable() &&
		len(s._unscheduledTasks) != 0 &&
		len(s._scheduledTasks) == 0

	if isDeadlock {
		unscheduledTaskIds := []string{}
		for _, task := range s._unscheduledTasks {
			unscheduledTaskIds = append(unscheduledTaskIds, string(task.Id))
		}
		log.Error(
			"scheduler is deadlocked! Some tasks still unscheduled while no tasks can be scheduled. Remaining unscheduled tasks:",
			"\n\t* "+strings.Join(unscheduledTaskIds, "\n\t* "),
		)
		return true
	}
	return false
}

//
// END: READER SECTION
//

//
// START: INTERNAL SECTION (depends on caller for read lock [no writers])
//
// It's easier to reason about lock usage by having utils NOT lock by themselves but depend on the public callers.
//

// lock: depends on caller read lock
func (s *Scheduler) anySchedulable() bool {
	for _, task := range s._unscheduledTasks {
		if s.areDepsCompleted(task) {
			return true
		}
	}
	return false
}

// lock: depends on caller read lock
func (s *Scheduler) areDepsCompleted(task defs.TaskDefinition) bool {
	// No deps
	if len(task.Deps) == 0 {
		return true
	}

	// Some deps
	for _, depId := range task.Deps {
		currCompleted := false

		// Find if this dep is completed
		for _, completedTask := range s._completedTasks {
			if depId == completedTask.Id {
				currCompleted = true
			}
		}

		// At least one dep isn't complete
		if !currCompleted {
			return false
		}
	}

	// All complete!
	return true
}

// lock: depends on caller read lock
func (s *Scheduler) removeFromUnscheduled(task defs.TaskDefinition) []defs.TaskDefinition {
	newUnscheduledTasks := []defs.TaskDefinition{}
	for _, unscheduledTask := range s._unscheduledTasks {
		if unscheduledTask.Id != task.Id {
			newUnscheduledTasks = append(newUnscheduledTasks, unscheduledTask)
		} else {
			log.Debug("removing task from unscheduled: ", task.Id)
		}
	}
	return newUnscheduledTasks
}

// lock: depends on caller read lock
func (s *Scheduler) removeFromScheduled(task defs.TaskDefinition) []defs.TaskDefinition {
	newScheduledTasks := []defs.TaskDefinition{}
	for _, scheduledTask := range s._scheduledTasks {
		if scheduledTask.Id != task.Id {
			newScheduledTasks = append(newScheduledTasks, scheduledTask)
		} else {
			log.Debug("removing task from scheduled: ", task.Id)
		}
	}
	return newScheduledTasks
}

//
// END: INTERNAL SECTION
//
