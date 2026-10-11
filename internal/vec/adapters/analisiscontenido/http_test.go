package analisiscontenido

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestHandlerAnalisisPDFPorMTLS(t *testing.T) {
	const propia = "spiffe://vec.test/analizador-cliente"
	const ajena = "spiffe://vec.test/otro-cliente"
	caPEM, claveCA, certCA := crearCertificadoCA(t)
	rutaCA := escribirPEMTemporal(t, "ca.pem", caPEM)
	rutaServidor, claveServidor, _ := crearCertificado(t, certCA, claveCA, true, "")
	for _, caso := range []struct {
		nombre, archivo, identidad, codigo string
		estado                             ports.EstadoAnalisisContenido
		http                               int
		alterarHuella                      bool
	}{
		{"pasivo", "pasivo.pdf", propia, "PDF_PASIVO", ports.EstadoAnalisisContenidoLimpio, http.StatusOK, false},
		{"truncado", "truncado.pdf", propia, "PDF_DANADO", ports.EstadoAnalisisContenidoSospechoso, http.StatusOK, false},
		{"byte_extra", "byte_extra.pdf", propia, "PDF_DANADO", ports.EstadoAnalisisContenidoSospechoso, http.StatusOK, false},
		{"huella_ajena", "pasivo.pdf", propia, "", "", http.StatusUnprocessableEntity, true},
		{"identidad_ajena", "pasivo.pdf", ajena, "", "", http.StatusForbidden, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			contenido, err := os.ReadFile(filepath.Join("testdata", caso.archivo))
			if err != nil {
				t.Fatal(err)
			}
			version := versionFalsa(time.Now().UTC(), "27604")
			socket, cerrar := clamdFalso(t, contenido, "stream: OK", []string{version, version})
			defer cerrar()
			servicio, err := NuevoServicio(ConfiguracionServicio{
				ConectorID: "analizador_test", ClamdSocket: socket,
				Bwrap: buscarHerramienta(t, "bwrap"), QPDF: buscarHerramienta(t, "qpdf"), Prlimit: buscarHerramienta(t, "prlimit"),
				IdentidadesCliente: []string{propia}, MaxEdadFirmas: time.Hour,
			})
			if err != nil {
				t.Fatal(err)
			}
			configTLS, err := servicio.TLSConfig(rutaServidor, claveServidor, rutaCA)
			if err != nil {
				t.Fatal(err)
			}
			servidor := httptest.NewUnstartedServer(servicio.Handler())
			servidor.TLS = configTLS
			servidor.StartTLS()
			defer servidor.Close()
			cliente := clienteMTLSPrueba(t, caPEM, certCA, claveCA, caso.identidad)
			defer cliente.CloseIdleConnections()
			meta := solicitudRemota{
				Objeto:            ports.ReferenciaObjetoAlmacen{Referencia: "objeto:sintetico:uno", Version: "v1"},
				ConectorAlmacenID: "almacen_test", Zona: ports.ZonaAlmacenCuarentena,
				MIME: "application/pdf", Tamano: int64(len(contenido)),
				HuellaSHA256: huella(contenido), CorrelacionRef: "correlacion:test", AutorizacionRef: "autorizacion:test",
				Finalidad: "prueba", Clasificacion: "sintetica",
			}
			if caso.alterarHuella {
				meta.HuellaSHA256 = huella([]byte("otros bytes"))
			}
			var cuerpo bytes.Buffer
			multi := multipart.NewWriter(&cuerpo)
			metadatos, err := json.Marshal(meta)
			if err != nil {
				t.Fatal(err)
			}
			if err := multi.WriteField("metadatos", string(metadatos)); err != nil {
				t.Fatal(err)
			}
			parte, err := multi.CreateFormFile("archivo", "sintetico.pdf")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parte.Write(contenido); err != nil {
				t.Fatal(err)
			}
			if err := multi.Close(); err != nil {
				t.Fatal(err)
			}
			peticion, err := http.NewRequestWithContext(context.Background(), http.MethodPost, servidor.URL+"/v1/analisis", &cuerpo)
			if err != nil {
				t.Fatal(err)
			}
			peticion.Header.Set("Content-Type", multi.FormDataContentType())
			respuesta, err := cliente.Do(peticion)
			if err != nil {
				t.Fatal(err)
			}
			defer respuesta.Body.Close()
			if respuesta.StatusCode != caso.http {
				mensaje, _ := io.ReadAll(io.LimitReader(respuesta.Body, 512))
				t.Fatalf("HTTP %d, esperado %d: %q", respuesta.StatusCode, caso.http, mensaje)
			}
			if caso.http != http.StatusOK {
				return
			}
			var resultado ports.ResultadoAnalisisContenido
			if err := json.NewDecoder(respuesta.Body).Decode(&resultado); err != nil {
				t.Fatal(err)
			}
			if err := resultado.Validar(); err != nil {
				t.Fatalf("resultado inválido: %v", err)
			}
			if resultado.Estado != caso.estado || resultado.CodigoResultado != caso.codigo ||
				resultado.HuellaObjetoSHA256 != huella(contenido) || resultado.TamanoObjeto != int64(len(contenido)) ||
				resultado.BytesAnalizados != int64(len(contenido)) || resultado.Objeto != meta.Objeto ||
				resultado.FirmasRef != "clamdb:27604" {
				t.Fatalf("resultado no vinculado al documento: %+v", resultado)
			}
		})
	}
}

