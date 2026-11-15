package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Action represents a single task that stu can perform.
type Action interface {
	// Name returns the display name of the action.
	Name() string
	// Execute performs the action once.
	Execute(ctx context.Context, ch *amqp.Channel, queue string) error
	// CanRepeat returns true if the action can be performed multiple times in a row.
	CanRepeat() bool
	// MaxParallel returns the maximum number of instances of this action that can run simultaneously.
	// 0 means unlimited.
	MaxParallel() int
}

// Registry stores all available actions for stu and tracks their active executions.
type Registry struct {
	actions []Action
	mu      sync.Mutex
	active  map[string]int
}

func NewRegistry() *Registry {
	return &Registry{
		actions: []Action{},
		active:  make(map[string]int),
	}
}

func (r *Registry) Register(a Action) {
	r.actions = append(r.actions, a)
}

func (r *Registry) GetRandom() Action {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.actions) == 0 {
		return nil
	}

	// Try multiple times to find an action that hasn't reached its parallel limit
	// We shuffle indices to avoid bias
	indices := rand.Perm(len(r.actions))
	for _, idx := range indices {
		a := r.actions[idx]
		limit := a.MaxParallel()
		if limit == 0 || r.active[a.Name()] < limit {
			return a
		}
	}

	return nil
}

func (r *Registry) StartAction(a Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[a.Name()]++
}

func (r *Registry) EndAction(a Action) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[a.Name()]--
}

// --- Concrete Actions ---

type PublishAction struct{}

func (p PublishAction) Name() string     { return "Publish" }
func (p PublishAction) CanRepeat() bool  { return true }
func (p PublishAction) MaxParallel() int { return 0 } // Unlimited
func (p PublishAction) Execute(ctx context.Context, ch *amqp.Channel, queue string) error {
	body := fmt.Sprintf("Event from Stu at %s", fmt.Sprint(rand.Intn(1000)))
	return ch.PublishWithContext(
		ctx,
		"",    // exchange
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		},
	)
}

type FetchAction struct{}

func (f FetchAction) Name() string     { return "Fetch" }
func (f FetchAction) CanRepeat() bool  { return true }
func (f FetchAction) MaxParallel() int { return 0 } // Unlimited
func (f FetchAction) Execute(ctx context.Context, ch *amqp.Channel, queue string) error {
	_, _, err := ch.Get(queue, true)
	if err != nil {
		return err
	}

	return nil
}

type PurgeAction struct{}

func (p PurgeAction) Name() string     { return "Purge" }
func (p PurgeAction) CanRepeat() bool  { return false } // Purge is usually a one-time thing per cycle
func (p PurgeAction) MaxParallel() int { return 1 }
func (p PurgeAction) Execute(ctx context.Context, ch *amqp.Channel, queue string) error {
	_, err := ch.QueuePurge(queue, false)
	return err
}

type ConsumeAction struct{}

func (c ConsumeAction) Name() string     { return "Consume" }
func (c ConsumeAction) CanRepeat() bool  { return false }
func (c ConsumeAction) MaxParallel() int { return 2 }
func (c ConsumeAction) Execute(ctx context.Context, ch *amqp.Channel, queue string) error {
	consumeTime := rand.Intn(60) + 1
	ctx, cancel := context.WithTimeout(ctx, time.Duration(consumeTime)*time.Second)
	defer cancel()

	// Set QoS to limit the number of unacknowledged messages on the channel.
	// global=true ensures the limit is per-channel, not per-consumer.
	err := ch.Qos(10000, 0, true)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	tag := fmt.Sprintf("stu-consumer-%d-%d", time.Now().UnixNano(), rand.Intn(1000))
	msgs, err := ch.Consume(
		queue,
		tag,   // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	// Ensure consumer is cancelled when we are done
	defer func() {
		_ = ch.Cancel(tag, false)
	}()

	for {
		select {
		case m, ok := <-msgs:
			if !ok {
				return nil
			}
			time.Sleep(10 * time.Millisecond)
			if err := m.Ack(true); err != nil {
				return err
			}

		case <-ctx.Done():
			// Stop new messages from arriving
			_ = ch.Cancel(tag, false)

			// Drain the remaining messages in the library's buffer and return them to RMQ
			// This clears the 'Unacked' status for messages we didn't get to process.
			select {
			case m, ok := <-msgs:
				if !ok {
					return nil
				}
				_ = m.Nack(true, true) // Requeue=true

				return nil
			default:
				return nil
			}

		}
	}
}
