package main

import (
	"fmt"
	"os"

	"link-inspector/crawler"
)

func main() {
	testURL := "https://go.dev"

	// Fase 1: Obtener el HTML
	fmt.Printf("🔍 Obteniendo HTML de: %s\n", testURL)
	body, err := crawler.FetchURL(testURL)
	if err != nil {
		fmt.Printf("❌ Error obteniendo HTML: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Recibidos %d bytes de HTML\n\n", len(body))

	// Fase 2: Extraer los enlaces
	fmt.Println("🔗 Extrayendo enlaces...")
	links, err := crawler.ExtractLinks(body, testURL)
	if err != nil {
		fmt.Printf("❌ Error extrayendo enlaces: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Encontrados %d enlaces\n\n", len(links))

	// Mostramos los primeros 15 enlaces para no llenar la pantalla.
	// math.Min no existe para ints en Go (solo float64), así que
	// usamos un if simple. Esto es una peculiaridad de Go que
	// cambió en versiones recientes con generics, pero el if es claro.
	limit := 15
	if len(links) < limit {
		limit = len(links)
	}

	fmt.Println("--- Primeros enlaces encontrados ---")
	for i, link := range links[:limit] {
		// %2d formatea el número con 2 dígitos de ancho (alineación)
		fmt.Printf("  %2d. %s\n", i+1, link)
	}

	if len(links) > limit {
		fmt.Printf("  ... y %d más\n", len(links)-limit)
	}
}
