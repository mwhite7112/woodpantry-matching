package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type PantryItem struct {
	ID           string  `json:"id"`
	IngredientID string  `json:"ingredient_id"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
}

type PantryClient struct {
	baseURL string
	http    *http.Client
}

func NewPantryClient(baseURL string) *PantryClient {
	return &PantryClient{baseURL: baseURL, http: &http.Client{}}
}

func (c *PantryClient) GetPantry(ctx context.Context) ([]PantryItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/pantry", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pantry service returned %d", resp.StatusCode)
	}

	var items []PantryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return items, nil
}
