package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	nombreZIPTeselasOSM = "granada-base-20260719-z8-z12.zip"
	sha256ZIPTeselasOSM = "0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165"
	maxBytesTeselaOSM   = 1024 * 1024
	maxBytesZIPOSM      = 32 * 1024 * 1024
	maxEntradasZIPOSM   = 4096
	maxConcurrentesOSM  = 16
)

var firmaPNGTeselaOSM = []byte{137, 80, 78, 71, 13, 10, 26, 10}

var errZIPTeselasOSMInvalido = errors.New("archivo de teselas no válido")

type indiceTeselasOSM struct {
	mu           sync.Mutex
	datos        []byte // Inmutables: mismo contenido validado que consume archive/zip.
	entradas     map[string]*zip.File
	errorCarga   error
	hashEsperado string
	cupos        chan struct{}
}

// registrarTeselasOSM publica sólo el mapa base, sin geometrías ni datos de
// Dietas. La frontera HTTP interna ya protege este listener; la ruta pública
// no se registra. El ZIP es un artefacto fijo y no se toma de la petición.
func registrarTeselasOSMConHash(mux *http.ServeMux, hashZIP string) {
	mux.Handle("/tiles/osm/", soloLecturaHTTP(suprimirCuerpoHEAD(manejadorTeselasOSMConHash(hashZIP))))
}

func manejadorTeselasOSMConHash(hashZIP string) http.Handler {
	indice := &indiceTeselasOSM{hashEsperado: hashZIP, cupos: make(chan struct{}, maxConcurrentesOSM)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entrada, valida := entradaTeselaOSM(r.URL.Path)
		if !valida {
			http.NotFound(w, r)
			return
		}
		select {
		case indice.cupos <- struct{}{}:
			defer func() { <-indice.cupos }()
		default:
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		archivo, err := indice.obtener(entrada)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if archivo == nil {
			http.NotFound(w, r)
			return
		}
		contenido, ok := leerEntradaTeselaOSM(archivo)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(contenido)
		}
	})
}

func (indice *indiceTeselasOSM) obtener(nombre string) (*zip.File, error) {
	indice.mu.Lock()
	defer indice.mu.Unlock()
	if indice.errorCarga != nil {
		return nil, indice.errorCarga
	}
	if indice.entradas == nil {
		archivo, err := abrirZIPTeselasOSM()
		if err != nil {
			return nil, err
		}
		defer archivo.Close()
		info, err := archivo.Stat()
		if err != nil {
			return nil, err
		}
		datos, err := leerZIPTeselasOSMValidado(archivo, info.Size(), indice.hashEsperado)
		if err != nil {
			indice.errorCarga = err
			return nil, err
		}
		lector, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
		if err != nil || len(lector.File) == 0 || len(lector.File) > maxEntradasZIPOSM {
			indice.errorCarga = errZIPTeselasOSMInvalido
			return nil, errZIPTeselasOSMInvalido
		}
		entradas := make(map[string]*zip.File, len(lector.File))
		for _, entrada := range lector.File {
			if _, duplicada := entradas[entrada.Name]; duplicada {
				indice.errorCarga = errZIPTeselasOSMInvalido
				return nil, errZIPTeselasOSMInvalido
			}
			entradas[entrada.Name] = entrada
		}
		indice.datos = datos
		indice.entradas = entradas
	}
	return indice.entradas[nombre], nil
}

// El hash fijo se comprueba sobre los mismos bytes que leerá archive/zip.
// Ningún ZIP alterado, incluido uno con finales EOCD ambiguos, alcanza el
// parser ni puede provocar asignaciones proporcionales a entradas falsas.
func leerZIPTeselasOSMValidado(archivo *os.File, tamano int64, hashEsperado string) ([]byte, error) {
	if tamano < 22 || tamano > maxBytesZIPOSM {
		return nil, errZIPTeselasOSMInvalido
	}
	datos := make([]byte, int(tamano))
	if _, err := archivo.ReadAt(datos, 0); err != nil {
		return nil, err
	}
	suma := sha256.Sum256(datos)
	if hex.EncodeToString(suma[:]) != hashEsperado {
		return nil, errZIPTeselasOSMInvalido
	}
	return datos, nil
}

