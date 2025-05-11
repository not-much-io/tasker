package main

import (
	"fmt"
	"inference-tasker/lib/defs"
	"inference-tasker/lib/tasker"
	"inference-tasker/lib/tasker/common"
	"inference-tasker/lib/tasker/scheduler"
	"inference-tasker/lib/tasker/skipper"
	"os"
	"strconv"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
)

func main() {
	// log.SetLevel(log.InfoLevel)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		PadLevelText:  true,
		FullTimestamp: false,
	})
	ctxLogger := log.WithField("bin", os.Args[0])

	ws := defs.InitWorkspace(ctxLogger)
	ws.Dump() // dump it right away to be able to debug if something goes wrong
	ctx := common.NewContext(ctxLogger, ws)
	args := common.ParseTaskerArgs(ctx, os.Args[1:])

	// non-std tasks need scheduler/runner
	scheduler := scheduler.NewScheduler(&ctx, args)
	skipper := skipper.NewSkipper(&ctx, args)
	runner := tasker.NewRunner(&scheduler, skipper)

	// Will block until all tasks are done or deadlock is reached
	fmt.Println(buildReport(runner.Start(&ctx)))
}

func longestNonTaskCellElement(result tasker.RunnerRunResult) string {
	longestCellElement := strconv.FormatInt(result.Taken(), 10)
	if len(longestCellElement) < len("Taken") {
		longestCellElement = "Taken"
	}
	return longestCellElement
}

func longestTaskCellElement(result tasker.RunnerRunResult) string {
	longestCellElement := ""
	for _, taskResult := range result.TaskRunResults {
		if len(taskResult.TaskId) > len(longestCellElement) {
			longestCellElement = string(taskResult.TaskId)
		}
	}
	return longestCellElement
}

func buildReportHeader(nonTaskCellPadding string, taskCellPadding string) string {
	header := "| \u23F5 "
	header += fmt.Sprintf("|%"+nonTaskCellPadding+"s", "Start")
	header += fmt.Sprintf("|%"+nonTaskCellPadding+"s", "End")
	header += fmt.Sprintf("|%"+nonTaskCellPadding+"s", "Taken")
	header += fmt.Sprintf("| %-"+taskCellPadding+"s|", "Task")
	return header
}

func buildReportSeparator(header string) string {
	separator := ""
	for i := 0; i < len(header); i++ {
		separator += "-"
	}
	return separator
}

func buildReport(result tasker.RunnerRunResult) string {
	report := ""

	longestNonTaskCellElement := longestNonTaskCellElement(result)
	nonTaskCellPadding := strconv.Itoa(len(longestNonTaskCellElement) + 2) // +2 for ms postfix
	longestTaskCellElement := longestTaskCellElement(result)
	taskCellPadding := strconv.Itoa(len(longestTaskCellElement))

	header := buildReportHeader(nonTaskCellPadding, taskCellPadding)
	separator := buildReportSeparator(header)

	report += separator + "\n"
	report += header + "\n"
	report += separator

	for _, taskResult := range result.TaskRunResults {
		report += "\n"

		if taskResult.Result == tasker.Success {
			report += "| " + color.GreenString("%s", "\u2713") + " "
		} else if taskResult.Result == tasker.Failure {
			report += "| " + color.RedString("%s", "\u2717") + " "
		} else if taskResult.Result == tasker.NotRun {
			report += "| " + color.YellowString("%s", "\u26A0") + " "
		} else if taskResult.Result == tasker.Cached {
			report += "| " + color.BlueString("%s", "\u267A") + " "
		} else {
			log.Fatal("Unknown task result")
		}

		report += fmt.Sprintf("|%"+nonTaskCellPadding+"s",
			taskResult.StartTimeSinceRunBegin(result),
		)
		report += fmt.Sprintf("|%"+nonTaskCellPadding+"s",
			taskResult.EndTimeSinceRunBegin(result),
		)
		report += fmt.Sprintf("|%"+nonTaskCellPadding+"s",
			taskResult.Taken(),
		)
		report += fmt.Sprintf("| %-"+taskCellPadding+"s|", string(taskResult.TaskId))
	}

	report += "\n" + separator

	return report
}
