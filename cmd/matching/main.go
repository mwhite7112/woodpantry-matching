package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mwhite7112/woodpantry-matching/internal/api"
	"github.com/mwhite7112/woodpantry-matching/internal/clients"
	"github.com/mwhite7112/woodpantry-matching/internal/logging"
	"github.com/mwhite7112/woodpantry-matching/internal/service"
)

func main() {
	logging.Setup()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pantryURL := os.Getenv("PANTRY_URL")
	if pantryURL == "" {
		slog.Error("PANTRY_URL is required")
		os.Exit(1)
	}

	recipeURL := os.Getenv("RECIPE_URL")
	if recipeURL == "" {
		slog.Error("RECIPE_URL is required")
		os.Exit(1)
	}

	dictionaryURL := os.Getenv("DICTIONARY_URL")
	if dictionaryURL == "" {
		slog.Error("DICTIONARY_URL is required")
		os.Exit(1)
	}

	svc := service.New(
		clients.NewPantryClient(pantryURL),
		clients.NewRecipeClient(recipeURL),
		clients.NewDictionaryClient(dictionaryURL),
	)

	handler := api.NewRouter(svc)

	addr := fmt.Sprintf(":%s", port)
	slog.Info("matching service listening", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
