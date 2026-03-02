package services

import (
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"

	"github.com/tupyy/dcm-agent/internal/models"
)

type Executor struct {
	updateInterval time.Duration
	mu             sync.Mutex
	scheduler      *scheduler.Scheduler[any]
	workUnits      []models.WorkUnit[any]
	dispatchCh     chan models.WorkUnit[any] // received, not owned
	statusChan     chan models.WorkStatus[any]
	close          chan any
}

func NewExecutor(s *scheduler.Scheduler[any], dispatchCh chan models.WorkUnit[any]) (*Executor, chan models.WorkStatus[any]) {
	statusChan := make(chan models.WorkStatus[any], 100)
	svc := &Executor{
		updateInterval: 2 * time.Second,
		scheduler:      s,
		workUnits:      make([]models.WorkUnit[any], 0),
		dispatchCh:     dispatchCh,
		statusChan:     statusChan,
	}

	go svc.run()

	zap.S().Named("executor").Info("service started")

	return svc, statusChan
}

func (t *Executor) Stop() {
	t.mu.Lock()
	closeCh := t.close
	statusCh := t.statusChan
	t.statusChan = nil
	t.mu.Unlock()

	if closeCh != nil {
		closeCh <- struct{}{}
	}
	// close statusChan to inform Watcher (if not already closed)
	if statusCh != nil {
		close(statusCh)
	}
}

func (t *Executor) run() {
	t.close = make(chan any, 1)
	defer func() {
		zap.S().Named("executor").Info("service stopped")
		t.close = nil
	}()

	interval := t.updateInterval

	for {
		select {
		case wu, ok := <-t.dispatchCh:
			if !ok {
				zap.S().Named("executor").Info("dispatch channel closed, no more work units")
				// FIX: wait for the work in progress.
				t.mu.Lock()
				if t.statusChan != nil {
					close(t.statusChan)
					t.statusChan = nil
				}
				t.mu.Unlock()
				return
			}
			t.workUnits = append(t.workUnits, wu)
		case <-t.close:
			return
		case <-time.After(interval):
		}

		// loop through workunits and start those without future and without result
		for i := range t.workUnits {
			wu := &t.workUnits[i]

			// start work units that have no future and no result yet
			if wu.Future == nil && wu.Result == nil {
				wu.Future = t.scheduler.AddWork(wu.Fn)
				zap.S().Named("executor").Debug("started work unit")
				t.statusChan <- wu.Status()
				continue
			}

			select {
			case result := <-wu.Future.C():
				wu.Result = &result
				wu.Future = nil
				zap.S().Named("executor").Debugw("work completed", "success", result.Err == nil)
				t.statusChan <- wu.Status()
			default:
			}
		}
	}
}
