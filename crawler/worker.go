package crawler

import (
	"sync"
)

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
func Worker(id int, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	// TODO: tu código aquí
	defer wg.Done()
	for URL := range jobs {
		instancia := Result{
			URL:        URL,
			StatusCode: 0,
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
func RunWorkers(urls []string, numWorkers int) []Result {
	// TODO: tu código aquí
	jobs := make(chan string, len(urls))
	results := make(chan Result, len(urls))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go Worker(i, jobs, results, &wg)

	}
	for _, url := range urls {
		jobs <- url
	}

	close(jobs)
	wg.Wait()
	close(results)
	var FinalResults []Result
	for res := range results {
		FinalResults = append(FinalResults, res)
	}
	return FinalResults
}
