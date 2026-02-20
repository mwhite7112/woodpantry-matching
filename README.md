# woodpantry-matching

Matching Service for WoodPantry. Stateless query layer that scores recipes by pantry coverage — answering "what can I make right now?" No database. Reads live from Pantry Service and Recipe Service.

Phase 3 adds semantic re-ranking: a natural language prompt is embedded and used to re-rank the deterministic candidate set.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/matches` | Recipes scored by pantry coverage |
| POST | `/matches/query` | Deterministic + semantic combined query |

### GET /matches

```
GET /matches?allow_subs=true&max_missing=2
```

Returns all recipes ranked by pantry coverage percentage. Optional params:
- `allow_subs` — count substitute ingredients as available
- `max_missing` — only return recipes missing at most N required ingredients

```json
{
  "results": [
    {
      "recipe": { "id": "uuid", "title": "Garlic Pasta", "cook_minutes": 20, "tags": ["italian"] },
      "coverage_pct": 100,
      "can_make": true,
      "missing_ingredients": []
    },
    {
      "recipe": { "id": "uuid", "title": "Chicken Stir Fry", "cook_minutes": 25, "tags": ["asian"] },
      "coverage_pct": 80,
      "can_make": false,
      "missing_ingredients": [{ "name": "soy sauce", "quantity": 2, "unit": "tbsp" }]
    }
  ]
}
```

### POST /matches/query

The primary Cook View interface. Phase 1: ignores `prompt`, runs deterministic scoring. Phase 3: uses `prompt` for semantic re-ranking.

```json
// Request
{
  "prompt": "something spicy and quick, maybe Asian",
  "pantry_constrained": true,
  "max_missing": 2
}

// Response — same shape as GET /matches
```

## Scoring Logic

**Phase 1 — Deterministic:**
```
coverage_score = matched_required_ingredients / total_required_ingredients
```

**Phase 3 — Semantic re-ranking:**
```
final_score = (1 - SEMANTIC_WEIGHT) * coverage_score + SEMANTIC_WEIGHT * cosine_similarity(prompt_embedding, recipe_embedding)
```
The LLM re-ranks from the deterministic candidate set — it never surfaces recipes you can't make.

## Events (Phase 2+)

| Event | Direction | Description |
|-------|-----------|-------------|
| `pantry.updated` | Subscribes | Invalidates cached pantry state |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `PANTRY_URL` | required | Pantry Service base URL |
| `RECIPE_URL` | required | Recipe Service base URL |
| `DICTIONARY_URL` | required | Ingredient Dictionary base URL |
| `OPENAI_API_KEY` | optional (Phase 3) | Required for Phase 3 semantic re-ranking embeddings |
| `EMBED_MODEL` | `text-embedding-3-small` | OpenAI embedding model for query vectors (Phase 3) |
| `SEMANTIC_WEIGHT` | `0.4` | Semantic vs coverage score weight (Phase 3) |
| `RABBITMQ_URL` | optional | Enables pantry.updated cache invalidation (Phase 2+) |
| `LOG_LEVEL` | `info` | Log level |

## Development

```bash
go run ./cmd/matching/main.go
```
