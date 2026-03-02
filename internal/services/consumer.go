package services

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/tupyy/dcm-agent/internal/models"
)

type ConsumerService struct {
	nc         *nats.Conn
	subject    string
	msgChan    chan *nats.Msg
	sub        *nats.Subscription
	dispatchCh chan models.WorkUnit[any]
	close      chan any
}

func NewConsumerService(nc *nats.Conn, subject string) (*ConsumerService, chan models.WorkUnit[any]) {
	msgChan := make(chan *nats.Msg)
	dispatchCh := make(chan models.WorkUnit[any])

	sub, err := nc.ChanSubscribe(subject, msgChan)
	if err != nil {
		zap.S().Named("consumer").Errorw("failed to subscribe", "subject", subject, "error", err)
		return nil, nil
	}

	c := &ConsumerService{
		nc:         nc,
		subject:    subject,
		msgChan:    msgChan,
		sub:        sub,
		dispatchCh: dispatchCh,
		close:      make(chan any),
	}

	go c.run()

	zap.S().Named("consumer").Infow("service started", "subject", subject)

	return c, dispatchCh
}

func (c *ConsumerService) Stop() {
	c.close <- struct{}{}
	close(c.dispatchCh)
}

func (c *ConsumerService) run() {
	defer func() {
		zap.S().Named("consumer").Info("service stopped")
	}()

	for {
		select {
		case msg := <-c.msgChan:
			wu := models.NewWorkUnit(func(ctx context.Context) (any, error) {
				// placeholder - to be implemented
				zap.S().Named("consumer").Debugw("processing message", "subject", msg.Subject)
				return nil, nil
			})
			c.dispatchCh <- wu
		case <-c.close:
			c.sub.Unsubscribe()
			return
		}
	}
}
