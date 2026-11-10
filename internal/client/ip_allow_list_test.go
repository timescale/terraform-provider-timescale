package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type gqlCall struct {
	Op        string         `json:"operationName"`
	Variables map[string]any `json:"variables"`
}

// newRecordingClient returns a client whose gateway replies with reply and records every request.
func newRecordingClient(t *testing.T, reply string) (*Client, *[]gqlCall) {
	t.Helper()
	var calls []gqlCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call gqlCall
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		calls = append(calls, call)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(srv.URL)
	c.projectID = "proj"
	return c, &calls
}

func TestGetIPAllowList(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"getIPAllowLists":[
		{"listId":1,"description":"a","cidrBlocks":["10.0.0.0/8"],"associatedServices":null},
		{"listId":2,"description":"b","cidrBlocks":["1.2.3.4/32"],"associatedServices":["svc1"]}]}}`)

	l, err := c.GetIPAllowList(context.Background(), 2)
	require.NoError(t, err)
	require.Equal(t, &IPAllowList{ListID: 2, Description: "b", CIDRBlocks: []string{"1.2.3.4/32"}, AssociatedServices: []string{"svc1"}}, l)
	require.Equal(t, "proj", (*calls)[0].Variables["projectId"])

	l, err = c.GetIPAllowList(context.Background(), 99)
	require.NoError(t, err)
	require.Nil(t, l, "missing list must be reported as nil, not an error")
}

func TestConfigureIPAllowList(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"configureIPAllowList":{"listId":5}}}`)

	id, err := c.ConfigureIPAllowList(context.Background(), nil, "d", []string{"1.1.1.1/32"})
	require.NoError(t, err)
	require.Equal(t, int64(5), id)
	require.NotContains(t, (*calls)[0].Variables, "listId", "create must omit listId")

	listID := int64(5)
	_, err = c.ConfigureIPAllowList(context.Background(), &listID, "d", []string{"1.1.1.1/32"})
	require.NoError(t, err)
	require.EqualValues(t, 5, (*calls)[1].Variables["listId"])
}

func TestDeleteIPAllowList_SendsNullCIDRs(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"configureIPAllowList":{"listId":5}}}`)
	require.NoError(t, c.DeleteIPAllowList(context.Background(), 5))
	v := (*calls)[0].Variables
	require.Contains(t, v, "cidrBlocks")
	require.Nil(t, v["cidrBlocks"])
	require.EqualValues(t, 5, v["listId"])
}

func TestBindUnbindIPAllowList(t *testing.T) {
	c, calls := newRecordingClient(t, `{"data":{"x":"ok"}}`)
	require.NoError(t, c.BindIPAllowList(context.Background(), "svc", 3))
	require.NoError(t, c.UnbindIPAllowList(context.Background(), "svc", 3))
	require.Equal(t, "BindIPAllowList", (*calls)[0].Op)
	require.Equal(t, "UnbindIPAllowList", (*calls)[1].Op)
	require.Equal(t, "svc", (*calls)[1].Variables["serviceId"])
	require.EqualValues(t, 3, (*calls)[1].Variables["listId"])
}

func TestIPAllowList_APIError(t *testing.T) {
	c, _ := newRecordingClient(t, `{"errors":[{"message":"invalid cidr"}]}`)
	_, err := c.ConfigureIPAllowList(context.Background(), nil, "d", []string{"bad"})
	require.ErrorContains(t, err, "invalid cidr")
}
