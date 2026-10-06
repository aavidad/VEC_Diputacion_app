package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Solo merece la pena comprimir a partir de este tamaño: por debajo, la
	// cabecera gzip y el trabajo superan el ahorro.
	tamanoMinimoEstaticoComprimible = 1 << 10
	// Ningún recurso de texto del portal se acerca a este tamaño; uno mayor
	// se sirve tal cual en lugar de cargarlo entero en memoria.
	tamanoMaximoEstaticoComprimible = 8 << 20
	// Tope de memoria de la caché comprimida por manejador. Los manifiestos
	// suman hoy unos 2 MiB comprimidos; al alcanzarlo se sirve sin comprimir.
	presupuestoCacheEstaticosComprimidos = 64 << 20
)

// extensionesComprimibles son los recursos de texto del portal cuyo tipo
// conoce la tabla interna de Go: ServeContent lo deduce por la extensión y,
// si no la conociera, lo adivinaría a partir de los bytes ya comprimidos.
// Imágenes y tipografías ya van comprimidas y no se tocan.
var extensionesComprimibles = map[string]struct{}{
	".js": {}, ".mjs": {}, ".css": {}, ".json": {}, ".html": {}, ".svg": {},
}

type estaticoComprimido struct {
	modificado time.Time
	tamano     int64
	// gzip vacío marca un fichero que no gana nada comprimido: se recuerda
	// para no volver a intentarlo en cada petición.
	gzip []byte
}

// cacheEstaticosComprimidos guarda la versión gzip de cada fichero estático
// de texto, ligada a su fecha y tamaño en disco: si el fichero cambia se
// vuelve a comprimir. Los estáticos son el mismo contenido para cualquier
// visitante (sin datos personales ni secretos), así que comprimirlos no abre
// ataques del tipo BREACH, que necesitan un secreto en la misma respuesta.
type cacheEstaticosComprimidos struct {
	entradas sync.Map // ruta absoluta -> *estaticoComprimido
	ocupado  atomic.Int64
}

func aceptaGzip(r *http.Request) bool {
	for _, parte := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		codificacion, parametros, _ := strings.Cut(strings.TrimSpace(parte), ";")
		if !strings.EqualFold(strings.TrimSpace(codificacion), "gzip") {
			continue
		}
		parametros = strings.ReplaceAll(strings.TrimSpace(parametros), " ", "")
		return parametros != "q=0" && parametros != "q=0.0" && parametros != "q=0.00" && parametros != "q=0.000"
	}
	return false
}

// servir entrega la versión gzip si el navegador la acepta y el fichero lo
// merece; en cualquier otro caso delega en el servidor de ficheros normal,
// que conserva redirecciones de directorio, rangos y errores.
func (c *cacheEstaticosComprimidos) servir(w http.ResponseWriter, r *http.Request, directorio string, siguiente http.Handler) {
	ruta := r.URL.Path
	nombre := ruta
	if strings.HasSuffix(ruta, "/") {
		nombre = ruta + "index.html"
	}
	_, comprimible := extensionesComprimibles[strings.ToLower(path.Ext(nombre))]
	if comprimible {
		w.Header().Add("Vary", "Accept-Encoding")
	}
	if c == nil || directorio == "" || !comprimible || !aceptaGzip(r) || r.Header.Get("Range") != "" ||
		strings.HasSuffix(ruta, "/index.html") {
		siguiente.ServeHTTP(w, r)
		return
	}
	entrada, ok := c.obtener(directorio, nombre)
	if !ok {
		siguiente.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	http.ServeContent(w, r, path.Base(nombre), entrada.modificado, bytes.NewReader(entrada.gzip))
}

func (c *cacheEstaticosComprimidos) obtener(directorio, nombre string) (*estaticoComprimido, bool) {
	// http.Dir aplica la misma limpieza de ruta que el servidor de ficheros
	// y rechaza los intentos de salir del directorio.
	fichero, err := http.Dir(directorio).Open(path.Clean("/" + nombre))
	if err != nil {
		return nil, false
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < tamanoMinimoEstaticoComprimible ||
		info.Size() > tamanoMaximoEstaticoComprimible {
		return nil, false
	}
	clave := filepath.Join(directorio, filepath.FromSlash(path.Clean("/"+nombre)))
	if valor, ok := c.entradas.Load(clave); ok {
		entrada := valor.(*estaticoComprimido)
		if entrada.tamano == info.Size() && entrada.modificado.Equal(info.ModTime()) {
			return entrada, len(entrada.gzip) > 0
		}
	}
	contenido, err := io.ReadAll(io.LimitReader(fichero, tamanoMaximoEstaticoComprimible+1))
	if err != nil || int64(len(contenido)) != info.Size() {
		return nil, false
	}
	var comprimido bytes.Buffer
	escritor, err := gzip.NewWriterLevel(&comprimido, gzip.BestCompression)
	if err != nil {
		return nil, false
	}
	if _, err := escritor.Write(contenido); err != nil || escritor.Close() != nil {
		return nil, false
	}
	entrada := &estaticoComprimido{modificado: info.ModTime(), tamano: info.Size(), gzip: comprimido.Bytes()}
	if comprimido.Len() >= len(contenido) {
		entrada.gzip = nil
	}
	if anterior, cargada := c.entradas.Swap(clave, entrada); cargada {
		c.ocupado.Add(-int64(len(anterior.(*estaticoComprimido).gzip)))
	}
	if c.ocupado.Add(int64(len(entrada.gzip))) > presupuestoCacheEstaticosComprimidos {
		// Sin espacio: se retira y se sirve sin comprimir. No se expulsan
		// otras entradas para que la memoria quede acotada sin más lógica.
		if c.entradas.CompareAndDelete(clave, entrada) {
			c.ocupado.Add(-int64(len(entrada.gzip)))
		}
		return nil, false
	}
	return entrada, len(entrada.gzip) > 0
}

// directorioEstaticos resuelve la carpeta de estáticos igual que
// staticFileServer: desde la raíz del repositorio o desde un paquete de pruebas.
func directorioEstaticos() string {
	for _, dir := range []string{"web/static", "../../../web/static"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			if absoluto, err := filepath.Abs(dir); err == nil {
				return absoluto
			}
		}
	}
	return ""
}
