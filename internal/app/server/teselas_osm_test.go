package server

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

func hashZIPTeselasOSMPrueba(t *testing.T, ruta string) string {
	t.Helper()
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func handlerInternoTeselasOSMPrueba(t *testing.T, ruta string) http.Handler {
	t.Helper()
	return newHandlerInternoConHashTeselasOSM(config.Config{}, http.NotFoundHandler(), nil, hashZIPTeselasOSMPrueba(t, ruta))
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
		{"interna", handlerInternoTeselasOSMPrueba(t, rutaZIP), http.StatusOK},
		{"integrada", newHandlerIntegradoConHashTeselasOSM(config.Config{}, http.NotFoundHandler(), nil, hashZIPTeselasOSMPrueba(t, rutaZIP)), http.StatusOK},
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
	handler := handlerInternoTeselasOSMPrueba(t, rutaZIP)
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
	handler := handlerInternoTeselasOSMPrueba(t, rutaZIP)
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
	handler = newHandlerInternoConHashTeselasOSM(config.Config{}, http.NotFoundHandler(), nil, hashZIPTeselasOSMPrueba(t, otraRuta))
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
	handlerInternoTeselasOSMPrueba(t, rutaZIP).ServeHTTP(
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
			handlerInternoTeselasOSMPrueba(t, rutaZIP).ServeHTTP(
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
	handlerInternoTeselasOSMPrueba(t, rutaZIP).ServeHTTP(
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

func TestTeselasOSMRechazaEOCDInteriorAntesDeAnalizarZIP(t *testing.T) {
	var base bytes.Buffer
	escritor := zip.NewWriter(&base)
	for i := 0; i < maxEntradasZIPOSM+1; i++ {
		nombre := "extra/" + strconv.Itoa(i)
		if i == 0 {
			nombre = "tiles/8/125/99.png"
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
	datosBase := base.Bytes()
	finalBase := datosBase[len(datosBase)-22:]
	if binary.LittleEndian.Uint32(finalBase) != 0x06054b50 {
		t.Fatal("fixture ZIP sin EOCD normal")
	}
	inicioCentral := int(binary.LittleEndian.Uint32(finalBase[16:]))
	primeraCentral := datosBase[inicioCentral:]
	if binary.LittleEndian.Uint32(primeraCentral) != 0x02014b50 {
		t.Fatal("fixture ZIP sin cabecera central")
	}
	longitudPrimera := 46 +
		int(binary.LittleEndian.Uint16(primeraCentral[28:])) +
		int(binary.LittleEndian.Uint16(primeraCentral[30:])) +
		int(binary.LittleEndian.Uint16(primeraCentral[32:]))
	datos := append([]byte(nil), datosBase...)
	datos = append(datos, primeraCentral[:longitudPrimera]...)
	var finalExterior [22]byte
	binary.LittleEndian.PutUint32(finalExterior[:], 0x06054b50)
	binary.LittleEndian.PutUint16(finalExterior[8:], 1)
	binary.LittleEndian.PutUint16(finalExterior[10:], 1)
	binary.LittleEndian.PutUint32(finalExterior[12:], uint32(longitudPrimera))
	binary.LittleEndian.PutUint32(finalExterior[16:], uint32(len(datosBase)))
	binary.LittleEndian.PutUint16(finalExterior[20:], 23)
	datos = append(datos, finalExterior[:]...)
	datos = append(datos, finalBase...)
	datos = append(datos, 0) // El EOCD interior no termina en EOF.
	// Go elige el final interior con 4097 entradas; el final exterior parece
	// tener una sola y fue la evasión reproducida por ambas revisiones R2.
	lector, err := zip.NewReader(bytes.NewReader(datos), int64(len(datos)))
	if err != nil || len(lector.File) != maxEntradasZIPOSM+1 {
		t.Fatalf("fixture adversarial no reproduce la elección de Go: %v, entradas=%d", err, len(lector.File))
	}
	directorio := filepath.Join(t.TempDir(), "web", "cartografia")
	if err := os.MkdirAll(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	rutaZIP := filepath.Join(directorio, nombreZIPTeselasOSM)
	if err := os.WriteFile(rutaZIP, datos, 0600); err != nil {
		t.Fatal(err)
	}
	archivo, err := os.Open(rutaZIP)
	if err != nil {
		t.Fatal(err)
	}
	defer archivo.Close()
	if contenido, err := leerZIPTeselasOSMValidado(archivo, int64(len(datos)), sha256ZIPTeselasOSM); err == nil || contenido != nil {
		t.Fatal("el ZIP manipulado alcanzó el parser pese a no tener el SHA-256 fijado")
	}
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/8/125/99.png", nil),
	)
	if respuesta.Code != http.StatusNotFound {
		t.Fatalf("ZIP con EOCD ambiguo = %d; esperado 404", respuesta.Code)
	}
}

func TestTeselasOSMConsultasConcurrentesAcotadas(t *testing.T) {
	rutaZIP := crearZIPTeselasOSMPrueba(t, pngTeselaOSMPrueba(t), "tiles/8/125/99.png")
	t.Chdir(filepath.Dir(filepath.Dir(filepath.Dir(rutaZIP))))
	handler := handlerInternoTeselasOSMPrueba(t, rutaZIP)
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

func TestTeselasOSMZIPHistoricoSiEstaInstalado(t *testing.T) {
	proyecto := os.Getenv("VEC_TEST_OSM_PROJECT")
	if proyecto == "" {
		proyecto = "../../.."
	}
	rutaZIP := filepath.Join(proyecto, "web", "cartografia", nombreZIPTeselasOSM)
	if _, err := os.Stat(rutaZIP); os.IsNotExist(err) {
		t.Skip("el ZIP histórico se empaqueta en otra rama")
	} else if err != nil {
		t.Fatal(err)
	}
	if got := hashZIPTeselasOSMPrueba(t, rutaZIP); got != sha256ZIPTeselasOSM {
		t.Fatalf("SHA-256 histórico = %s; no coincide con el fijado", got)
	}
	t.Chdir(proyecto)
	respuesta := httptest.NewRecorder()
	NewHandlerInternoWithConfig(config.Config{}, http.NotFoundHandler()).ServeHTTP(
		respuesta, peticionServidorPrueba(http.MethodGet, "/tiles/osm/10/498/394.png", nil),
	)
	if respuesta.Code != http.StatusOK || respuesta.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("tesela histórica = %d, tipo %q; esperado 200 image/png", respuesta.Code, respuesta.Header().Get("Content-Type"))
	}
}
