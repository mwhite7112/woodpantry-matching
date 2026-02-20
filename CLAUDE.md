# woodpantry-matching — Matching Service

## Role in Architecture

Stateless query layer that answers "what can I make right now?" by scoring recipes against current pantry state. This service owns no data and has no database — it reads live from the Pantry Service and Recipe Service on every request.

In Phase 1, matching is deterministic: ingredient coverage scoring (what % of a recipe's required ingredients are in the pantry). In Phase 3, `POST /matches/query` adds a semantic re-ranking layer using pgvector embeddings and a natural language prompt.

## Technology

- Language: Go
- HTTP: chi
- No database (stateless)
- RabbitMQ (Phase 2+): subscribes to `pantry.updated` for cache invalidation
- LLM/embeddings (Phase 3): OpenAI API (`text-embedding-3-small`) for generating query embeddings to re-rank results

## Service Dependencies

- **Calls**: Pantry Service (`GET /pantry`), Recipe Service (`GET /recipes`), Ingredient Dictionary (`GET /ingredients/:id` for substitute data)
- **Called by**: Web frontend, CLI
- **Subscribes to** (Phase 2+): `pantry.updated` (cache invalidation)
- **Publishes**: nothing

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/matches` | Recipes scored by pantry coverage |
| POST | `/matches/query` | Combined deterministic + semantic query |

### GET /matches

Query params:
- `allow_subs=true` — use substitute ingredient data from Dictionary when scoring
- `max_missing=N` — include recipes missing at most N required ingredients

### POST /matches/query

The primary "what do I cook tonight?" interface. Sample request:

```json
{
  "prompt": "something spicy and quick, maybe Asian",
  "pantry_constrained": true,
  "max_missing": 2
}
```

**Phase 1 behaviour**: `prompt` is ignored. Runs deterministic coverage scoring only.
**Phase 3 behaviour**: Deterministic scoring produces a candidate set, then semantic similarity against the prompt re-ranks results. This prevents the LLM from hallucinating recipes you cannot make.

## Key Patterns

### Deterministic Coverage Scoring

Coverage score per recipe = (matched required ingredients) / (total required ingredients)

"Matched" means the pantry contains that ingredient_id at quantity ≥ 0 (any amount counts as "have it"). When `allow_subs=true`, also check if a substitute for the missing ingredient is in the pantry.

### Semantic Re-ranking (Phase 3)

1. Run deterministic scoring to get candidate set
2. Filter by `pantry_constrained` and `max_missing`
3. Generate embedding for the user's `prompt` via OpenAI API (`text-embedding-3-small`)
4. Score each candidate recipe by cosine similarity between prompt embedding and stored recipe embedding
5. Final rank = weighted combination: `(1 - SEMANTIC_WEIGHT) * coverage_score + SEMANTIC_WEIGHT * cosine_similarity`

### Response Shape

Each recipe in the result includes:
- Recipe card (title, tags, prep_minutes, cook_minutes)
- `coverage_pct` — percentage of required ingredients in pantry
- `missing_ingredients` — list of what's missing (ingredient name + quantity needed)
- `can_make` — boolean (true if coverage_pct == 100% or missing ≤ max_missing)

### Caching (Phase 2+)

On `pantry.updated` event, invalidate any in-memory pantry state cache. The Matching Service may cache the pantry state for a short TTL to avoid hammering the Pantry Service on every request. Cache must be invalidated on any pantry change.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `PANTRY_URL` | required | Pantry Service base URL |
| `RECIPE_URL` | required | Recipe Service base URL |
| `DICTIONARY_URL` | required | Ingredient Dictionary base URL |
| `OPENAI_API_KEY` | optional (Phase 3) | OpenAI API key — required for Phase 3 semantic re-ranking embeddings |
| `EMBED_MODEL` | `text-embedding-3-small` | OpenAI embedding model for query vectors (Phase 3) |
| `SEMANTIC_WEIGHT` | `0.4` | Weight of semantic score vs coverage score (Phase 3) |
| `RABBITMQ_URL` | optional | Enables pantry.updated subscription for cache invalidation (Phase 2+) |
| `LOG_LEVEL` | `info` | Log level |

## Directory Layout

```
woodpantry-matching/
├── cmd/matching/main.go
├── internal/
│   ├── api/
│   │   └── handlers.go
│   ├── service/
│   │   ├── scoring.go         ← deterministic coverage scoring
│   │   ├── semantic.go        ← embedding generation + cosine similarity (Phase 3)
│   │   └── cache.go           ← pantry state cache (Phase 2+)
│   ├── clients/
│   │   ├── pantry.go          ← HTTP client for Pantry Service
│   │   ├── recipes.go         ← HTTP client for Recipe Service
│   │   └── dictionary.go      ← HTTP client for Ingredient Dictionary
│   └── events/
│       └── subscriber.go      ← consume pantry.updated (Phase 2+)
├── kubernetes/
├── Dockerfile
├── go.mod
└── go.sum
```

## What to Avoid

- Do not add a database — this service is intentionally stateless.
- Do not allow the LLM/semantic layer to return recipes that aren't in the candidate set produced by deterministic scoring. The LLM re-ranks; it does not hallucinate new candidates.
- Do not call the Pantry Service or Recipe Service per recipe in a loop — fetch all pantry items and all recipes in bulk, then score in memory.
- Do not add RabbitMQ or caching in Phase 1 — keep it simple until needed.