func entradaTeselaOSM(ruta string) (string, bool) {
	const prefijo = "/tiles/osm/"
	if !strings.HasPrefix(ruta, prefijo) || !strings.HasSuffix(ruta, ".png") {
		return "", false
	}
	partes := strings.Split(strings.TrimSuffix(strings.TrimPrefix(ruta, prefijo), ".png"), "/")
	if len(partes) != 3 {
		return "", false
	}
	coordenadas := [3]uint64{}
	for i, parte := range partes {
		if parte == "" {
			return "", false
		}
		valor, err := strconv.ParseUint(parte, 10, 64)
		if err != nil || strconv.FormatUint(valor, 10) != parte {
			return "", false
		}
		coordenadas[i] = valor
	}
	z := coordenadas[0]
	if z < 8 || z > 12 || coordenadas[1] >= 1<<z || coordenadas[2] >= 1<<z {
		return "", false
	}
	return "tiles/" + strings.Join(partes, "/") + ".png", true
}

func abrirZIPTeselasOSM() (*os.File, error) {
	for _, proyecto := range []string{".", "../../.."} {
		archivo, err := abrirZIPTeselasOSMEnProyecto(proyecto)
		if err == nil {
			return archivo, nil
		}
	}
	return nil, os.ErrNotExist
}

func abrirZIPTeselasOSMEnProyecto(proyecto string) (*os.File, error) {
	raiz, err := os.OpenRoot(proyecto)
	if err != nil {
		return nil, err
	}
	defer raiz.Close()
	web, err := abrirDirectorioOSMSinEnlace(raiz, "web")
	if err != nil {
		return nil, err
	}
	defer web.Close()
	cartografia, err := abrirDirectorioOSMSinEnlace(web, "cartografia")
	if err != nil {
		return nil, err
	}
	defer cartografia.Close()
	info, err := cartografia.Lstat(nombreZIPTeselasOSM)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxBytesZIPOSM {
		return nil, os.ErrNotExist
	}
	// os.Root impide que un enlace cambiado entre Lstat y Open salga del
	// directorio cartografico. No se leen rutas proporcionadas por clientes.
	archivo, err := cartografia.Open(nombreZIPTeselasOSM)
	if err != nil {
		return nil, err
	}
	info, err = archivo.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxBytesZIPOSM {
		_ = archivo.Close()
		return nil, os.ErrNotExist
	}
	return archivo, nil
}

func abrirDirectorioOSMSinEnlace(raiz *os.Root, nombre string) (*os.Root, error) {
	info, err := raiz.Lstat(nombre)
	if err != nil || !info.IsDir() {
		return nil, os.ErrNotExist
	}
	return raiz.OpenRoot(nombre)
}

func leerEntradaTeselaOSM(encontrada *zip.File) ([]byte, bool) {
	if !encontrada.FileInfo().Mode().IsRegular() || encontrada.UncompressedSize64 > maxBytesTeselaOSM {
		return nil, false
	}
	archivo, err := encontrada.Open()
	if err != nil {
		return nil, false
	}
	defer archivo.Close()
	contenido, err := io.ReadAll(io.LimitReader(archivo, maxBytesTeselaOSM+1))
	if err != nil || len(contenido) > maxBytesTeselaOSM || !bytes.HasPrefix(contenido, firmaPNGTeselaOSM) {
		return nil, false
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(contenido))
	if err != nil || cfg.Width != 256 || cfg.Height != 256 {
		return nil, false
	}
	lectorPNG := bytes.NewReader(contenido)
	if _, err := png.Decode(lectorPNG); err != nil || lectorPNG.Len() != 0 {
		return nil, false
	}
	return contenido, true
}
