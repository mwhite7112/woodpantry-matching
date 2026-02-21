package service

import (
	"context"

	"github.com/mwhite7112/woodpantry-matching/internal/clients"
)

// PantryFetcher abstracts the Pantry Service client for testing.
type PantryFetcher interface {
	GetPantry(ctx context.Context) ([]clients.PantryItem, error)
}

// RecipeFetcher abstracts the Recipe Service client for testing.
type RecipeFetcher interface {
	GetRecipes(ctx context.Context) ([]clients.Recipe, error)
}

// DictionaryFetcher abstracts the Ingredient Dictionary client for testing.
type DictionaryFetcher interface {
	GetIngredient(ctx context.Context, id string) (*clients.IngredientDetail, error)
	GetSubstitutes(ctx context.Context, ingredientID string) ([]clients.IngredientSubstitute, error)
}
