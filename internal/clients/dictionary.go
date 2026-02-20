package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// IngredientDetail mirrors the fields returned by GET /ingredients/:id on the
// Ingredient Dictionary service. Field names are capitalized because the
// dictionary serialises sqlc-generated structs that have no json tags.
type IngredientDetail struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

// IngredientSubstitute mirrors the response from GET /ingredients/:id/substitutes.
type IngredientSubstitute struct {
	IngredientID string  `json:"ingredient_id"`
	SubstituteID string  `json:"substitute_id"`
	Ratio        float64 `json:"ratio"`
	Notes        string  `json:"notes"`
}

type DictionaryClient struct {
	baseURL string
	http    *http.Client
}

func NewDictionaryClient(baseURL string) *DictionaryClient {
	return &DictionaryClient{baseURL: baseURL, http: &http.Client{}}
}

// GetIngredient fetches a single ingredient by ID. Returns nil if not found.
func (c *DictionaryClient) GetIngredient(ctx context.Context, id string) (*IngredientDetail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ingredients/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dictionary service returned %d", resp.StatusCode)
	}

	var ing IngredientDetail
	if err := json.NewDecoder(resp.Body).Decode(&ing); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &ing, nil
}

// GetSubstitutes fetches substitute ingredients for the given ingredient ID.
// Returns nil without error if the endpoint is not yet available (404/405),
// making this safe to call before the dictionary service exposes the endpoint.
func (c *DictionaryClient) GetSubstitutes(ctx context.Context, ingredientID string) ([]IngredientSubstitute, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ingredients/"+ingredientID+"/substitutes", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dictionary service returned %d", resp.StatusCode)
	}

	var subs []IngredientSubstitute
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return subs, nil
}
