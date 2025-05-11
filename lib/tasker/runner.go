package tasker

import (
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker/common"
	"inference-tasker/lib/tasker/scheduler"
	"inference-tasker/lib/tasker/skipper"
	"inference-tasker/lib/tasker/tasks"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// TODO: Add option to limit parallelism
type Runner struct {
	// Queue of tasks to execute
	Queue chan *tasks.Task
	// Scheduler to decide which task to execute and signal when all complete
	Scheduler *scheduler.Scheduler
	// TODO
	Skipper skipper.Skipper
	// Final results of runner run
	RunResults      *RunnerRunResult
	runResultsMutex sync.Mutex
}

type RunnerRunResult struct {
	StartTime      time.Time
	EndTime        time.Time
	TaskRunResults []TaskRunResult
}

func (rr RunnerRunResult) Taken() int64 {
	return rr.EndTime.Sub(rr.StartTime).Milliseconds()
}

type taskRunResult string

const (
	Success taskRunResult = "success"
	Failure taskRunResult = "failure"
	Cached  taskRunResult = "cached"
	NotRun  taskRunResult = "not-run"
)

type TaskRunResult struct {
	TaskId    defs.TaskId
	StartTime time.Time
	EndTime   time.Time
	Result    taskRunResult
}

// TODO: These are report concerns, should be moved there

func (trr TaskRunResult) StartTimeSinceRunBegin(rr RunnerRunResult) string {
	if trr.StartTime.IsZero() {
		return "-"
	}
	return strconv.FormatInt(trr.StartTime.Sub(rr.StartTime).Milliseconds(), 10) + "ms"
}

func (trr TaskRunResult) EndTimeSinceRunBegin(rr RunnerRunResult) string {
	if trr.EndTime.IsZero() {
		return "-"
	}
	return strconv.FormatInt(trr.EndTime.Sub(rr.StartTime).Milliseconds(), 10) + "ms"
}

func (trr TaskRunResult) Taken() string {
	if trr.EndTime.IsZero() || trr.StartTime.IsZero() {
		return "-"
	}
	return strconv.FormatInt(trr.EndTime.Sub(trr.StartTime).Milliseconds(), 10) + "ms"
}

func NewRunner(scheduler *scheduler.Scheduler, skipper skipper.Skipper) Runner {
	return Runner{
		Queue:           make(chan *tasks.Task),
		Skipper:         skipper,
		Scheduler:       scheduler,
		RunResults:      nil,
		runResultsMutex: sync.Mutex{},
	}
}

func (r *Runner) Start(ctx *common.Context) RunnerRunResult {
	log.Debug("starting runner")

	r.RunResults = &RunnerRunResult{
		StartTime:      time.Now(),
		EndTime:        time.Time{},
		TaskRunResults: []TaskRunResult{},
	}

	// Just cleanup. Signaling the end of work is not this, but instead done via nil task
	defer close(r.Queue)

	// Spawn a goroutine that queue tasks based on scheduler's decisions
	go func() {
		defer func() {
			log.Debug("signalling to dequeueing loop that all tasks complete")
			r.Queue <- nil
		}()
		for {
			// We always act as any failure is fatal
			if r.Scheduler.AnyFailed() {
				log.Error("failed tasks, exiting queueing loop!")
				break
			}
			if r.Scheduler.IsDeadlocked() {
				log.Error("deadlocked, exiting queueing loop!")
				break
			}
			if r.Scheduler.AllComplete() {
				log.Info("all tasks completed, exiting queueing loop!")
				break
			}

			// Queue anything we can
			newScheduledTaskDefs := r.Scheduler.GetAllSchedulable()
			for _, taskDef := range newScheduledTaskDefs {
				newTask := tasks.NewTask(*ctx, taskDef)
				log.Debug("queueing task: ", newTask.TaskDef.Id)
				r.Scheduler.MarkScheduled(taskDef)
				r.Queue <- &newTask
				log.Debug("queued task: ", newTask.TaskDef.Id)
			}
		}
	}()

	// Block Start() on this dequeueing tasks until above goroutine signals all tasks complete via nil task
	for {
		task := <-r.Queue
		if task == nil {
			log.Debug("signal from queueing loop: all tasks completed, exiting dequeueing loop")
			break
		}
		log.Debug("dequeued task: ", task.TaskDef.Id)

		if r.Skipper.ShouldSkip(*task) {
			log.Debug("skipping task: ", task.TaskDef.Id)
			r.Scheduler.MarkCompleted(task.TaskDef) // worth modeling skipped / completed separately?
			r.runResultsMutex.Lock()
			r.RunResults.TaskRunResults = append(r.RunResults.TaskRunResults, TaskRunResult{
				TaskId:    task.TaskDef.Id,
				StartTime: time.Time{},
				EndTime:   time.Time{},
				Result:    Cached,
			})
			r.runResultsMutex.Unlock()
			continue
		}

		// Spawn a goroutine to run the task - we run parallel by default
		// The scheduler takes care of dependency resolution and ordering
		go func() {
			runnerResult := TaskRunResult{
				TaskId:    task.TaskDef.Id,
				StartTime: time.Now(),
				EndTime:   time.Time{},
			}

			_, err := task.Run(*ctx)
			if err != nil {
				r.Scheduler.MarkFailed(task.TaskDef)
				runnerResult.Result = Failure
			} else {
				r.Scheduler.MarkCompleted(task.TaskDef)
				runnerResult.Result = Success
			}
			runnerResult.EndTime = time.Now()

			r.runResultsMutex.Lock()
			defer r.runResultsMutex.Unlock()
			r.RunResults.TaskRunResults = append(r.RunResults.TaskRunResults, runnerResult)
		}()
	}

	for _, task := range r.Scheduler.GetAllUnscheduled() {
		r.RunResults.TaskRunResults = append(r.RunResults.TaskRunResults, TaskRunResult{
			TaskId:    task.Id,
			StartTime: time.Time{},
			EndTime:   time.Time{},
			Result:    NotRun,
		})
	}

	r.RunResults.EndTime = time.Now()
	return *r.RunResults
}
