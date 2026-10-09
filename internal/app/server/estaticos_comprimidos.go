package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"log/slog"
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
		// Sin versión comprimida: el servidor de ficheros normal responde
		// (y da el 404 si no existe). Solo se registra lo que no es «no existe».
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Warn("estatico comprimido no disponible", "etapa", "abrir", "ruta", nombre, "error", err)
		}
		return nil
	}
	defer fichero.Close()
	info, err := fichero.Stat()
	if err != nil {
		slog.Warn("estatico comprimido no disponible", "etapa", "stat", "ruta", nombre, "error", err)
		return nil
	}
	if !info.Mode().IsRegular() {
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
		slog.Warn("estatico comprimido no disponible", "etapa", "leer", "ruta", nombre, "error", err)
		return nil
	}
	var comprimido bytes.Buffer
	escritor := gzip.NewWriter(&comprimido)
	if _, err := escritor.Write(contenido); err != nil {
		slog.Warn("estatico comprimido no disponible", "etapa", "comprimir", "ruta", nombre, "error", err)
		return nil
	}
	if err := escritor.Close(); err != nil {
		slog.Warn("estatico comprimido no disponible", "etapa", "comprimir", "ruta", nombre, "error", err)
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

// cacheSegunVersion: con ?v= la URL cambia con el contenido y se guarda un
// año sin preguntar; sin versión se guarda pero se revalida siempre (304).
func cacheSegunVersion(r *http.Request) string {
	if r.URL.Query().Get("v") != "" {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

// cacheCatalogoTextos: con ?huella= igual a la de los catálogos que sirve este
// proceso (VERSION_TEXTOS de comun/textos.js) se guardan un año. Cualquier otra
// huella, sin huella o con los ?v= de los manifiestos PWA, se revalida (304):
// así un despliegue a medias o una huella inventada nunca fijan contenido viejo.
func cacheCatalogoTextos(r *http.Request) string {
	if pedida := r.URL.Query().Get("huella"); pedida != "" && pedida == huellaCatalogosTextos() {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

// huellaCatalogosTextos se calcula una vez por proceso: cambiar un catálogo
// exige desplegar y reiniciar, como el resto de estáticos.
var huellaCatalogosTextos = sync.OnceValue(func() string {
	directorio := directorioEstaticos()
	if directorio == "" {
		return ""
	}
	return calcularHuellaCatalogosTextos(os.DirFS(filepath.Join(directorio, "textos")))
})

// calcularHuellaCatalogosTextos replica comun/textos-version.test.mjs: SHA-256
// de "<idioma>/<fichero>\0<bytes>\0" de cada <idioma>/*.json en orden, 16 hex.
// Ante cualquier error devuelve "" y ninguna huella se acepta.
func calcularHuellaCatalogosTextos(raiz fs.FS) string {
	huella := sha256.New()
	idiomas, err := fs.ReadDir(raiz, ".")
	if err != nil {
		return ""
	}
	for _, idioma := range idiomas {
		if !idioma.IsDir() {
			continue
		}
		ficheros, err := fs.ReadDir(raiz, idioma.Name())
		if err != nil {
			return ""
		}
		for _, fichero := range ficheros {
			if fichero.IsDir() || !strings.HasSuffix(fichero.Name(), ".json") {
				continue
			}
			ruta := idioma.Name() + "/" + fichero.Name()
			contenido, err := fs.ReadFile(raiz, ruta)
			if err != nil {
				return ""
			}
			huella.Write([]byte(ruta + "\x00"))
			huella.Write(contenido)
			huella.Write([]byte("\x00"))
		}
	}
	return hex.EncodeToString(huella.Sum(nil))[:16]
}

// fijarCacheEstatico sustituye la política no-store que securityHeaders pone
// a toda respuesta y retira su Pragma: no-cache. Ese Pragma, heredado de
// HTTP/1.0, junto a una política almacenable hacía que el navegador volviera
// a preguntar por cada estático en cada recarga. Las API lo conservan.
func fijarCacheEstatico(w http.ResponseWriter, politica string) {
	w.Header().Set("Cache-Control", politica)
	w.Header().Del("Pragma")
}

// directorioLocales devuelve el directorio de traducciones, o "" si no hay.
func directorioLocales() string {
	for _, dir := range []string{"locales", "../../../locales"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			if absoluto, err := filepath.Abs(dir); err == nil {
				return absoluto
			}
		}
	}
	return ""
}
