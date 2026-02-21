package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/mwhite7112/woodpantry-matching/internal/clients"
	"github.com/mwhite7112/woodpantry-matching/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestScoreRecipe_AllInPantry(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Pasta",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", Quantity: 1, Unit: "cup", IsOptional: false},
			{ID: "ri2", IngredientID: "ing2", Quantity: 2, Unit: "tbsp", IsOptional: false},
		},
	}
	pantrySet := map[string]bool{"ing1": true, "ing2": true}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 100.0, result.CoveragePct)
	assert.True(t, result.CanMake)
	assert.Empty(t, result.MissingIngredients)
}

func TestScoreRecipe_PartialCoverage(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Pasta",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", Quantity: 1, Unit: "cup", IsOptional: false},
			{ID: "ri2", IngredientID: "ing2", Quantity: 2, Unit: "tbsp", IsOptional: false},
		},
	}
	pantrySet := map[string]bool{"ing1": true}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 50.0, result.CoveragePct)
	assert.False(t, result.CanMake)
	assert.Len(t, result.MissingIngredients, 1)
	assert.Equal(t, "ing2", result.MissingIngredients[0].IngredientID)
}

func TestScoreRecipe_OptionalOnlyMissing(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Salad",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			{ID: "ri2", IngredientID: "ing2", IsOptional: true},
		},
	}
	pantrySet := map[string]bool{"ing1": true}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 100.0, result.CoveragePct)
	assert.True(t, result.CanMake)
	assert.Empty(t, result.MissingIngredients)
}

func TestScoreRecipe_MaxMissingFiltering(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Stew",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			{ID: "ri2", IngredientID: "ing2", IsOptional: false},
			{ID: "ri3", IngredientID: "ing3", IsOptional: false},
		},
	}
	pantrySet := map[string]bool{"ing1": true}

	// Missing 2 ingredients, maxMissing=2 → can make
	result := scoreRecipe(recipe, pantrySet, nil, 2)
	assert.True(t, result.CanMake)

	// Missing 2 ingredients, maxMissing=1 → cannot make
	result = scoreRecipe(recipe, pantrySet, nil, 1)
	assert.False(t, result.CanMake)
}

func TestScoreRecipe_SubstituteMatching(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Cake",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			{ID: "ri2", IngredientID: "ing2", IsOptional: false},
		},
	}
	pantrySet := map[string]bool{"ing1": true, "sub_ing2": true}
	subsMap := map[string][]clients.IngredientSubstitute{
		"ing2": {{IngredientID: "ing2", SubstituteID: "sub_ing2", Ratio: 1.0}},
	}

	result := scoreRecipe(recipe, pantrySet, subsMap, 0)

	assert.Equal(t, 100.0, result.CoveragePct)
	assert.True(t, result.CanMake)
	assert.Empty(t, result.MissingIngredients)
}

func TestScoreRecipe_EmptyPantry(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Pasta",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", IsOptional: false},
		},
	}
	pantrySet := map[string]bool{}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 0.0, result.CoveragePct)
	assert.False(t, result.CanMake)
}

func TestScoreRecipe_EmptyRecipeIngredients(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:          "r1",
		Title:       "Water",
		Ingredients: []clients.RecipeIngredient{},
	}
	pantrySet := map[string]bool{}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 100.0, result.CoveragePct)
	assert.True(t, result.CanMake)
}

func TestScoreRecipe_AllOptional(t *testing.T) {
	t.Parallel()
	recipe := clients.Recipe{
		ID:    "r1",
		Title: "Garnished Water",
		Ingredients: []clients.RecipeIngredient{
			{ID: "ri1", IngredientID: "ing1", IsOptional: true},
			{ID: "ri2", IngredientID: "ing2", IsOptional: true},
		},
	}
	pantrySet := map[string]bool{}

	result := scoreRecipe(recipe, pantrySet, nil, 0)

	assert.Equal(t, 100.0, result.CoveragePct)
	assert.True(t, result.CanMake)
}

