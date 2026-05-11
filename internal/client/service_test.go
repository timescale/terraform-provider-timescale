package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockGraphQLServer returns an httptest server that responds with the supplied
// GraphQL response body for any POST. The Client points its base URL here via
// TIMESCALE_DEV_URL during the test.
func mockGraphQLServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}))
}

// newTestClient builds a Client pointing at the given mock-server URL. We use
// the unexported `url` field directly because this is a white-box test in the
// same package, which keeps the production NewClient API focused.
func newTestClient(url string) *Client {
	c := NewClient("token", "proj", "test", "1.0.0")
	c.url = url
	return c
}

func TestGetService_NotFoundReturnsSentinel(t *testing.T) {
	srv := mockGraphQLServer(t, `{"errors":[{"message":"no service with that id exists"}]}`)
	defer srv.Close()

	_, err := newTestClient(srv.URL).GetService(context.Background(), "any-id")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrServiceNotFound), "expected ErrServiceNotFound, got %v", err)
}

func TestGetService_OtherErrorPassthrough(t *testing.T) {
	srv := mockGraphQLServer(t, `{"errors":[{"message":"some other API error"}]}`)
	defer srv.Close()

	_, err := newTestClient(srv.URL).GetService(context.Background(), "any-id")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrServiceNotFound), "non-not-found errors must not match the sentinel")
	require.Contains(t, err.Error(), "some other API error")
}

func TestDeleteService_NotFoundReturnsSentinel(t *testing.T) {
	srv := mockGraphQLServer(t, `{"errors":[{"message":"no service with that id exists"}]}`)
	defer srv.Close()

	_, err := newTestClient(srv.URL).DeleteService(context.Background(), "any-id")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrServiceNotFound), "expected ErrServiceNotFound, got %v", err)
}
