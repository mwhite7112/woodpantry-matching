package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIngredient_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ingredients/abc-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ID":"abc-123","Name":"garlic"}`))
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	detail, err := client.GetIngredient(context.Background(), "abc-123")

	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "abc-123", detail.ID)
	assert.Equal(t, "garlic", detail.Name)
}

func TestGetIngredient_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	detail, err := client.GetIngredient(context.Background(), "missing")

	require.ErrorIs(t, err, ErrIngredientNotFound)
	assert.Nil(t, detail)
}

func TestGetIngredient_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetIngredient(context.Background(), "abc")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetSubstitutes_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ingredients/abc-123/substitutes", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"ingredient_id":"abc-123","substitute_id":"def-456","ratio":1.0,"notes":""}]`))
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	subs, err := client.GetSubstitutes(context.Background(), "abc-123")

	require.NoError(t, err)
	require.Len(t, subs, 1)
	assert.Equal(t, "abc-123", subs[0].IngredientID)
	assert.Equal(t, "def-456", subs[0].SubstituteID)
}

func TestGetSubstitutes_NotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	subs, err := client.GetSubstitutes(context.Background(), "missing")

	require.NoError(t, err)
	assert.Empty(t, subs)
}

func TestGetSubstitutes_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	subs, err := client.GetSubstitutes(context.Background(), "abc")

	require.NoError(t, err)
	assert.Empty(t, subs)
}

func TestGetSubstitutes_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &DictionaryClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetSubstitutes(context.Background(), "abc")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
