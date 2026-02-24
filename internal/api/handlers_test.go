package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/mwhite7112/woodpantry-matching/internal/clients"
	"github.com/mwhite7112/woodpantry-matching/internal/mocks"
	"github.com/mwhite7112/woodpantry-matching/internal/service"
)

func setupRouter(
	t *testing.T,
) (http.Handler, *mocks.MockPantryFetcher, *mocks.MockRecipeFetcher) {
	pantryMock := mocks.NewMockPantryFetcher(t)
	recipeMock := mocks.NewMockRecipeFetcher(t)
	dictMock := mocks.NewMockDictionaryFetcher(t)

	svc := service.New(pantryMock, recipeMock, dictMock)
	router := NewRouter(svc)
	return router, pantryMock, recipeMock
}

func TestHealthz(t *testing.T) {
	router, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestGetMatches_Success(t *testing.T) {
	router, pantryMock, recipeMock := setupRouter(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{
		{ID: "p1", IngredientID: "ing1"},
	}, nil)
	recipeMock.EXPECT().GetRecipes(mock.Anything).Return([]clients.Recipe{
		{
			ID:    "r1",
			Title: "Simple",
			Ingredients: []clients.RecipeIngredient{
				{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/matches?max_missing=0", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var results []service.MatchResult
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&results))
	require.Len(t, results, 1)
	assert.InDelta(t, 100.0, results[0].CoveragePct, 0.0001)
}

func TestGetMatches_InvalidMaxMissing(t *testing.T) {
	router, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/matches?max_missing=abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMatches_NegativeMaxMissing(t *testing.T) {
	router, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/matches?max_missing=-1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMatches_BackendError(t *testing.T) {
	router, pantryMock, _ := setupRouter(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return(nil, errors.New("down"))

	req := httptest.NewRequest(http.MethodGet, "/matches", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestPostMatchQuery_Success(t *testing.T) {
	router, pantryMock, recipeMock := setupRouter(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{}, nil)
	recipeMock.EXPECT().GetRecipes(mock.Anything).Return([]clients.Recipe{}, nil)

	body := `{"prompt":"something quick","max_missing":0}`
	req := httptest.NewRequest(http.MethodPost, "/matches/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPostMatchQuery_InvalidBody(t *testing.T) {
	router, _, _ := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/matches/query", strings.NewReader(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPostMatchQuery_NegativeMaxMissing(t *testing.T) {
	router, pantryMock, recipeMock := setupRouter(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{}, nil)
	recipeMock.EXPECT().GetRecipes(mock.Anything).Return([]clients.Recipe{}, nil)

	body := `{"max_missing":-5}`
	req := httptest.NewRequest(http.MethodPost, "/matches/query", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Negative max_missing is clamped to 0, not an error
	assert.Equal(t, http.StatusOK, rec.Code)
}
