package crawler

// Importamos varios paquetes de la biblioteca estándar de Go.
// Cuando necesitas más de uno, se agrupan dentro de paréntesis.
import (
	"fmt"      // Para formatear strings (como sprintf en C o f-strings en Python)
	"io"       // Input/Output: leer y copiar flujos de datos (streams)
	"net/http" // Cliente y servidor HTTP — todo lo que necesitas para hacer requests
)

// FetchURL recibe una URL como string y devuelve el cuerpo HTML como string.
//
// Retorna DOS valores: (string, error)
// - Si todo sale bien: ("contenido HTML", nil)
// - Si algo falla:    ("", error con descripción del problema)
//
// En Go, las funciones exportadas (visibles fuera del paquete) empiezan con
// MAYÚSCULA. Si escribieras "fetchURL" (minúscula), solo sería visible
// dentro del paquete crawler. "FetchURL" con mayúscula es pública.
func FetchURL(url string) (string, error) {

	// http.Get() hace una petición GET a la URL.
	// Retorna un *http.Response (puntero a Response) y un error.
	//
	// ¿Por qué un puntero (*http.Response)?
	// Porque Response es un struct grande con headers, body, status, etc.
	// Pasar una copia sería costoso en memoria; el puntero es una
	// referencia al mismo dato en memoria (como en C, pero seguro).
	resp, err := http.Get(url)

	// Primer chequeo de error: ¿la petición falló?
	// Esto atrapa errores de red: DNS no resuelve, conexión rechazada,
	// timeout, URL malformada, etc.
	if err != nil {
		// fmt.Errorf crea un nuevo error con un mensaje formateado.
		// El %w "envuelve" (wraps) el error original, permitiendo que
		// quien llame esta función pueda inspeccionar el error raíz
		// con errors.Is() o errors.As() si lo necesita después.
		return "", fmt.Errorf("error al obtener %s: %w", url, err)
	}

	// defer ejecuta una función JUSTO ANTES de que la función actual retorne.
	// Es como un "finally" en otros lenguajes, pero más elegante:
	// lo escribes cerca del recurso que abriste, no al final de la función.
	//
	// ¿Por qué cerrar el Body?
	// resp.Body es un flujo de datos (stream) abierto con la conexión TCP.
	// Si no lo cierras, Go mantiene la conexión abierta → fuga de recursos.
	// Con defer, garantizas que se cierra SIN IMPORTAR cómo salga la función
	// (retorno normal o retorno por error más abajo).
	defer resp.Body.Close()

	// Verificamos que el servidor respondió con un status 2xx (éxito).
	// http.Get NO retorna error por un 404 o 500 — esos son respuestas
	// HTTP válidas. Solo retorna error por fallos de RED (no se pudo conectar).
	// Nosotros decidimos que un 404/500 en la URL principal SÍ es un error.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status no exitoso para %s: %d", url, resp.StatusCode)
	}

	// io.ReadAll lee TODO el contenido del Body y lo retorna como []byte.
	// []byte es un slice de bytes — piensa en él como un array dinámico
	// de bytes (la representación cruda del texto).
	body, err := io.ReadAll(resp.Body)

	// Segundo chequeo: ¿falló la lectura del body?
	// Puede pasar si la conexión se corta a mitad de la descarga.
	if err != nil {
		return "", fmt.Errorf("error leyendo body de %s: %w", url, err)
	}

	// string(body) convierte los bytes a string.
	// En Go, string y []byte son tipos diferentes:
	// - []byte: mutable, para manipular datos binarios
	// - string: inmutable, para texto que no va a cambiar
	// Aquí convertimos porque el HTML ya lo leímos completo y no lo
	// vamos a modificar byte por byte.
	return string(body), nil
}
