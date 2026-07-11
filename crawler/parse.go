package crawler

import (
	"net/url" // Para parsear y resolver URLs relativas a absolutas
	"strings" // Manipulación de strings (contiene, prefijo, etc.)

	// Esta es nuestra ÚNICA dependencia externa.
	// golang.org/x/net/html es un parser HTML del equipo de Go.
	// Recorre el HTML como un árbol de nodos (DOM), similar a
	// cómo funciona el navegador internamente.
	"golang.org/x/net/html"
)

// ExtractLinks recibe el HTML como string y la URL base (para resolver
// rutas relativas como "/about" → "https://go.dev/about").
//
// Retorna un []string con todas las URLs absolutas encontradas en
// etiquetas <a href="...">.
//
// []string es un "slice" de strings — un arreglo de tamaño dinámico.
// A diferencia de un array fijo [10]string, un slice puede crecer
// con append() según vamos encontrando enlaces.
func ExtractLinks(htmlBody string, baseURL string) ([]string, error) {

	// url.Parse convierte el string de la URL base en un struct *url.URL.
	// Este struct tiene campos separados: Scheme ("https"), Host ("go.dev"),
	// Path ("/docs"), etc. Lo necesitamos para resolver rutas relativas.
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
		// nil es el "valor cero" para slices, maps, punteros, etc.
		// Es como null en otros lenguajes. Un []string nil significa
		// "no hay lista, ni siquiera vacía".
	}

	// strings.NewReader envuelve nuestro string de HTML en un io.Reader.
	// ¿Qué es un io.Reader? Es una INTERFAZ — uno de los conceptos más
	// poderosos de Go. Cualquier cosa que implemente el método:
	//     Read(p []byte) (n int, err error)
	// es un io.Reader. Archivos, conexiones de red, strings... todos
	// pueden ser "leídos" de la misma forma.
	//
	// html.Parse espera un io.Reader, no un string directamente.
	// Esto es diseño Go: funciones genéricas que trabajan con cualquier
	// fuente de datos, no solo strings.
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	// Creamos nuestro slice vacío donde iremos acumulando los enlaces.
	// var links []string → declara un slice nil (sin memoria asignada).
	// Pero append() funciona bien con slices nil, así que no hay problema.
	var links []string

	// Usamos una técnica llamada "recursive descent" para recorrer
	// el árbol HTML. Definimos una función anónima (closure) que se
	// llama a sí misma recursivamente para visitar cada nodo.
	//
	// ¿Por qué una función anónima y no una función separada?
	// Porque esta función necesita acceso a "links" y "base" del scope
	// exterior — un closure "captura" esas variables automáticamente.
	// Es como las arrow functions en JS que acceden a variables del padre.
	var visit func(n *html.Node)
	visit = func(n *html.Node) {

		// html.ElementNode indica que este nodo es una etiqueta HTML
		// (como <a>, <div>, <p>), no texto ni comentario.
		// n.Data contiene el nombre de la etiqueta ("a", "div", etc.)
		if n.Type == html.ElementNode && n.Data == "a" {

			// Recorremos los atributos de la etiqueta <a>.
			// n.Attr es un slice de html.Attribute, cada uno con
			// Key (nombre del atributo) y Val (su valor).
			//
			// range es el "for...of" de Go: itera sobre slices, maps,
			// strings y channels. Retorna (índice, valor) en cada paso.
			// Usamos _ para ignorar el índice porque no lo necesitamos.
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue // Saltamos atributos que no son href
				}

				// Intentamos parsear el href encontrado como URL.
				href, err := url.Parse(attr.Val)
				if err != nil {
					continue // Si el href es basura, lo ignoramos
				}

				// base.ResolveReference convierte URLs relativas a absolutas:
				// - "/about"          → "https://go.dev/about"
				// - "docs/faq"        → "https://go.dev/docs/faq"
				// - "https://x.com"   → "https://x.com" (ya es absoluta, no cambia)
				resolved := base.ResolveReference(href)

				// Solo nos interesan enlaces HTTP/HTTPS.
				// Descartamos mailto:, javascript:, tel:, #fragmentos, etc.
				if resolved.Scheme == "http" || resolved.Scheme == "https" {
					links = append(links, resolved.String())
				}
			}
		}

		// Recorremos recursivamente todos los hijos de este nodo.
		// n.FirstChild apunta al primer hijo; c.NextSibling al hermano.
		// Es una lista enlazada (linked list), no un slice.
		//
		// Este for sin condición de "init" ni "post" es válido en Go:
		//     for condición { ... }
		// es equivalente a while(condición) en otros lenguajes.
		// Go NO tiene while — usa for para todo tipo de bucles.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}

	// Arrancamos la recursión desde la raíz del documento.
	visit(doc)

	return links, nil
}
