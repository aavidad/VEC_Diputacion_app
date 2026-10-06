package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

// Compresión gzip de los recursos estáticos de texto (JS, CSS, JSON, HTML,
// SVG). Solo se aplica a ficheros públicos del repositorio servidos tal cual:
// no contienen secretos ni reflejan entrada del cliente, así que no abren la
// vía de BREACH. Las respuestas de API nunca pasan por aquí.
//
// El resultado comprimido se calcula una vez por versión de fichero (ruta,
// fecha de modificación y tamaño) y se guarda en memoria con un tope global;
// al desplegar otra web cambian fecha o tamaño y se recalcula.

const (
	minimoBytesComprimir     = 1024
	maximoBytesComprimir     = 4 << 20
	maximoBytesCacheComprimo = 64 << 20
)

var extensionesComprimibles = map[string]struct{}{
	".js": {}, ".mjs": {}, ".css": {}, ".json": {}, ".html": {}, ".svg": {}, ".txt": {},
}

type entradaComprimida struct {
	modificado time.Time
	tamano     int64
	cuerpo     []byte // nil: no compensa comprimir esta versión
	etiqueta   string
}

type cacheComprimidos struct {
	mu      sync.Mutex
	entrada map[string]entradaComprimida
	bytes   int
}

var cacheEstaticosComprimidos = &cacheComprimidos{entrada: map[string]entradaComprimida{}}

// comprimirEstaticos envuelve un servidor de ficheros sobre raiz. Si el
// cliente admite gzip y el recurso es de texto, responde con la variante
// comprimida (conservando 304 por If-None-Match/If-Modified-Since); en
// cualquier otro caso delega en siguiente sin cambios.
func comprimirEstaticos(raiz func() (string, bool), siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ruta := path.Clean("/" + r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/") {
			ruta = path.Join(ruta, "index.html")
		} else if strings.HasSuffix(r.URL.Path, "/index.html") {
			// http.FileServer redirige index.html a su directorio.
			siguiente.ServeHTTP(w, r)
			return
		}
		ext := strings.ToLower(path.Ext(ruta))
		if _, comprimible := extensionesComprimibles[ext]; !comprimible ||
			(r.Method != http.MethodGet && r.Method != http.MethodHead) {
			siguiente.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Accept-Encoding")
		tipo := mime.TypeByExtension(ext)
		dir, hay := raiz()
		if !hay || tipo == "" || r.Header.Get("Range") != "" || !aceptaGzip(r.Header) {
			siguiente.ServeHTTP(w, r)
			return
		}
		// Cualquier fallo al preparar la variante comprimida deja la
		// respuesta como antes: el servidor de ficheros decide 200/404.
		if entrada, err := cacheEstaticosComprimidos.obtener(dir, ruta); err == nil && entrada.cuerpo != nil {
			w.Header().Set("Content-Type", tipo)
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("ETag", entrada.etiqueta)
			http.ServeContent(w, r, ruta, entrada.modificado, bytes.NewReader(entrada.cuerpo))
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}

// aceptaGzip interpreta Accept-Encoding de forma conservadora: gzip (o *)
// presente y sin q=0.
func aceptaGzip(cabeceras http.Header) bool {
	for _, valor := range cabeceras.Values("Accept-Encoding") {
		for _, parte := range strings.Split(valor, ",") {
			campos := strings.Split(parte, ";")
			nombre := strings.ToLower(strings.TrimSpace(campos[0]))
			if nombre != "gzip" && nombre != "*" {
				continue
			}
			rechazado := false
			for _, parametro := range campos[1:] {
				p := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(parametro)), " ", "")
				if p == "q=0" || p == "q=0.0" || p == "q=0.00" || p == "q=0.000" {
					rechazado = true
				}
			}
			if !rechazado {
				return true
			}
		}
	}
	return false
}

// errSinVariante indica que el fichero se sirve sin comprimir (no regular,
// tamaño fuera de rango, lectura incompleta o caché lleno).
var errSinVariante = errors.New("estaticos: sin variante comprimida")

func (c *cacheComprimidos) obtener(dir, ruta string) (entradaComprimida, error) {
	fichero, err := http.Dir(dir).Open(ruta)
	if err != nil {
		return entradaComprimida{}, err
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil {
		return entradaComprimida{}, err
	}
	if !info.Mode().IsRegular() || info.Size() < minimoBytesComprimir || info.Size() > maximoBytesComprimir {
		return entradaComprimida{}, errSinVariante
	}
	clave := dir + "\x00" + ruta
	// Todo el fallo de caché va bajo el mismo bloqueo: cada versión se
	// comprime una sola vez aunque lleguen muchas peticiones a la vez, y la
	// cuenta de bytes refleja exactamente lo guardado. Ocurre una vez por
	// fichero tras arrancar o desplegar.
	c.mu.Lock()
	defer c.mu.Unlock()
	previa, existe := c.entrada[clave]
	if existe && previa.tamano == info.Size() && previa.modificado.Equal(info.ModTime()) {
		return previa, nil
	}
	if existe {
		c.bytes -= len(previa.cuerpo)
		delete(c.entrada, clave)
	}
	original, err := io.ReadAll(io.LimitReader(fichero, maximoBytesComprimir+1))
	if err != nil {
		return entradaComprimida{}, err
	}
	if int64(len(original)) != info.Size() {
		return entradaComprimida{}, errSinVariante
	}
	nueva := entradaComprimida{modificado: info.ModTime(), tamano: info.Size()}
	var comprimido bytes.Buffer
	escritor, _ := gzip.NewWriterLevel(&comprimido, gzip.BestCompression)
	if _, err := escritor.Write(original); err != nil {
		return entradaComprimida{}, err
	}
	if err := escritor.Close(); err != nil {
		return entradaComprimida{}, err
	}
	if comprimido.Len() < len(original) {
		nueva.cuerpo = bytes.Clone(comprimido.Bytes())
		resumen := sha256.Sum256(nueva.cuerpo)
		nueva.etiqueta = `"gz-` + hex.EncodeToString(resumen[:16]) + `"`
	}
	if c.bytes+len(nueva.cuerpo) > maximoBytesCacheComprimo {
		// Sin sitio: se sirve sin comprimir en vez de crecer sin límite.
		return entradaComprimida{}, errSinVariante
	}
	c.bytes += len(nueva.cuerpo)
	c.entrada[clave] = nueva
	return nueva, nil
}

// directorioEstaticos resuelve la raíz de web/static igual que
// staticFileServer.
func directorioEstaticos() (string, bool) {
	return primerDirectorio("web/static", "../../../web/static")
}

func directorioLocales() (string, bool) {
	return primerDirectorio("locales", "../../../locales")
}

func primerDirectorio(candidatos ...string) (string, bool) {
	for _, dir := range candidatos {
		info, err := os.Stat(dir)
		if err == nil && info.IsDir() {
			return dir, true
		}
	}
	return "", false
}
