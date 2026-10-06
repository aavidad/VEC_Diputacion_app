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
	"time"
)

// Tipos de texto que Go reconoce por extensión (ServeContent deduce el tipo
// por el nombre; si no lo conociera lo adivinaría de los bytes comprimidos).
var extensionesComprimibles = map[string]bool{
	".js": true, ".mjs": true, ".css": true, ".json": true, ".html": true, ".svg": true,
}

type estaticoComprimido struct {
	modificado time.Time
	tamano     int64
	gzip       []byte // vacío: el fichero no gana nada comprimido
}

// cacheEstaticosComprimidos guarda el gzip de cada estático de texto ligado a
// su fecha y tamaño; si el fichero cambia se rehace. Los estáticos son iguales
// para todos y sin secretos, así que comprimirlos no expone a BREACH. La
// memoria la acota el propio contenido de web/static (unos 8 MiB de texto).
type cacheEstaticosComprimidos struct {
	entradas sync.Map // ruta absoluta -> *estaticoComprimido
}

// servir entrega la versión gzip si el navegador la acepta; en otro caso
// delega en el servidor de ficheros normal (redirecciones, rangos, errores).
func (c *cacheEstaticosComprimidos) servir(w http.ResponseWriter, r *http.Request, directorio string, siguiente http.Handler) {
	nombre := r.URL.Path
	if strings.HasSuffix(nombre, "/") {
		nombre += "index.html"
	}
	if !extensionesComprimibles[strings.ToLower(path.Ext(nombre))] {
		siguiente.ServeHTTP(w, r)
		return
	}
	w.Header().Add("Vary", "Accept-Encoding")
	if directorio == "" || !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") ||
		r.Header.Get("Range") != "" || strings.HasSuffix(r.URL.Path, "/index.html") {
		siguiente.ServeHTTP(w, r)
		return
	}
	entrada := c.obtener(directorio, nombre)
	if entrada == nil || len(entrada.gzip) == 0 {
		siguiente.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	http.ServeContent(w, r, path.Base(nombre), entrada.modificado, bytes.NewReader(entrada.gzip))
}

func (c *cacheEstaticosComprimidos) obtener(directorio, nombre string) *estaticoComprimido {
	// http.Dir limpia la ruta como el servidor de ficheros y no deja salir
	// del directorio.
	nombre = path.Clean("/" + nombre)
	fichero, err := http.Dir(directorio).Open(nombre)
	if err != nil {
		return nil
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	clave := filepath.Join(directorio, filepath.FromSlash(nombre))
	if valor, ok := c.entradas.Load(clave); ok {
		if entrada := valor.(*estaticoComprimido); entrada.tamano == info.Size() && entrada.modificado.Equal(info.ModTime()) {
			return entrada
		}
	}
	contenido, err := io.ReadAll(fichero)
	if err != nil {
		return nil
	}
	var comprimido bytes.Buffer
	escritor := gzip.NewWriter(&comprimido)
	if _, err := escritor.Write(contenido); err != nil || escritor.Close() != nil {
		return nil
	}
	entrada := &estaticoComprimido{modificado: info.ModTime(), tamano: info.Size()}
	if comprimido.Len() < len(contenido) {
		entrada.gzip = comprimido.Bytes()
	}
	c.entradas.Store(clave, entrada)
	return entrada
}

// directorioEstaticos resuelve la carpeta igual que staticFileServer.
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
