package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
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
		entrada, ok := cacheEstaticosComprimidos.obtener(dir, ruta)
		if !ok || entrada.cuerpo == nil {
			siguiente.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", tipo)
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("ETag", entrada.etiqueta)
		http.ServeContent(w, r, ruta, entrada.modificado, bytes.NewReader(entrada.cuerpo))
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

func (c *cacheComprimidos) obtener(dir, ruta string) (entradaComprimida, bool) {
	fichero, err := http.Dir(dir).Open(ruta)
	if err != nil {
		return entradaComprimida{}, false
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < minimoBytesComprimir || info.Size() > maximoBytesComprimir {
		return entradaComprimida{}, false
	}
	clave := dir + "\x00" + ruta
	c.mu.Lock()
	previa, existe := c.entrada[clave]
	c.mu.Unlock()
	if existe && previa.tamano == info.Size() && previa.modificado.Equal(info.ModTime()) {
		return previa, true
	}
	original, err := io.ReadAll(io.LimitReader(fichero, maximoBytesComprimir+1))
	if err != nil || int64(len(original)) != info.Size() {
		return entradaComprimida{}, false
	}
	nueva := entradaComprimida{modificado: info.ModTime(), tamano: info.Size()}
	var comprimido bytes.Buffer
	escritor, _ := gzip.NewWriterLevel(&comprimido, gzip.BestCompression)
	if _, err := escritor.Write(original); err != nil || escritor.Close() != nil {
		return entradaComprimida{}, false
	}
	if comprimido.Len() < len(original) {
		resumen := sha256.Sum256(original)
		nueva.cuerpo = comprimido.Bytes()
		nueva.etiqueta = `"gz-` + hex.EncodeToString(resumen[:16]) + `"`
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if existe {
		c.bytes -= len(previa.cuerpo)
	}
	if c.bytes+len(nueva.cuerpo) > maximoBytesCacheComprimo {
		// Sin sitio: se sirve sin comprimir en vez de crecer sin límite.
		delete(c.entrada, clave)
		return entradaComprimida{}, false
	}
	c.bytes += len(nueva.cuerpo)
	c.entrada[clave] = nueva
	return nueva, true
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
