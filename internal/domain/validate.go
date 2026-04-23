package domain

import (
	"fmt"
	"strings"
)

func ParseBusyLevel(s string) (BusyLevel, error) {
	switch BusyLevel(strings.ToLower(strings.TrimSpace(s))) {
	case BusyQuiet:
		return BusyQuiet, nil
	case BusyModerate:
		return BusyModerate, nil
	case BusyBusy:
		return BusyBusy, nil
	case BusyClosed:
		return BusyClosed, nil
	default:
		return "", fmt.Errorf("invalid busy_level")
	}
}

func ParseStatusSource(s string) (StatusSource, error) {
	switch StatusSource(strings.ToLower(strings.TrimSpace(s))) {
	case SourceMerchant:
		return SourceMerchant, nil
	case SourceOperator:
		return SourceOperator, nil
	case SourceIntegration:
		return SourceIntegration, nil
	case SourceCrowd:
		return SourceCrowd, nil
	default:
		return "", fmt.Errorf("invalid source")
	}
}

func ParseTaskType(s string) (TaskType, error) {
	switch TaskType(strings.ToLower(strings.TrimSpace(s))) {
	case TaskTypeQueueForMe:
		return TaskTypeQueueForMe, nil
	default:
		return "", fmt.Errorf("invalid task_type")
	}
}

func ParseTaskStatus(s string) (TaskStatus, error) {
	switch TaskStatus(strings.ToLower(strings.TrimSpace(s))) {
	case TaskStatusCreated:
		return TaskStatusCreated, nil
	case TaskStatusMatching:
		return TaskStatusMatching, nil
	case TaskStatusAccepted:
		return TaskStatusAccepted, nil
	case TaskStatusArrived:
		return TaskStatusArrived, nil
	case TaskStatusQueuing:
		return TaskStatusQueuing, nil
	case TaskStatusCompleted:
		return TaskStatusCompleted, nil
	case TaskStatusFailed:
		return TaskStatusFailed, nil
	case TaskStatusCancelled:
		return TaskStatusCancelled, nil
	default:
		return "", fmt.Errorf("invalid task_status")
	}
}

func ParseRunnerStatus(s string) (RunnerStatus, error) {
	switch RunnerStatus(strings.ToLower(strings.TrimSpace(s))) {
	case RunnerCandidate:
		return RunnerCandidate, nil
	case RunnerApproved:
		return RunnerApproved, nil
	case RunnerProbation:
		return RunnerProbation, nil
	case RunnerActive:
		return RunnerActive, nil
	case RunnerSuspended:
		return RunnerSuspended, nil
	case RunnerOffboarded:
		return RunnerOffboarded, nil
	default:
		return "", fmt.Errorf("invalid runner_status")
	}
}

func ParseReporterType(s string) (ReporterType, error) {
	switch ReporterType(strings.ToLower(strings.TrimSpace(s))) {
	case ReporterRunner:
		return ReporterRunner, nil
	case ReporterUser:
		return ReporterUser, nil
	case ReporterOperator:
		return ReporterOperator, nil
	default:
		return "", fmt.Errorf("invalid reporter_type")
	}
}
