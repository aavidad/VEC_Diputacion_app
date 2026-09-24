package server

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
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
	maxBytesTeselaOSM   = 1024 * 1024
	maxBytesZIPOSM      = 2 * 1024 * 1024 * 1024
	maxEntradasZIPOSM   = 4096
	maxDirectorioZIPOSM = 1024 * 1024
	maxConcurrentesOSM  = 16
)

var firmaPNGTeselaOSM = []byte{137, 80, 78, 71, 13, 10, 26, 10}

var errZIPTeselasOSMInvalido = errors.New("archivo de teselas no válido")

type indiceTeselasOSM struct {
	mu       sync.Mutex
	archivo  *os.File // El índice conserva el ReaderAt hasta terminar el proceso.
	entradas map[string]*zip.File
	cupos    chan struct{}
}

// registrarTeselasOSM publica sólo el mapa base, sin geometrías ni datos de
// Dietas. La frontera HTTP interna ya protege este listener; la ruta pública
// no se registra. El ZIP es un artefacto fijo y no se toma de la petición.
func registrarTeselasOSM(mux *http.ServeMux) {
	mux.Handle("/tiles/osm/", soloLecturaHTTP(suprimirCuerpoHEAD(manejadorTeselasOSM())))
}

func manejadorTeselasOSM() http.Handler {
	indice := &indiceTeselasOSM{cupos: make(chan struct{}, maxConcurrentesOSM)}
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
	if indice.entradas == nil {
		archivo, err := abrirZIPTeselasOSM()
		if err != nil {
			return nil, err
		}
		info, err := archivo.Stat()
		if err != nil {
			_ = archivo.Close()
			return nil, err
		}
		cantidad, err := validarDirectorioZIPTeselasOSM(archivo, info.Size())
		if err != nil {
			_ = archivo.Close()
			return nil, err
		}
		lector, err := zip.NewReader(archivo, info.Size())
		if err != nil || len(lector.File) != cantidad {
			_ = archivo.Close()
			return nil, errZIPTeselasOSMInvalido
		}
		entradas := make(map[string]*zip.File, cantidad)
		for _, entrada := range lector.File {
			if _, duplicada := entradas[entrada.Name]; duplicada {
				_ = archivo.Close()
				return nil, errZIPTeselasOSMInvalido
			}
			entradas[entrada.Name] = entrada
		}
		indice.archivo = archivo
		indice.entradas = entradas
	}
	return indice.entradas[nombre], nil
}

// Valida el EOCD y recorre la cabecera central sin construir el índice ZIP.
// Rechaza ZIP64, multidisco, contadores falsos y directorios excesivos antes
// de que archive/zip asigne memoria proporcional al número de entradas.
func validarDirectorioZIPTeselasOSM(archivo *os.File, tamano int64) (int, error) {
	if tamano < 22 || tamano > maxBytesZIPOSM {
		return 0, errZIPTeselasOSMInvalido
	}
	longitud := tamano
	if longitud > 22+65535 {
		longitud = 22 + 65535
	}
	cola := make([]byte, longitud)
	if _, err := archivo.ReadAt(cola, tamano-longitud); err != nil {
		return 0, err
	}
	for i := len(cola) - 22; i >= 0; i-- {
		if binary.LittleEndian.Uint32(cola[i:]) != 0x06054b50 ||
			i+22+int(binary.LittleEndian.Uint16(cola[i+20:])) != len(cola) {
			continue
		}
		eocd := cola[i : i+22]
		cantidad := int(binary.LittleEndian.Uint16(eocd[10:]))
		tamanoCentral := int64(binary.LittleEndian.Uint32(eocd[12:]))
		inicioCentral := int64(binary.LittleEndian.Uint32(eocd[16:]))
		posEOCD := tamano - longitud + int64(i)
		if binary.LittleEndian.Uint16(eocd[4:]) != 0 ||
			binary.LittleEndian.Uint16(eocd[6:]) != 0 ||
			binary.LittleEndian.Uint16(eocd[8:]) != uint16(cantidad) ||
			cantidad == 0 || cantidad > maxEntradasZIPOSM ||
			tamanoCentral <= 0 || tamanoCentral > maxDirectorioZIPOSM ||
			inicioCentral < 0 || inicioCentral+tamanoCentral != posEOCD {
			return 0, errZIPTeselasOSMInvalido
		}
		pos := inicioCentral
		var cabecera [46]byte
		for j := 0; j < cantidad; j++ {
			if pos+int64(len(cabecera)) > posEOCD {
				return 0, errZIPTeselasOSMInvalido
			}
			if _, err := archivo.ReadAt(cabecera[:], pos); err != nil || binary.LittleEndian.Uint32(cabecera[:]) != 0x02014b50 {
				return 0, errZIPTeselasOSMInvalido
			}
			pos += int64(len(cabecera)) +
				int64(binary.LittleEndian.Uint16(cabecera[28:])) +
				int64(binary.LittleEndian.Uint16(cabecera[30:])) +
				int64(binary.LittleEndian.Uint16(cabecera[32:]))
			if pos > posEOCD {
				return 0, errZIPTeselasOSMInvalido
			}
		}
		if pos != posEOCD {
			return 0, errZIPTeselasOSMInvalido
		}
		return cantidad, nil
	}
	return 0, errZIPTeselasOSMInvalido
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
