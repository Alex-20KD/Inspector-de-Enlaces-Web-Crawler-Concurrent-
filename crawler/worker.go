package crawler

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CheckURL hace una petición HTTP HEAD a la URL dada y retorna
// el código de estado HTTP o un error.
//
// Parámetros:
//   - rawURL: la URL a verificar (ej: "https://go.dev/doc")
//
// Retorna:
//   - int:   el código HTTP (200, 301, 404, 500, etc.)
//   - error: si hubo un problema de conexión (timeout, DNS, etc.)
//     Si la petición fue exitosa (aunque sea 404), error es nil.
//     Un 404 NO es un error de conexión — es una respuesta válida.
//
// Comportamiento esperado:
//
//  1. Crear un http.Client con un Timeout (ej: 10 segundos).
//     ¿Por qué? Porque sin timeout, si un servidor no responde,
//     tu goroutine se queda bloqueada PARA SIEMPRE.
//
//  2. Hacer una petición HEAD con client.Head(rawURL).
//     HEAD es como GET pero solo devuelve headers, sin body.
//     Más rápido para solo verificar si un enlace vive.
//
//  3. Si la petición retorna error, devolver (0, error).
//     Usa fmt.Errorf con %w para envolver el error, igual que en FetchURL.
//
//  4. Si la petición fue exitosa, CERRAR el Body con resp.Body.Close().
//     Sí, incluso en HEAD — el Body existe, solo que está vacío.
//     Si no lo cierras, Go mantiene la conexión TCP abierta →
//     con 148 URLs, se te acaban las conexiones del sistema.
//     PISTA: ¿recuerdas defer?
//
//  5. Retornar (resp.StatusCode, nil).
//
// Pistas:
//   - &http.Client{Timeout: 10 * time.Second}
//   - client.Head(url) retorna (*http.Response, error)
//   - resp.StatusCode es un int con el código HTTP
//   - defer resp.Body.Close() para liberar la conexión
//   - fmt.Errorf("mensaje: %w", err) para envolver errores
//
// NUEVA FIRMA: ctx es el primer parámetro (convención de Go).
// En Go, context.Context SIEMPRE va como primer parámetro, SIEMPRE
// se llama "ctx", y NUNCA se guarda en un struct. Es una convención
// tan fuerte que los linters la verifican.
//
// Cambios que necesitas hacer:
//
//  1. En vez de client.Head(rawURL), crea un request con contexto:
//     req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
//     ("HEAD" es el método HTTP, nil es el body — HEAD no tiene body)
//
//  2. Ejecuta el request con:
//     resp, err := client.Do(req)
//     client.Do() acepta cualquier *http.Request, incluyendo los
//     que tienen contexto. Si el contexto se cancela, Do() retorna
//     error inmediatamente.
//
//  3. Maneja los errores de crear el request Y de ejecutarlo.
//
// El Timeout del http.Client (10s) sigue siendo útil como timeout
// POR REQUEST. El context es un timeout GLOBAL para todo el programa.
// El que expire primero "gana" y cancela la petición.
func CheckURL(ctx context.Context, rawURL string) (int, error) {
	// TODO: modifica tu código aquí para usar ctx
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return 0, fmt.Errorf("error al crear la petición: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("error al hacer petición: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// Result almacena el resultado de verificar un enlace.
// Es un struct — un contenedor de datos con campos nombrados.
// Cada campo tiene un nombre y un tipo.
//
// En Go, los structs son VALORES (como int o string), no referencias.
// Si pasas un Result a una función, se copia completo.
// Si necesitas evitar la copia, pasas un puntero: *Result.
type Result struct {
	URL        string // La URL que se verificó
	StatusCode int    // El código HTTP recibido (200, 404, 500, etc.)
	Err        error  // Si hubo error de conexión (timeout, DNS, etc.)
}

// Worker es una goroutine que lee URLs del channel 'jobs', verifica
// cada una (obteniendo su código HTTP), y envía el resultado al
// channel 'results'.
//
// Parámetros:
//   - id:      número identificador del worker (para debug/logging)
//   - jobs:    channel de SOLO LECTURA (<-chan) de donde saca las URLs
//   - results: channel de SOLO ESCRITURA (chan<-) donde pone los resultados
//   - wg:      puntero a WaitGroup para avisar cuando este worker termine
//
// Sobre las direcciones del channel:
//
//	<-chan string   → solo puedes LEER de este channel (recibir)
//	chan<- Result   → solo puedes ESCRIBIR en este channel (enviar)
//	chan string     → puedes leer Y escribir (bidireccional)
//
// Restringir la dirección evita bugs: si un worker intenta cerrar
// el channel de jobs (que no le corresponde), el compilador lo impide.
//
// Sobre *sync.WaitGroup (puntero):
//
//	WaitGroup es un struct. Si lo pasaras por valor (sin *), cada worker
//	tendría su PROPIA COPIA, y el wg.Done() de cada copia no afectaría
//	al WaitGroup original en main → main nunca sabría que terminaron.
//	Con puntero, todos apuntan al MISMO WaitGroup.
//
// Comportamiento esperado:
//  1. Usa un for-range sobre el channel 'jobs' para leer URLs.
//     (for-range sobre un channel itera hasta que el channel se CIERRA)
//  2. Para cada URL, obtén el código HTTP (por ahora, simúlalo o llama
//     a una función CheckURL que crearemos en la Fase 4).
//  3. Envía un Result{} al channel 'results'.
//  4. Cuando el channel 'jobs' se cierre y no queden URLs, el for-range
//     termina. Ahí debes llamar a wg.Done() para avisar que este worker
//     acabó. PISTA: defer es tu mejor amigo aquí — ¿por qué?
//
// Pistas de stdlib:
//   - sync.WaitGroup: wg.Done() decrementa el contador
//   - for url := range jobs { ... } itera sobre un channel
//   - defer se ejecuta al SALIR de la función
//
// NUEVA FIRMA: ctx se agrega como primer parámetro.
// El worker debe pasar ctx a CheckURL.
func Worker(ctx context.Context, id int, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	// TODO: tu código aquí
	defer wg.Done()
	for URL := range jobs {
		code, err := CheckURL(ctx, URL)
		instancia := Result{
			URL:        URL,
			StatusCode: code,
			Err:        err,
		}
		results <- instancia
	}

}

// RunWorkers orquesta todo el proceso concurrente.
//
// Parámetros:
//   - urls:       slice de URLs a verificar (la lista de enlaces extraídos)
//   - numWorkers: cuántos workers lanzar en paralelo
//
// Retorna:
//   - []Result: slice con el resultado de verificar cada URL
//
// Comportamiento esperado:
//  1. Crear el channel 'jobs' (¿con qué buffer?)
//  2. Crear el channel 'results' (¿con qué buffer?)
//  3. Crear un sync.WaitGroup
//  4. Lanzar numWorkers goroutines, cada una ejecutando Worker()
//     (no olvides wg.Add(1) ANTES de lanzar cada goroutine)
//  5. Meter todas las URLs en el channel 'jobs'
//  6. Cerrar el channel 'jobs' (para que los workers sepan que no hay más)
//  7. Esperar a que todos los workers terminen (wg.Wait())
//  8. Cerrar el channel 'results'
//  9. Leer todos los resultados del channel 'results' y retornarlos
//
// ⚠️  CUIDADO con el ORDEN de los pasos 7, 8, 9.
//
//	Piensa: ¿qué pasa si cierras 'results' ANTES de que los workers
//	terminen de escribir en él? → panic: send on closed channel
//
// ⚠️  CUIDADO con DEADLOCKS:
//
//	Si el channel de results no tiene buffer suficiente, y todos los
//	workers están intentando escribir, pero nadie está leyendo...
//	todos se bloquean esperando → deadlock.
//
//	Hay varias formas de resolverlo. Una pista: ¿qué pasa si el buffer
//	de results es del tamaño de len(urls)?
//
// Pistas de stdlib:
//   - make(chan Type, bufferSize) para crear channels con buffer
//   - sync.WaitGroup: Add(n), Done(), Wait()
//   - go func() { ... }() para lanzar goroutines
//   - close(channel) para cerrar un channel
//   - for result := range results { ... } para leer hasta que se cierre
//
// NUEVA FIRMA: ctx se agrega como primer parámetro.
// RunWorkers debe pasar ctx a cada Worker que lance.
func RunWorkers(ctx context.Context, urls []string, numWorkers int) []Result {
	// TODO: tu código aquí
	jobs := make(chan string, len(urls))
	results := make(chan Result, len(urls))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go Worker(ctx, i, jobs, results, &wg)

	}
	for _, url := range urls {
		jobs <- url
	}

	close(jobs)
	go func() {
		wg.Wait()
		close(results)
	}()
	var FinalResults []Result
	for res := range results {
		FinalResults = append(FinalResults, res)
	}
	return FinalResults
}
