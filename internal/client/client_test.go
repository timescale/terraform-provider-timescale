package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

const invalidTokenBody = `{"errors":[{"message":"unauthorized: Unauthorized. Invalid token provided.","extensions":{"status":401}}]}`

// tokenGateway accepts only the latest token it issued from GetJWTForClientCredentials, like the gateway does once a JWT expires.
type tokenGateway struct {
	mu        sync.Mutex
	valid     string
	issued    atomic.Int32
	rejected  atomic.Int32
	vpcLookup atomic.Int32
}

func (g *tokenGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OperationName string `json:"operationName"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	w.Header().Set("Content-Type", "application/json")

	g.mu.Lock()
	defer g.mu.Unlock()
	if body.OperationName == "GetJWTForClientCredentials" {
		g.valid = fmt.Sprintf("jwt-%d", g.issued.Add(1))
		_, _ = fmt.Fprintf(w, `{"data":{"getJWTForClientCredentials":%q}}`, g.valid)
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+g.valid {
		g.rejected.Add(1)
		_, _ = fmt.Fprint(w, invalidTokenBody)
		return
	}
	g.vpcLookup.Add(1)
	_, _ = fmt.Fprint(w, `{"data":{"getVpc":{"id":"7","status":"CREATED"}}}`)
}

func newCCClient(t *testing.T, g *tokenGateway) *Client {
	t.Helper()
	srv := httptest.NewServer(g)
	t.Cleanup(srv.Close)
	c := newTestClient(srv.URL)
	require.NoError(t, JWTFromCC(c, "access", "secret"))
	return c
}

// expire makes the gateway reject every token issued so far.
func (g *tokenGateway) expire() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.valid = "expired"
}

func TestDo_RefreshesExpiredToken(t *testing.T) {
	g := &tokenGateway{}
	c := newCCClient(t, g)
	g.expire()

	vpc, err := c.GetVPCByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "CREATED", vpc.Status)
	require.Equal(t, int32(2), g.issued.Load(), "one token at configure time, one refresh")
	require.Equal(t, int32(1), g.rejected.Load())
}

func TestDo_ConcurrentRequestsRefreshOnce(t *testing.T) {
	g := &tokenGateway{}
	c := newCCClient(t, g)
	g.expire()

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := c.GetVPCByID(context.Background(), 7)
			require.NoError(t, err)
		})
	}
	wg.Wait()
	require.Equal(t, int32(2), g.issued.Load(), "requests rejected with the same stale token must share one refresh")
	require.Equal(t, int32(10), g.vpcLookup.Load())
}

func TestDo_NoRefreshWithoutClientCredentials(t *testing.T) {
	g := &tokenGateway{valid: "other"}
	srv := httptest.NewServer(g)
	t.Cleanup(srv.Close)

	_, err := newTestClient(srv.URL).GetVPCByID(context.Background(), 7)
	require.ErrorContains(t, err, "Invalid token provided")
	require.Equal(t, int32(0), g.issued.Load(), "a static access_token can't be refreshed")
}
