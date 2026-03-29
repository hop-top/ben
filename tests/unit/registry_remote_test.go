package unit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"hop.top/ben/internal/registry"
	"hop.top/ben/internal/run"
)

func newTestRun() *run.Run {
	return newRun("test-suite", 1, time.Now())
}

func TestRemoteClient_Push_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/runs", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"remote-abc"}`))
	}))
	defer srv.Close()

	client := registry.NewRemoteClient(srv.URL)
	client.SetHTTPClient(srv.Client())

	remoteID, err := client.Push(context.Background(), newTestRun())
	require.NoError(t, err)
	assert.Equal(t, "remote-abc", remoteID)
}

func TestRemoteClient_Pull_Success(t *testing.T) {
	r1 := newRun("pull-suite", 1, time.Now())
	r2 := newRun("pull-suite", 1, time.Now().Add(time.Minute))

	payload := struct {
		Runs []*run.Run `json:"runs"`
	}{Runs: []*run.Run{r1, r2}}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/runs", r.URL.Path)
		assert.Equal(t, "pull-suite", r.URL.Query().Get("suite"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	client := registry.NewRemoteClient(srv.URL)
	client.SetHTTPClient(srv.Client())

	runs, err := client.Pull(context.Background(), "pull-suite", 10)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	assert.Equal(t, r1.RunID, runs[0].RunID)
	assert.Equal(t, r2.RunID, runs[1].RunID)
}

func TestRemoteClient_Push_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := registry.NewRemoteClient(srv.URL)
	client.SetHTTPClient(srv.Client())

	_, err := client.Push(context.Background(), newTestRun())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestRemoteClient_Pull_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"runs": [NOT VALID JSON`))
	}))
	defer srv.Close()

	client := registry.NewRemoteClient(srv.URL)
	client.SetHTTPClient(srv.Client())

	_, err := client.Pull(context.Background(), "", 10)
	require.Error(t, err)
}
