package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"

	"github.com/jadegopher/trabbit/internal/log"
)

type config struct {
	url      string
	queue    string
	interval time.Duration
	workers  int
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.url, "url", "amqp://guest:guest@localhost:5672/", "RabbitMQ AMQP URL")
	flag.StringVar(&cfg.queue, "queue", "stu-queue", "Queue name (query)")
	flag.DurationVar(&cfg.interval, "interval", 1*time.Second, "Interval between action cycles")
	flag.IntVar(&cfg.workers, "workers", 5, "Number of concurrent workers")
	flag.Parse()

	// Initialize Registry with default actions
	registry := NewRegistry()
	registry.Register(PublishAction{})
	registry.Register(FetchAction{})
	registry.Register(ConsumeAction{})
	//registry.Register(PurgeAction{})

	if err := run(cfg, registry); err != nil {
		log.Error("Stu failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config, registry *Registry) error {
	conn, err := amqp.Dial(cfg.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	// Initial setup channel to declare the queue
	setupCh, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a setup channel: %w", err)
	}
	_, err = setupCh.QueueDeclare(
		cfg.queue, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	setupCh.Close()
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	log.Info("Stu framework started", "queue", cfg.queue, "interval", cfg.interval, "workers", cfg.workers)

	for i := 0; i < cfg.workers; i++ {
		workerID := i
		g.Go(
			func() error {
				ch, err := conn.Channel()
				if err != nil {
					return fmt.Errorf("worker %d failed to open a channel: %w", workerID, err)
				}
				defer ch.Close()

				ticker := time.NewTicker(cfg.interval)
				defer ticker.Stop()

				for {
					select {
					case <-ticker.C:
						action := registry.GetRandom()
						if action == nil {
							// This might happen if all actions reached their parallel limit
							continue
						}

						n := 1
						if action.CanRepeat() {
							n = rand.Intn(1000) + 1 // 1 to 100
						}

						log.Info("Worker performing action", "worker", workerID, "action", action.Name(), "times", n)

						registry.StartAction(action)
						for j := 0; j < n; j++ {
							err := action.Execute(ctx, ch, cfg.queue)
							if err != nil {
								log.ErrorContext(
									ctx,
									"Action execution notice",
									"worker",
									workerID,
									"action",
									action.Name(),
									"error",
									err,
								)
							}
						}
						registry.EndAction(action)

					case <-ctx.Done():
						log.Info("Worker stopping...", "worker", workerID)
						return nil
					}
				}
			},
		)
	}

	return g.Wait()
}
