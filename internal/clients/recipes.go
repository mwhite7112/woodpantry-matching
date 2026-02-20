package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type RecipeIngredient struct {
	ID           string  `json:"id"`
	IngredientID string  `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
	IsOptional   bool    `json:"is_optional"`
}

type Recipe struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Tags        []string           `json:"tags"`
	PrepMinutes int                `json:"prep_minutes"`
	CookMinutes int                `json:"cook_minutes"`
	Ingredients []RecipeIngredient `json:"ingredients"`
}

type RecipeClient struct {
	baseURL string
	http    *http.Client
}

func NewRecipeClient(baseURL string) *RecipeClient {
	return &RecipeClient{baseURL: baseURL, http: &http.Client{}}
}

func (c *RecipeClient) GetRecipes(ctx context.Context) ([]Recipe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/recipes", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recipe service returned %d", resp.StatusCode)
	}

	var recipes []Recipe
	if err := json.NewDecoder(resp.Body).Decode(&recipes); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return recipes, nil
}
