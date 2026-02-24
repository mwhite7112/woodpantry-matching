package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRecipes_Success(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/recipes", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.Write(
			[]byte(
				`[{"id":"r1","title":"Pasta","tags":["dinner"],"prep_minutes":10,"cook_minutes":20,"ingredients":[{"id":"ri1","ingredient_id":"ing1","quantity":1,"unit":"cup","is_optional":false}]}]`,
			),
		)
	}))
	defer server.Close()

	client := &RecipeClient{baseURL: server.URL, http: server.Client()}
	recipes, err := client.GetRecipes(context.Background())

	require.NoError(t, err)
	require.Len(t, recipes, 1)
	assert.Equal(t, "r1", recipes[0].ID)
	assert.Equal(t, "Pasta", recipes[0].Title)
	require.Len(t, recipes[0].Ingredients, 1)
	assert.Equal(t, "ing1", recipes[0].Ingredients[0].IngredientID)
}

func TestGetRecipes_ServerError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &RecipeClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetRecipes(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestGetRecipes_InvalidJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{bad`))
	}))
	defer server.Close()

	client := &RecipeClient{baseURL: server.URL, http: server.Client()}
	_, err := client.GetRecipes(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}
