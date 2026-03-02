package services

import (
	"go.uber.org/zap"

	"github.com/tupyy/dcm-agent/internal/models"
)

type Watcher struct {
	statusChan chan models.WorkStatus[any]
	close      chan any
}

func NewWatcher(statusChan chan models.WorkStatus[any]) *Watcher {
	svc := &Watcher{
		statusChan: statusChan,
		close:      make(chan any, 1),
	}

	go svc.run()

	zap.S().Named("watcher").Info("service started")

	return svc
}

func (w *Watcher) Stop() {
	if w.close != nil {
		w.close <- struct{}{}
	}
}

func (w *Watcher) run() {
	defer func() {
		zap.S().Named("watcher").Info("service stopped")
		w.close = nil
	}()

	for {
		select {
		case status, ok := <-w.statusChan:
			if !ok {
				zap.S().Named("watcher").Info("status channel closed, no more status updates")
				return
			}
			zap.S().Named("watcher").Debugw("received status update",
				"id", status.ID,
				"status", status.Status,
			)
		case <-w.close:
			return
		}
	}
}
