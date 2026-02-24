package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPantry_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pantry", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"p1","ingredient_id":"ing1","quantity":2.5,"unit":"cup"}]`))
	}))
	defer server.Close()

	client := &PantryClient{baseURL: server.URL, http: server.Client()}
	items, err := client.GetPantry(context.Background())

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "p1", items[0].ID)
	assert.Equal(t, "ing1", items[0].IngredientID)
	assert.InDelta(t, 2.5, items[0].Quantity, 0.0001)
	assert.Equal(t, "cup", items[0].Unit)
}

func TestGetPantry_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &PantryClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetPantry(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetPantry_InvalidJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := &PantryClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetPantry(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}
