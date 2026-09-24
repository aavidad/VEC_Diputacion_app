package server

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/config"
)

func crearZIPTeselasOSMPrueba(t *testing.T, contenido []byte, nombre string) string {
	t.Helper()
	directorio := filepath.Join(t.TempDir(), "web", "cartografia")
	if err := os.MkdirAll(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(directorio, nombreZIPTeselasOSM)
	archivo, err := os.Create(ruta)
	if err != nil {
		t.Fatal(err)
	}
	escritor := zip.NewWriter(archivo)
	entrada, err := escritor.Create(nombre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entrada.Write(contenido); err != nil {
		t.Fatal(err)
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func pngTeselaOSMPrueba(t *testing.T) []byte {
	t.Helper()
	imagen := image.NewRGBA(image.Rect(0, 0, 256, 256))
	imagen.Set(0, 0, color.RGBA{R: 10, G: 30, B: 60, A: 255})
	var salida bytes.Buffer
	if err := png.Encode(&salida, imagen); err != nil {
		t.Fatal(err)
	}
	return salida.Bytes()
}

func TestTeselasOSMInternoIntegradoYPublico(t *testing.T) {
	contenido := pngTeselaOSMPrueba(t)
	rutaZIP := crearZIPTeselasOSMPrueba(t, contenido, "tiles/8/125/99.png")
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	for _, superficie := range []struct {
		nombre  string
		handler http.Handler
		estado  int
	}{
		{"interna", NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()), http.StatusOK},
		{"integrada", NewHandlerWithConfig(config.Config{}, http.NotFoundHandler()), http.StatusOK},
		{"publica", NewHandlerPublicoWithConfig(config.Config{}, http.NotFoundHandler()), http.StatusNotFound},
	} {
		for _, metodo := range []string{http.MethodGet, http.MethodHead} {
			t.Run(superficie.nombre+" "+metodo, func(t *testing.T) {
				respuesta := httptest.NewRecorder()
				superficie.handler.ServeHTTP(respuesta, peticionServidorPrueba(metodo, "/tiles/osm/8/125/99.png", nil))
				if respuesta.Code != superficie.estado {
					t.Fatalf("estado = %d; esperado %d", respuesta.Code, superficie.estado)
				}
				if superficie.estado != http.StatusOK {
					if bytes.Equal(respuesta.Body.Bytes(), contenido) {
						t.Fatal("la superficie pública sirvió la tesela")
					}
					return
				}
				if respuesta.Header().Get("Content-Type") != "image/png" || respuesta.Header().Get("Content-Length") != strconv.Itoa(len(contenido)) {
					t.Fatalf("cabeceras PNG incorrectas: %v", respuesta.Header())
				}
				if cache := respuesta.Header().Get("Cache-Control"); cache != "no-store" || strings.Contains(cache, "immutable") {
					t.Fatalf("caché de URL estable: %q", cache)
				}
				if respuesta.Header().Get("Set-Cookie") != "" {
					t.Fatal("respuesta con cookie")
				}
				if metodo == http.MethodHead && respuesta.Body.Len() != 0 {
					t.Fatal("HEAD devolvió cuerpo")
				}
				if metodo == http.MethodGet && !bytes.Equal(respuesta.Body.Bytes(), contenido) {
					t.Fatal("GET no devolvió la tesela exacta")
				}
			})
		}
	}
}

func TestTeselasOSMRechazaMetodoRutaYArchivoAusente(t *testing.T) {
	rutaZIP := crearZIPTeselasOSMPrueba(t, pngTeselaOSMPrueba(t), "tiles/8/125/99.png")
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	handler := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	for _, caso := range []struct {
		metodo, ruta string
		estado       int
	}{
		{http.MethodPost, "/tiles/osm/8/125/99.png", http.StatusMethodNotAllowed},
		{http.MethodGet, "/tiles/osm/08/125/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/0125/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/256/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/13/125/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/-1/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/125/99.PNG", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/125/99.png/otro", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/125/", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/../99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/%31%32%35/99.png", http.StatusNotFound},
		{http.MethodGet, "/tiles/osm/8/126/99.png", http.StatusNotFound},
	} {
		t.Run(caso.metodo+" "+caso.ruta, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticionServidorPrueba(caso.metodo, caso.ruta, nil))
			if respuesta.Code != caso.estado {
				t.Fatalf("estado = %d; esperado %d", respuesta.Code, caso.estado)
			}
		})
	}
	if err := os.Remove(rutaZIP); err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
	)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("ZIP ausente = %d; esperado 404", respuesta.Code)
	}
}

func TestTeselasOSMRechazaZIPEnlaceYContenidoNoPNG(t *testing.T) {
	rutaZIP := crearZIPTeselasOSMPrueba(t, []byte("contenido ajeno"), "tiles/8/125/99.png")
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	handler := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	consultar := func() int {
		respuesta := httptest.NewRecorder()
		handler.ServeHTTP(respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil))
		return respuesta.Code
	}
	if estado := consultar(); estado != http.StatusNotFound {
		t.Fatalf("contenido no PNG = %d", estado)
	}
	otraRuta := filepath.Join(t.TempDir(), "fuera.zip")
	if err := os.Rename(rutaZIP, otraRuta); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otraRuta, rutaZIP); err != nil {
		t.Fatal(err)
	}
	if estado := consultar(); estado != http.StatusNotFound {
		t.Fatalf("ZIP simbólico = %d", estado)
	}
}

