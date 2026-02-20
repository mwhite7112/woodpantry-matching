package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mwhite7112/woodpantry-matching/internal/api"
	"github.com/mwhite7112/woodpantry-matching/internal/clients"
	"github.com/mwhite7112/woodpantry-matching/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	pantryURL := os.Getenv("PANTRY_URL")
	if pantryURL == "" {
		log.Fatal("PANTRY_URL is required")
	}

	recipeURL := os.Getenv("RECIPE_URL")
	if recipeURL == "" {
		log.Fatal("RECIPE_URL is required")
	}

	dictionaryURL := os.Getenv("DICTIONARY_URL")
	if dictionaryURL == "" {
		log.Fatal("DICTIONARY_URL is required")
	}

	svc := service.New(
		clients.NewPantryClient(pantryURL),
		clients.NewRecipeClient(recipeURL),
		clients.NewDictionaryClient(dictionaryURL),
	)

	handler := api.NewRouter(svc)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("matching service listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
