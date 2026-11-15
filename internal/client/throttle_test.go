package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jadegopher/trabbit/internal/app"
	"github.com/stretchr/testify/require"
)

func TestRequestDelayAppliesToEveryHTTPMethod(t *testing.T) {
	t.Parallel()

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				// A delay longer than the HTTP timeout must still allow the request.
				const delay = DefaultTimeout + time.Second
				started := time.Now()
				var sent int
				c := testClient(func(r *http.Request) (*http.Response, error) {
					sent++
					require.Equal(t, method, r.Method)
					require.Equal(t, time.Duration(sent)*delay, time.Since(started))
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
				})
				c.http.Timeout = DefaultTimeout
				c.requestDelay = delay

				require.NoError(t, c.request(context.Background(), method, "/api/test", nil, nil))
				require.NoError(t, c.request(context.Background(), method, "/api/test", nil, nil))
				require.Equal(t, 2, sent)
			})
		})
	}
}

func TestRequestDelayDisabledByDefault(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		started := time.Now()
		c := testClient(func(r *http.Request) (*http.Response, error) {
			require.Zero(t, time.Since(started))
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}"))}, nil
		})
		require.NoError(t, c.request(context.Background(), http.MethodGet, "/api/test", nil, nil))
	})
}

func TestRequestDelayHonorsCancellation(t *testing.T) {
	t.Parallel()

	for _, alreadyCanceled := range []bool{false, true} {
		name := "during wait"
		if alreadyCanceled {
			name = "before wait"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				const delay = 5 * time.Second
				const cancelAfter = time.Second
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if alreadyCanceled {
					cancel()
				} else {
					time.AfterFunc(cancelAfter, cancel)
				}

				c := testClient(func(r *http.Request) (*http.Response, error) {
					t.Error("canceled requests must not reach the transport")
					return nil, context.Canceled
				})
				c.requestDelay = delay
				started := time.Now()
				require.ErrorIs(t, c.request(ctx, http.MethodPost, "/api/test", nil, nil), context.Canceled)
				if alreadyCanceled {
					require.Zero(t, time.Since(started))
				} else {
					require.Equal(t, cancelAfter, time.Since(started))
				}
			})
		})
	}
}

func TestConnectDelayIncludesAuthentication(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		const delay = 5 * time.Second
		const timeout = time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		started := time.Now()
		service, err := ConnectWithDelay(ctx, app.Connection{URL: "http://rabbit", Username: "guest"}, delay)
		require.Nil(t, service)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, timeout, time.Since(started))
	})
}

func TestConnectRejectsNegativeDelay(t *testing.T) {
	t.Parallel()

	service, err := ConnectWithDelay(context.Background(), app.Connection{URL: "http://rabbit", Username: "guest"}, -time.Second)
	require.Nil(t, service)
	require.ErrorContains(t, err, "request delay must not be negative")
}