func TestTeselasOSMRechazaDirectorioCartografiaSimbolico(t *testing.T) {
	rutaZIP := crearZIPTeselasOSMPrueba(t, pngTeselaOSMPrueba(t), "tiles/8/125/99.png")
	proyecto := t.TempDir()
	if err := os.Mkdir(filepath.Join(proyecto, "web"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(rutaZIP), filepath.Join(proyecto, "web", "cartografia")); err != nil {
		t.Fatal(err)
	}
	t.Chdir(proyecto)
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
	)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("cartografía enlazada fuera del proyecto = %d; esperado 404", respuesta.Code)
	}
}

func TestTeselasOSMRechazaPNGTruncadoYCorrupto(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		contenido func(*testing.T) []byte
	}{
		{"firma_sola", func(*testing.T) []byte { return append([]byte(nil), firmaPNGTeselaOSM...) }},
		{"crc_corrupto", func(t *testing.T) []byte {
			png := pngTeselaOSMPrueba(t)
			png[len(png)-8] ^= 0xff
			return png
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			rutaZIP := crearZIPTeselasOSMPrueba(t, caso.contenido(t), "tiles/8/125/99.png")
			t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
			respuesta := httptest.NewRecorder()
			NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
				respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
			)
			if respuesta.Code != http.StatusNotFound {
				t.Fatalf("PNG incompleto o corrupto = %d; esperado 404", respuesta.Code)
			}
		})
	}
}

func TestTeselasOSMRechazaZIPConExcesoDeEntradas(t *testing.T) {
	directorio := filepath.Join(t.TempDir(), "web", "cartografia")
	if err := os.MkdirAll(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	rutaZIP := filepath.Join(directorio, nombreZIPTeselasOSM)
	archivo, err := os.Create(rutaZIP)
	if err != nil {
		t.Fatal(err)
	}
	escritor := zip.NewWriter(archivo)
	for i := 0; i < 4097; i++ {
		nombre := "tiles/8/125/99.png"
		if i != 0 {
			nombre = "extra/" + strconv.Itoa(i)
		}
		entrada, err := escritor.Create(nombre)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if _, err := entrada.Write(pngTeselaOSMPrueba(t)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
	)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("ZIP con 4097 entradas = %d; esperado 404", respuesta.Code)
	}
}

func TestTeselasOSMRechazaContadorCentralZIPFalso(t *testing.T) {
	directorio := filepath.Join(t.TempDir(), "web", "cartografia")
	if err := os.MkdirAll(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	rutaZIP := filepath.Join(directorio, nombreZIPTeselasOSM)
	archivo, err := os.Create(rutaZIP)
	if err != nil {
		t.Fatal(err)
	}
	escritor := zip.NewWriter(archivo)
	for _, nombre := range []string{"tiles/8/125/99.png", "extra/fuente.txt"} {
		entrada, err := escritor.Create(nombre)
		if err != nil {
			t.Fatal(err)
		}
		if nombre == "tiles/8/125/99.png" {
			if _, err := entrada.Write(pngTeselaOSMPrueba(t)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	info, err := archivo.Stat()
	if err != nil {
		t.Fatal(err)
	}
	// El EOCD sin comentario termina el archivo; falseamos sus dos contadores
	// sin alterar el directorio central para verificar su recuento físico.
	var contador [4]byte
	binary.LittleEndian.PutUint16(contador[:2], 1)
	binary.LittleEndian.PutUint16(contador[2:], 1)
	if _, err := archivo.WriteAt(contador[:], info.Size()-22+8); err != nil {
		t.Fatal(err)
	}
	if err := archivo.Close(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
	)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("EOCD con contador falso = %d; esperado 404", respuesta.Code)
	}
}

func TestTeselasOSMConsultasConcurrentesAcotadas(t *testing.T) {
	rutaZIP := crearZIPTeselasOSMPrueba(t, pngTeselaOSMPrueba(t), "tiles/8/125/99.png")
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	handler := NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler())
	const solicitudes = 32
	inicio := make(chan struct{})
	estados := make(chan int, solicitudes)
	var grupo sync.WaitGroup
	for i := 0; i < solicitudes; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil))
			estados <- respuesta.Code
		}()
	}
	close(inicio)
	grupo.Wait()
	close(estados)
	exitos := 0
	for estado := range estados {
		if estado == http.StatusOK {
			exitos++
			continue
		}
		if estado != http.StatusServiceUnavailable {
			t.Fatalf("consulta concurrente = %d; esperado 200 o límite 503", estado)
		}
	}
	if exitos == 0 {
		t.Fatal("ninguna consulta concurrente pudo cargar el mapa")
	}
}
