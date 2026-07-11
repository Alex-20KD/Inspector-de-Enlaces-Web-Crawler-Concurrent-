package main

import (
	"flag" // Paquete de la stdlib para parsear argumentos de línea de comandos
	"fmt"
	"os"
	"sort"    // Para ordenar slices
	"strings" // Para manipulación de strings
	"time"    // Para medir duración

	"link-inspector/crawler"
)

// Códigos de color ANSI para la terminal.
// Estos son secuencias de escape que la terminal interpreta como instrucciones
// de formato, no como texto. "\033[" es el prefijo de escape, y el número
// indica el color/estilo. "0m" resetea a normal.
//
// Esto funciona en prácticamente cualquier terminal moderna (Linux, macOS,
// Windows Terminal). Si alguna terminal no los soporta, simplemente se verían
// los códigos como texto — no rompería nada.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func main() {
	// ===== FLAGS DE LÍNEA DE COMANDOS =====
	//
	// El paquete "flag" permite definir argumentos con nombre.
	// flag.String retorna un *string (puntero), no un string.
	// ¿Por qué puntero? Porque flag necesita escribir el valor DESPUÉS
	// de que tú declares la variable — el puntero le da acceso para
	// modificarla cuando parsee los argumentos.
	//
	// Cada flag se define con: (nombre, valor_por_defecto, descripción)
	targetURL := flag.String("url", "", "URL a inspeccionar (requerida)")
	numWorkers := flag.Int("workers", 10, "Número de workers concurrentes")

	// flag.Parse() lee os.Args y rellena las variables de arriba.
	// DEBE llamarse después de definir todos los flags y ANTES de usarlos.
	flag.Parse()

	// Validación: la URL es obligatoria.
	// *targetURL desreferencia el puntero para obtener el string real.
	if *targetURL == "" {
		fmt.Println(colorRed + "❌ Error: debes proporcionar una URL con --url" + colorReset)
		fmt.Println()
		fmt.Println("Uso:")
		// flag.PrintDefaults() imprime automáticamente la ayuda de todos
		// los flags registrados, con sus tipos y valores por defecto.
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Ejemplo:")
		fmt.Println("  go run main.go --url=https://go.dev --workers=10")
		os.Exit(1)
	}

	// ===== BANNER =====
	printBanner()

	// ===== FASE 1: OBTENER HTML =====
	fmt.Printf("%s📡 Descargando HTML de:%s %s\n", colorCyan, colorReset, *targetURL)
	startFetch := time.Now()

	body, err := crawler.FetchURL(*targetURL)
	if err != nil {
		fmt.Printf("%s❌ Error: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	fmt.Printf("%s   ✓ %d bytes en %v%s\n\n", colorDim, len(body), time.Since(startFetch).Round(time.Millisecond), colorReset)

	// ===== FASE 2: EXTRAER ENLACES =====
	fmt.Printf("%s🔗 Extrayendo enlaces...%s\n", colorCyan, colorReset)

	links, err := crawler.ExtractLinks(body, *targetURL)
	if err != nil {
		fmt.Printf("%s❌ Error: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	// Eliminamos duplicados. Una página puede tener el mismo enlace
	// en el header, footer, y contenido — no necesitamos verificarlo 3 veces.
	links = removeDuplicates(links)
	fmt.Printf("%s   ✓ %d enlaces únicos encontrados%s\n\n", colorDim, len(links), colorReset)

	if len(links) == 0 {
		fmt.Println("No se encontraron enlaces. ¿Es la URL correcta?")
		return
	}

	// ===== FASE 3-5: VERIFICAR CONCURRENTEMENTE =====
	fmt.Printf("%s⚡ Verificando con %d workers...%s\n\n", colorCyan, *numWorkers, colorReset)
	startCheck := time.Now()

	results := crawler.RunWorkers(links, *numWorkers)

	elapsed := time.Since(startCheck).Round(time.Millisecond)

	// ===== ORDENAR Y MOSTRAR RESULTADOS =====
	// Ordenamos por status code para que los errores se vean agrupados.
	// sort.Slice recibe un slice y una función "less" que define el orden.
	// Es un ejemplo de pasar funciones como argumentos (first-class functions).
	sort.Slice(results, func(i, j int) bool {
		// Los errores (StatusCode == 0) van al final
		if results[i].StatusCode == 0 && results[j].StatusCode != 0 {
			return false
		}
		if results[i].StatusCode != 0 && results[j].StatusCode == 0 {
			return true
		}
		return results[i].StatusCode < results[j].StatusCode
	})

	printResultsTable(results)
	printSummary(results, elapsed)
}

// printBanner imprime un header decorativo al iniciar el programa.
func printBanner() {
	fmt.Println()
	fmt.Printf("%s%s", colorBold, colorCyan)
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║     🔍 LINK INSPECTOR v1.0          ║")
	fmt.Println("  ║     Concurrent Link Checker in Go    ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Printf("%s\n", colorReset)
}

// printResultsTable imprime los resultados en formato de tabla alineada.
func printResultsTable(results []crawler.Result) {
	// Encontrar la URL más larga para alinear la tabla.
	// Limitamos a 60 caracteres para que no se rompa en terminales angostas.
	maxURLLen := 0
	for _, r := range results {
		if len(r.URL) > maxURLLen {
			maxURLLen = len(r.URL)
		}
	}
	if maxURLLen > 60 {
		maxURLLen = 60
	}

	// Header de la tabla
	fmt.Printf("  %s%-8s %-*s %s%s\n",
		colorBold, "STATUS", maxURLLen, "URL", "DETALLE", colorReset)
	fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat("─", 8+maxURLLen+20), colorReset)

	// Filas
	for _, r := range results {
		// Truncar URLs largas para que la tabla no se rompa.
		displayURL := r.URL
		if len(displayURL) > maxURLLen {
			displayURL = displayURL[:maxURLLen-3] + "..."
		}

		if r.Err != nil {
			// Error de conexión (timeout, DNS, etc.)
			fmt.Printf("  %s  ERR   %-*s %s%s\n",
				colorRed, maxURLLen, displayURL, truncateError(r.Err), colorReset)
		} else if r.StatusCode >= 400 {
			// Error HTTP (404, 500, etc.)
			fmt.Printf("  %s  %d   %-*s %s%s\n",
				colorYellow, r.StatusCode, maxURLLen, displayURL, httpStatusText(r.StatusCode), colorReset)
		} else if r.StatusCode >= 300 {
			// Redirección (301, 302)
			fmt.Printf("  %s  %d   %-*s %s%s\n",
				colorCyan, r.StatusCode, maxURLLen, displayURL, httpStatusText(r.StatusCode), colorReset)
		} else {
			// Éxito (200, 201, etc.)
			fmt.Printf("  %s  %d   %s%-*s%s\n",
				colorGreen, r.StatusCode, colorReset, maxURLLen, displayURL, "")
		}
	}
	fmt.Println()
}

// printSummary imprime estadísticas finales.
func printSummary(results []crawler.Result, elapsed time.Duration) {
	var ok, clientErr, serverErr, connErr int
	for _, r := range results {
		switch {
		// switch sin expresión actúa como if-else if-else.
		// Cada case es una condición booleana. Go evalúa de arriba a abajo
		// y ejecuta el primer case que sea true.
		case r.Err != nil:
			connErr++
		case r.StatusCode >= 500:
			serverErr++
		case r.StatusCode >= 400:
			clientErr++
		default:
			ok++
		}
	}

	fmt.Printf("  %s%s📊 Resumen%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat("─", 40), colorReset)
	fmt.Printf("  %s✅ OK (2xx/3xx):    %d%s\n", colorGreen, ok, colorReset)
	if clientErr > 0 {
		fmt.Printf("  %s⚠️  Client (4xx):    %d%s\n", colorYellow, clientErr, colorReset)
	}
	if serverErr > 0 {
		fmt.Printf("  %s🔥 Server (5xx):    %d%s\n", colorRed, serverErr, colorReset)
	}
	if connErr > 0 {
		fmt.Printf("  %s❌ Conn Error:      %d%s\n", colorRed, connErr, colorReset)
	}
	fmt.Printf("  %s   Total:           %d enlaces en %v%s\n", colorDim, len(results), elapsed, colorReset)
	fmt.Println()
}

// removeDuplicates elimina URLs duplicadas manteniendo el orden.
// Usa un map como "set" — en Go no hay un tipo Set nativo, así que
// se usa map[string]bool donde la key es el elemento y el value
// indica "ya lo vi".
func removeDuplicates(urls []string) []string {
	seen := make(map[string]bool)
	var unique []string
	for _, url := range urls {
		// El "comma ok" pattern: seen[url] retorna (valor, existe).
		// Si la key no existe, retorna el valor cero (false) y ok=false.
		// Aquí simplificamos: si seen[url] es false (no visto), lo agregamos.
		if !seen[url] {
			seen[url] = true
			unique = append(unique, url)
		}
	}
	return unique
}

// truncateError acorta mensajes de error largos para la tabla.
func truncateError(err error) string {
	msg := err.Error()
	if len(msg) > 50 {
		return msg[:47] + "..."
	}
	return msg
}

// httpStatusText devuelve una descripción corta del código HTTP.
func httpStatusText(code int) string {
	// Un switch clásico sobre un valor. Go no necesita "break" al final
	// de cada case — a diferencia de C/Java, Go NO cae al siguiente case
	// automáticamente (no hay "fallthrough" implícito).
	switch code {
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found (redirect)"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	default:
		return ""
	}
}
