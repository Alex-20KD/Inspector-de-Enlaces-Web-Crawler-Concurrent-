package main

import (
	"fmt"
	"os"

	"link-inspector/crawler"
)

func main() {
	testURL := "https://go.dev"
	numWorkers := 5

	// Fase 1: Obtener el HTML
	fmt.Printf("🔍 Obteniendo HTML de: %s\n", testURL)
	body, err := crawler.FetchURL(testURL)
	if err != nil {
		fmt.Printf("❌ Error obteniendo HTML: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Recibidos %d bytes de HTML\n\n", len(body))

	// Fase 2: Extraer enlaces
	fmt.Println("🔗 Extrayendo enlaces...")
	links, err := crawler.ExtractLinks(body, testURL)
	if err != nil {
		fmt.Printf("❌ Error extrayendo enlaces: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Encontrados %d enlaces\n\n", len(links))

	// Fase 3+4: Verificar enlaces concurrentemente
	// Limitamos a los primeros 20 enlaces para la prueba (no queremos
	// esperar 148 peticiones HTTP en un test rápido).
	limit := 20
	if len(links) < limit {
		limit = len(links)
	}
	testLinks := links[:limit]

	fmt.Printf("⚡ Verificando %d enlaces con %d workers...\n\n", len(testLinks), numWorkers)
	results := crawler.RunWorkers(testLinks, numWorkers)

	// Mostrar resultados
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  ❌ ERROR  %s\n           → %v\n", r.URL, r.Err)
		} else if r.StatusCode >= 400 {
			fmt.Printf("  ⚠️  %d   %s\n", r.StatusCode, r.URL)
		} else {
			fmt.Printf("  ✅ %d   %s\n", r.StatusCode, r.URL)
		}
	}
}
