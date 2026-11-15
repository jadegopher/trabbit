// Package app contains operations shared by interactive and non-interactive clients.
package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jadegopher/trabbit/internal/domain"
	"github.com/jadegopher/trabbit/internal/log"
)

const DefaultExchange = "amq.default"

type Connection struct{ URL, Username, Password string }
type Connector func(context.Context, Connection) (*Service, error)

type ListParameters struct {
	Page, PageSize int
	VHost, Name    string
	UseRegex       bool
}
type CreateQueueParams struct {
	VHost, Name         string
	Durable, AutoDelete bool
	Arguments           map[string]interface{}
}

type CreateBindingParams struct {
	VHost, Queue, Source, RoutingKey string
	Arguments                        map[string]interface{}
}
type PublishMessageProperties struct{ Properties, Headers map[string]interface{} }
type PublishOptions struct {
	RoutingKey, Payload, PayloadEncoding string
	Properties                           PublishMessageProperties
}
type PublishResult struct {
	Routed bool `json:"routed"`
}

type Backend interface {
	ListQueues(context.Context, ListParameters) (domain.PagedQueues, error)
	CreateQueue(context.Context, CreateQueueParams) error
	DeleteQueue(context.Context, string, string) error
	PurgeQueue(context.Context, string, string) error
	ListVHosts(context.Context) (domain.VHosts, error)
	GetQueueDetailedStats(context.Context, string, string, time.Duration) (domain.QueueDetailedStats, error)
	Publish(context.Context, string, string, PublishOptions) (PublishResult, error)
	ListQueueBindings(context.Context, string, string) ([]domain.QueueBinding, error)
	CreateQueueBinding(context.Context, CreateBindingParams) error
	DeleteQueueBinding(context.Context, domain.QueueBinding) error
}

type Service struct {
	backend Backend
	logger  log.Logger
}

func New(backend Backend) *Service { return NewWithLogger(backend, log.DefaultLogger) }

func NewWithLogger(backend Backend, logger log.Logger) *Service {
	return &Service{backend: backend, logger: logger}
}

type OperationError struct {
	Operation, VHost, Name string
	Err                    error
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("%s %s/%s: %v", e.Operation, e.VHost, e.Name, e.Err)
}

func (e *OperationError) Unwrap() error { return e.Err }

// The service records each backend failure once, with operation identity. UI
// owners present the returned error; the transport adds only debug diagnostics.
func (s *Service) failure(ctx context.Context, op, vhost, name string, err error) error {
	if err == nil {
		return nil
	}
	attrs := []any{log.Component, "app", "operation", op, "vhost", vhost}
	// A list name is a query filter rather than a resource identity.
	if op != "list queues" {
		attrs = append(attrs, "name", name)
	}
	log.FailureContext(s.logger, ctx, "RabbitMQ operation failed", err, attrs...)
	return &OperationError{op, vhost, name, err}
}

func ValidateConnection(c Connection) error {
	u, err := url.Parse(c.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("URL must be an http or https endpoint")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

func ValidateQueue(vhost, name string) error {
	if vhost == "" {
		return fmt.Errorf("vhost is required")
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func ValidatePublish(vhost string, p PublishOptions) error {
	if vhost == "" {
		return fmt.Errorf("vhost is required")
	}
	switch p.PayloadEncoding {
	case "string":
	case "base64":
		if _, err := base64.StdEncoding.DecodeString(p.Payload); err != nil {
			return fmt.Errorf("invalid base64 payload: %w", err)
		}
	default:
		return fmt.Errorf("payload encoding must be string or base64")
	}
	return nil
}

func (s *Service) ListQueues(ctx context.Context, p ListParameters) (domain.PagedQueues, error) {
	if p.Page < 1 || p.PageSize < 1 {
		return domain.PagedQueues{}, fmt.Errorf("page and page size must be positive")
	}
	r, err := s.backend.ListQueues(ctx, p)
	return r, s.failure(ctx, "list queues", p.VHost, p.Name, err)
}

func (s *Service) ListVHosts(ctx context.Context) (domain.VHosts, error) {
	r, err := s.backend.ListVHosts(ctx)
	return r, s.failure(ctx, "list vhosts", "", "", err)
}

func (s *Service) GetQueueDetailedStats(ctx context.Context, vhost, name string, history time.Duration) (domain.QueueDetailedStats, error) {
	if err := ValidateQueue(vhost, name); err != nil {
		return domain.QueueDetailedStats{}, err
	}
	if history <= 0 || history > time.Hour {
		return domain.QueueDetailedStats{}, fmt.Errorf("history must be positive and at most one hour")
	}
	r, err := s.backend.GetQueueDetailedStats(ctx, vhost, name, history)
	return r, s.failure(ctx, "queue details", vhost, name, err)
}

func (s *Service) CreateQueue(ctx context.Context, p CreateQueueParams) error {
	if err := ValidateQueue(p.VHost, p.Name); err != nil {
		return err
	}
	return s.failure(ctx, "create queue", p.VHost, p.Name, s.backend.CreateQueue(ctx, p))
}

func (s *Service) DeleteQueue(ctx context.Context, vhost, name string) error {
	if err := ValidateQueue(vhost, name); err != nil {
		return err
	}
	return s.failure(ctx, "delete queue", vhost, name, s.backend.DeleteQueue(ctx, vhost, name))
}

func (s *Service) PurgeQueue(ctx context.Context, vhost, name string) error {
	if err := ValidateQueue(vhost, name); err != nil {
		return err
	}
	return s.failure(ctx, "purge queue", vhost, name, s.backend.PurgeQueue(ctx, vhost, name))
}

func (s *Service) Publish(ctx context.Context, vhost, exchange string, p PublishOptions) (PublishResult, error) {
	if err := ValidatePublish(vhost, p); err != nil {
		return PublishResult{}, err
	}
	r, err := s.backend.Publish(ctx, vhost, exchange, p)
	return r, s.failure(ctx, "publish", vhost, exchange, err)
}

func ValidateBinding(p CreateBindingParams) error {
	if err := ValidateQueue(p.VHost, p.Queue); err != nil {
		return err
	}
	if strings.TrimSpace(p.Source) == "" || p.Source == DefaultExchange {
		return fmt.Errorf("source exchange is required; default exchange bindings are automatic")
	}
	return nil
}

func (s *Service) ListQueueBindings(ctx context.Context, vhost, queue string) ([]domain.QueueBinding, error) {
	if err := ValidateQueue(vhost, queue); err != nil {
		return nil, err
	}
	bindings, err := s.backend.ListQueueBindings(ctx, vhost, queue)
	return bindings, s.failure(ctx, "list queue bindings", vhost, queue, err)
}

func (s *Service) CreateQueueBinding(ctx context.Context, p CreateBindingParams) error {
	if err := ValidateBinding(p); err != nil {
		return err
	}
	return s.failure(ctx, "bind queue", p.VHost, p.Queue, s.backend.CreateQueueBinding(ctx, p))
}

func (s *Service) DeleteQueueBinding(ctx context.Context, b domain.QueueBinding) error {
	if err := ValidateBinding(CreateBindingParams{VHost: b.VHost, Queue: b.Queue, Source: b.Source}); err != nil {
		return err
	}
	if b.PropertiesKey == "" {
		return fmt.Errorf("binding properties key is required")
	}
	return s.failure(ctx, "unbind queue", b.VHost, b.Queue, s.backend.DeleteQueueBinding(ctx, b))
}
