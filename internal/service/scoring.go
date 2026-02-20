package service

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/mwhite7112/woodpantry-matching/internal/clients"
)

type MissingIngredient struct {
	IngredientID string  `json:"ingredient_id"`
	Name         string  `json:"name,omitempty"`
	Quantity     float64 `json:"quantity"`
	Unit         string  `json:"unit"`
}

type MatchResult struct {
	Recipe             clients.Recipe      `json:"recipe"`
	CoveragePct        float64             `json:"coverage_pct"`
	MissingIngredients []MissingIngredient `json:"missing_ingredients"`
	CanMake            bool                `json:"can_make"`
}

type Service struct {
	pantry     *clients.PantryClient
	recipes    *clients.RecipeClient
	dictionary *clients.DictionaryClient
}

func New(pantry *clients.PantryClient, recipes *clients.RecipeClient, dictionary *clients.DictionaryClient) *Service {
	return &Service{pantry: pantry, recipes: recipes, dictionary: dictionary}
}

// Score fetches live pantry and recipe data, scores each recipe by ingredient
// coverage, and returns results ranked by coverage descending.
// Only recipes with missing_count <= maxMissing are included in the result.
func (s *Service) Score(ctx context.Context, allowSubs bool, maxMissing int) ([]MatchResult, error) {
	pantryItems, err := s.pantry.GetPantry(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch pantry: %w", err)
	}

	recipes, err := s.recipes.GetRecipes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch recipes: %w", err)
	}

	// Build ingredient_id presence set from pantry.
	pantrySet := make(map[string]bool, len(pantryItems))
	for _, item := range pantryItems {
		pantrySet[item.IngredientID] = true
	}

	// Pre-fetch substitute data in parallel for all required-but-missing ingredient IDs.
	subsMap := make(map[string][]clients.IngredientSubstitute)
	if allowSubs {
		missingIDs := make(map[string]bool)
		for _, recipe := range recipes {
			for _, ing := range recipe.Ingredients {
				if !ing.IsOptional && !pantrySet[ing.IngredientID] {
					missingIDs[ing.IngredientID] = true
				}
			}
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		for id := range missingIDs {
			wg.Add(1)
			go func(ingredientID string) {
				defer wg.Done()
				subs, err := s.dictionary.GetSubstitutes(ctx, ingredientID)
				if err != nil || subs == nil {
					return
				}
				mu.Lock()
				subsMap[ingredientID] = subs
				mu.Unlock()
			}(id)
		}
		wg.Wait()
	}

	results := make([]MatchResult, 0, len(recipes))
	for _, recipe := range recipes {
		results = append(results, scoreRecipe(recipe, pantrySet, subsMap, maxMissing))
	}

	// Sort by coverage descending, then fewest missing as tiebreaker.
	sort.Slice(results, func(i, j int) bool {
		if results[i].CoveragePct != results[j].CoveragePct {
			return results[i].CoveragePct > results[j].CoveragePct
		}
		return len(results[i].MissingIngredients) < len(results[j].MissingIngredients)
	})

	// Filter to only includable recipes (can_make == true).
	filtered := make([]MatchResult, 0, len(results))
	for _, r := range results {
		if r.CanMake {
			filtered = append(filtered, r)
		}
	}

	// Best-effort: resolve ingredient names from dictionary for missing ingredients.
	// Errors are silently ignored — the caller still receives results without names.
	s.resolveNames(ctx, filtered)

	return filtered, nil
}

// scoreRecipe computes a single recipe's coverage score against the pantry set.
// subsMap provides pre-fetched substitute data for allow_subs scoring.
func scoreRecipe(
	recipe clients.Recipe,
	pantrySet map[string]bool,
	subsMap map[string][]clients.IngredientSubstitute,
	maxMissing int,
) MatchResult {
	required := make([]clients.RecipeIngredient, 0, len(recipe.Ingredients))
	for _, ing := range recipe.Ingredients {
		if !ing.IsOptional {
			required = append(required, ing)
		}
	}

	if len(required) == 0 {
		return MatchResult{
			Recipe:             recipe,
			CoveragePct:        100.0,
			MissingIngredients: []MissingIngredient{},
			CanMake:            true,
		}
	}

	missing := make([]MissingIngredient, 0)
	matched := 0

	for _, ing := range required {
		if pantrySet[ing.IngredientID] {
			matched++
			continue
		}

		// Check if any substitute for this ingredient is in the pantry.
		foundSub := false
		for _, sub := range subsMap[ing.IngredientID] {
			if pantrySet[sub.SubstituteID] {
				matched++
				foundSub = true
				break
			}
		}

		if !foundSub {
			missing = append(missing, MissingIngredient{
				IngredientID: ing.IngredientID,
				Quantity:     ing.Quantity,
				Unit:         ing.Unit,
			})
		}
	}

	coveragePct := float64(matched) / float64(len(required)) * 100.0
	return MatchResult{
		Recipe:             recipe,
		CoveragePct:        coveragePct,
		MissingIngredients: missing,
		CanMake:            len(missing) <= maxMissing,
	}
}

// resolveNames fetches ingredient names from the dictionary for all unique
// missing ingredient IDs across results, populating the Name field in-place.
func (s *Service) resolveNames(ctx context.Context, results []MatchResult) {
	seen := make(map[string]bool)
	for _, r := range results {
		for _, m := range r.MissingIngredients {
			seen[m.IngredientID] = true
		}
	}
	if len(seen) == 0 {
		return
	}

	nameMap := make(map[string]string, len(seen))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for id := range seen {
		wg.Add(1)
		go func(ingredientID string) {
			defer wg.Done()
			detail, err := s.dictionary.GetIngredient(ctx, ingredientID)
			if err != nil || detail == nil {
				return
			}
			mu.Lock()
			nameMap[ingredientID] = detail.Name
			mu.Unlock()
		}(id)
	}
	wg.Wait()

	for i := range results {
		for j := range results[i].MissingIngredients {
			id := results[i].MissingIngredients[j].IngredientID
			if name, ok := nameMap[id]; ok {
				results[i].MissingIngredients[j].Name = name
			}
		}
	}
}