func TestScore_RankingOrder(t *testing.T) {
	t.Parallel()

	pantryMock := mocks.NewMockPantryFetcher(t)
	recipeMock := mocks.NewMockRecipeFetcher(t)
	dictMock := mocks.NewMockDictionaryFetcher(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{
		{ID: "p1", IngredientID: "ing1", Quantity: 1, Unit: "cup"},
	}, nil)

	recipeMock.EXPECT().GetRecipes(mock.Anything).Return([]clients.Recipe{
		{
			ID:    "r1",
			Title: "Full match",
			Ingredients: []clients.RecipeIngredient{
				{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			},
		},
		{
			ID:    "r2",
			Title: "No match",
			Ingredients: []clients.RecipeIngredient{
				{ID: "ri2", IngredientID: "ing2", IsOptional: false},
			},
		},
	}, nil)

	// resolveNames will try to fetch ingredient names for missing ingredients
	dictMock.EXPECT().GetIngredient(mock.Anything, "ing2").Return(&clients.IngredientDetail{ID: "ing2", Name: "butter"}, nil)

	svc := New(pantryMock, recipeMock, dictMock)
	results, err := svc.Score(context.Background(), false, 1)
	require.NoError(t, err)

	require.Len(t, results, 2)
	assert.Equal(t, "Full match", results[0].Recipe.Title)
	assert.Equal(t, 100.0, results[0].CoveragePct)
	assert.Equal(t, "No match", results[1].Recipe.Title)
	assert.Equal(t, 0.0, results[1].CoveragePct)
}

func TestScore_FiltersByCanMake(t *testing.T) {
	t.Parallel()

	pantryMock := mocks.NewMockPantryFetcher(t)
	recipeMock := mocks.NewMockRecipeFetcher(t)
	dictMock := mocks.NewMockDictionaryFetcher(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{
		{ID: "p1", IngredientID: "ing1"},
	}, nil)

	recipeMock.EXPECT().GetRecipes(mock.Anything).Return([]clients.Recipe{
		{
			ID:    "r1",
			Title: "Full match",
			Ingredients: []clients.RecipeIngredient{
				{ID: "ri1", IngredientID: "ing1", IsOptional: false},
			},
		},
		{
			ID:    "r2",
			Title: "Missing two",
			Ingredients: []clients.RecipeIngredient{
				{ID: "ri2", IngredientID: "ing2", IsOptional: false},
				{ID: "ri3", IngredientID: "ing3", IsOptional: false},
			},
		},
	}, nil)

	svc := New(pantryMock, recipeMock, dictMock)

	// maxMissing=0 → only fully matched recipes
	results, err := svc.Score(context.Background(), false, 0)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Full match", results[0].Recipe.Title)
}

func TestScore_PantryFetchError(t *testing.T) {
	t.Parallel()

	pantryMock := mocks.NewMockPantryFetcher(t)
	recipeMock := mocks.NewMockRecipeFetcher(t)
	dictMock := mocks.NewMockDictionaryFetcher(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return(nil, fmt.Errorf("pantry down"))

	svc := New(pantryMock, recipeMock, dictMock)
	_, err := svc.Score(context.Background(), false, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pantry")
}

func TestScore_RecipeFetchError(t *testing.T) {
	t.Parallel()

	pantryMock := mocks.NewMockPantryFetcher(t)
	recipeMock := mocks.NewMockRecipeFetcher(t)
	dictMock := mocks.NewMockDictionaryFetcher(t)

	pantryMock.EXPECT().GetPantry(mock.Anything).Return([]clients.PantryItem{}, nil)
	recipeMock.EXPECT().GetRecipes(mock.Anything).Return(nil, fmt.Errorf("recipes down"))

	svc := New(pantryMock, recipeMock, dictMock)
	_, err := svc.Score(context.Background(), false, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "recipes")
}