func TestCapacidadesExigeFirmasVigentesYParserOperativo(t *testing.T) {
	const propia = "spiffe://vec.test/analizador-cliente"
	caPEM, claveCA, certCA := crearCertificadoCA(t)
	rutaCA := escribirPEMTemporal(t, "ca.pem", caPEM)
	rutaServidor, claveServidor, _ := crearCertificado(t, certCA, claveCA, true, "")
	for _, caso := range []struct {
		nombre    string
		fecha     time.Time
		identidad string
		estado    int
	}{
		{"vigente", time.Now().UTC(), propia, http.StatusOK},
		{"caducada", time.Now().UTC().Add(-25 * time.Hour), propia, http.StatusServiceUnavailable},
		{"ajena", time.Now().UTC(), "spiffe://vec.test/otro-cliente", http.StatusForbidden},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			socket, cerrar := clamdFalso(t, []byte("sin lectura"), "stream: OK", []string{versionFalsa(caso.fecha, "27604")})
			defer cerrar()
			servicio, err := NuevoServicio(ConfiguracionServicio{
				ConectorID: "analizador_test", ClamdSocket: socket,
				Bwrap: buscarHerramienta(t, "bwrap"), QPDF: buscarHerramienta(t, "qpdf"), Prlimit: buscarHerramienta(t, "prlimit"),
				IdentidadesCliente: []string{propia}, MaxEdadFirmas: 24 * time.Hour,
			})
			if err != nil {
				t.Fatal(err)
			}
			configTLS, err := servicio.TLSConfig(rutaServidor, claveServidor, rutaCA)
			if err != nil {
				t.Fatal(err)
			}
			servidor := httptest.NewUnstartedServer(servicio.Handler())
			servidor.TLS = configTLS
			servidor.StartTLS()
			defer servidor.Close()
			cliente := clienteMTLSPrueba(t, caPEM, certCA, claveCA, caso.identidad)
			defer cliente.CloseIdleConnections()
			respuesta, err := cliente.Get(servidor.URL + "/v1/capacidades")
			if err != nil {
				t.Fatal(err)
			}
			defer respuesta.Body.Close()
			if respuesta.StatusCode != caso.estado {
				mensaje, _ := io.ReadAll(io.LimitReader(respuesta.Body, 512))
				t.Fatalf("HTTP %d, esperado %d: %q", respuesta.StatusCode, caso.estado, mensaje)
			}
			if caso.estado == http.StatusOK {
				var capacidades ports.CapacidadesAnalizadorContenido
				if err := json.NewDecoder(respuesta.Body).Decode(&capacidades); err != nil {
					t.Fatal(err)
				}
				if capacidades.ConectorID != "analizador_test" || !capacidades.DetectaMalware || !capacidades.DetectaContenidoActivo ||
					!capacidades.IdentidadMutua || capacidades.TamanoMaximo != tamanoMaximoPDF {
					t.Fatalf("capacidades incompletas: %+v", capacidades)
				}
			}
		})
	}
}

func buscarHerramienta(t *testing.T, nombre string) string {
	t.Helper()
	ruta, err := exec.LookPath(nombre)
	if err != nil {
		t.Skipf("%s no está instalado", nombre)
	}
	return ruta
}

func clienteMTLSPrueba(t *testing.T, caPEM []byte, ca *x509.Certificate, claveCA *ecdsa.PrivateKey, identidad string) *http.Client {
	t.Helper()
	rutaCert, rutaClave, _ := crearCertificado(t, ca, claveCA, false, identidad)
	par, err := tls.LoadX509KeyPair(rutaCert, rutaClave)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("CA sintética inválida")
	}
	return &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: pool, Certificates: []tls.Certificate{par}, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
}
