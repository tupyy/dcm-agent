package models

import (
	"github.com/google/uuid"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

type WorkStatusType string

const (
	WorkStatusPending WorkStatusType = "pending"
	WorkStatusRunning WorkStatusType = "running"
	WorkStatusDone    WorkStatusType = "done"
)

type WorkUnit[T any] struct {
	ID     string
	Fn     scheduler.Work[T]
	Future *scheduler.Future[scheduler.Result[T]]
	Result *scheduler.Result[T]
}

func NewWorkUnit[T any](fn scheduler.Work[T]) WorkUnit[T] {
	return WorkUnit[T]{
		ID: uuid.New().String(),
		Fn: fn,
	}
}

func (wu *WorkUnit[T]) Status() WorkStatus[T] {
	switch {
	case wu.Result != nil:
		return WorkStatus[T]{
			ID:     wu.ID,
			Status: WorkStatusDone,
			Result: wu.Result,
		}
	case wu.Future != nil:
		return WorkStatus[T]{
			ID:     wu.ID,
			Status: WorkStatusRunning,
		}
	default:
		return WorkStatus[T]{
			ID:     wu.ID,
			Status: WorkStatusPending,
		}
	}
}

type WorkStatus[T any] struct {
	ID     string
	Status WorkStatusType
	Result *scheduler.Result[T]
}
