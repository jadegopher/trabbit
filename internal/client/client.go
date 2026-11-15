package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/jadegopher/trabbit/internal/log"
)

const (
	DefaultTimeout                    = 10 * time.Second
	DefaultRequestDelay time.Duration = 0
)
const DefaultExchange = app.DefaultExchange

// Aliases preserve callers while the contracts live in the application layer.
type ListParameters = app.ListParameters
type CreateQueueParams = app.CreateQueueParams
type PublishOptions = app.PublishOptions
type PublishMessageProperties = app.PublishMessageProperties
type PublishResult = app.PublishResult

type Client struct {
	endpoint, username, password string
	http                         *http.Client
	requestDelay                 time.Duration
	logger                       log.Logger
}

func Connect(ctx context.Context, cfg app.Connection) (*app.Service, error) {
	return ConnectWithDelay(ctx, cfg, DefaultRequestDelay)
}

// ConnectWithDelay applies the delay to every request, including authentication.
func ConnectWithDelay(ctx context.Context, cfg app.Connection, delay time.Duration) (*app.Service, error) {
	if err := app.ValidateConnection(cfg); err != nil {
		return nil, err
	}
	if delay < 0 {
		return nil, fmt.Errorf("request delay must not be negative")
	}
	c := &Client{
		endpoint: strings.TrimRight(cfg.URL, "/"), username: cfg.Username, password: cfg.Password,
		http: &http.Client{Timeout: DefaultTimeout}, requestDelay: delay, logger: log.DefaultLogger,
	}
	if err := c.request(ctx, http.MethodGet, "/api/users/"+url.PathEscape(cfg.Username), nil, nil); err != nil {
		log.FailureContext(c.logger, ctx, "RabbitMQ connection failed", err, log.Component, "client")
		return nil, fmt.Errorf("connect: %w", err)
	}
	c.logger.InfoContext(ctx, "RabbitMQ connection established", log.Component, "client")
	return app.NewWithLogger(c, c.logger), nil
}

// request keeps cancellation and response cleanup identical for every endpoint.
func (c *Client) request(ctx context.Context, method, path string, body, result any) (requestErr error) {
	logger := c.logger
	if logger == nil {
		logger = log.DefaultLogger
	}
	started := time.Now()
	var status int
	defer func() {
		// Bodies, authorization and query values never enter transport logs.
		resource, _, _ := strings.Cut(path, "?")
		logger.DebugContext(ctx, "RabbitMQ request finished", log.Component, "client",
			"method", method, "path", resource, "status", status,
			"duration", time.Since(started), "failed", requestErr != nil)
	}()

	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	// Wait before Do so the delay does not consume the HTTP transport timeout.
	if c.requestDelay > 0 {
		timer := time.NewTimer(c.requestDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	status = resp.StatusCode
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &HTTPError{StatusCode: resp.StatusCode}
	}
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return err
}

type HTTPError struct{ StatusCode int }

func (e *HTTPError) Error() string {
	return fmt.Sprintf("RabbitMQ HTTP %d: %s", e.StatusCode, http.StatusText(e.StatusCode))
}
