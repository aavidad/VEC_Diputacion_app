package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

// La marca pertenece a las fuentes; solo la huella calculada llega al navegador.
const marcaVersionWeb = "VEC_WEB_VERSION"

var versionWeb = sync.OnceValue(func() *webVersionada {
	web, err := calcularVersionWeb(os.DirFS(directorioEstaticos()), cargarRutasWebProduccion())
	if err != nil {
		slog.Error("versión web no disponible", "error", err)
		return nil
	}
	return web
})

type webVersionada struct {
	version string
	textos  map[string][]byte
	gzip    sync.Map
}

// Solo se leen las rutas de la lista positiva. El orden y los nombres forman
// parte de la huella, también cuando cambia únicamente una hoja importada.
func calcularVersionWeb(raiz fs.FS, rutas map[string]struct{}) (*webVersionada, error) {
	nombres := make([]string, 0, len(rutas))
	for ruta := range rutas {
		if !strings.HasSuffix(ruta, "/") {
			nombres = append(nombres, ruta)
		}
	}
	sort.Strings(nombres)
	huella := sha256.New()
	web := &webVersionada{textos: make(map[string][]byte)}
	for _, nombre := range nombres {
		contenido, err := fs.ReadFile(raiz, strings.TrimPrefix(nombre, "/"))
		if err != nil {
			return nil, fmt.Errorf("leer recurso web: %w", err)
		}
		huella.Write([]byte(nombre + "\x00"))
		huella.Write(contenido)
		huella.Write([]byte("\x00"))
		if extensionesComprimibles[path.Ext(nombre)] && bytes.Contains(contenido, []byte(marcaVersionWeb)) {
			web.textos[nombre] = contenido
		}
	}
	web.version = hex.EncodeToString(huella.Sum(nil))[:16]
	for nombre, contenido := range web.textos {
		web.textos[nombre] = bytes.ReplaceAll(contenido, []byte(marcaVersionWeb), []byte(web.version))
	}
	return web, nil
}

// Conserva ServeContent (HEAD, rangos y condicionales), sin fechas heredadas
// del fichero: cambiar otra hoja también cambia estos bytes y su ETag.
func (web *webVersionada) servir(w http.ResponseWriter, r *http.Request) bool {
	nombre := r.URL.Path
	if strings.HasSuffix(nombre, "/") {
		nombre += "index.html"
	}
	contenido, existe := web.textos[nombre]
	if !existe {
		return false
	}
	w.Header().Set("ETag", `W/"`+web.version+`"`)
	w.Header().Add("Vary", "Accept-Encoding")
	if aceptaGzip(r.Header.Get("Accept-Encoding")) && r.Header.Get("Range") == "" {
		valor, existe := web.gzip.Load(nombre)
		if !existe {
			var salida bytes.Buffer
			escritor := gzip.NewWriter(&salida)
			if _, err := escritor.Write(contenido); err != nil {
				slog.Error("versión web: comprimir", "error", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				return true
			}
			if err := escritor.Close(); err != nil {
				slog.Error("versión web: cerrar compresión", "error", err)
				w.WriteHeader(http.StatusServiceUnavailable)
				return true
			}
			valor, _ = web.gzip.LoadOrStore(nombre, salida.Bytes())
		}
		comprimido := valor.([]byte)
		if len(comprimido) < len(contenido) {
			w.Header().Set("Content-Encoding", "gzip")
			contenido = comprimido
		}
	}
	http.ServeContent(w, r, path.Base(nombre), time.Time{}, bytes.NewReader(contenido))
	return true
}
