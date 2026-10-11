package analisiscontenido

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestClienteRechazaRespuestaDeOtroObjeto(t *testing.T) {
	contenido, err := os.ReadFile(filepath.Join("testdata", "pasivo.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	objeto := ports.ReferenciaObjetoAlmacen{Referencia: "objeto:sintetico:uno", Version: "v1"}
	contexto, err := pruebas.NuevoContextoAlmacen(time.Now().UTC(), "analisis_cliente", ports.AccionAlmacenAnalizarContenido, objeto)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := ports.SolicitudAnalizarContenido{
		Contexto: contexto, Objeto: objeto, ConectorAlmacenID: "almacen_test",
		Zona: ports.ZonaAlmacenCuarentena, MIME: "application/pdf", Tamano: int64(len(contenido)),
		HuellaSHA256: huella(contenido), Contenido: bytes.NewReader(contenido),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatal(err)
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Now().UTC()
	respuestaAjena := ports.ResultadoAnalisisContenido{
		Objeto:            ports.ReferenciaObjetoAlmacen{Referencia: "objeto:sintetico:otro", Version: "v1"},
		ConectorAlmacenID: solicitud.ConectorAlmacenID, HuellaObjetoSHA256: solicitud.HuellaSHA256,
		TamanoObjeto: solicitud.Tamano, MIMEDeclarado: solicitud.MIME, MIMEDetectado: solicitud.MIME,
		ConectorAnalizadorID: "analizador_test", VersionConector: 1, MotorRef: "clamav",
		VersionMotor: "1.4.2", FirmasRef: "clamdb:27604", Estado: ports.EstadoAnalisisContenidoLimpio,
		CodigoResultado: "PDF_PASIVO", BytesAnalizados: solicitud.Tamano,
		EvidenciaRef:       "analisis:sintetico",
		AnalisisIniciadoEn: inicio, AnalisisCompletadoEn: inicio.Add(time.Second),
		CorrelacionRef: proyeccion.CorrelacionRef, AutorizacionRef: proyeccion.AutorizacionRef,
		Finalidad: proyeccion.Finalidad, Clasificacion: proyeccion.Clasificacion,
	}
	brutoEvidencia, err := json.Marshal(respuestaAjena)
	if err != nil {
		t.Fatal(err)
	}
	respuestaAjena.HuellaEvidenciaSHA256 = huella(brutoEvidencia)
	if err := respuestaAjena.Validar(); err != nil {
		t.Fatalf("la respuesta adulterada debe ser individualmente válida: %v", err)
	}
	if !evidenciaIntegra(respuestaAjena) {
		t.Fatal("la evidencia sintética debe tener una huella coherente")
	}
	if !errors.Is(respuestaAjena.ValidarContra(solicitud), ports.ErrResultadoAnalisisContenidoInvalido) {
		t.Fatal("solo el vínculo al objeto solicitado debe ser inválido")
	}
	caPEM, claveCA, certCA := crearCertificadoCA(t)
	rutaCA := escribirPEMTemporal(t, "ca.pem", caPEM)
	rutaServidor, rutaClaveServidor, _ := crearCertificado(t, certCA, claveCA, true, "")
	rutaCliente, rutaClaveCliente, _ := crearCertificado(t, certCA, claveCA, false, "spiffe://vec.test/cliente")
	parServidor, err := tls.LoadX509KeyPair(rutaServidor, rutaClaveServidor)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	var peticiones atomic.Int32
	servidor := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peticiones.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/analisis" || len(r.TLS.VerifiedChains) == 0 {
			t.Errorf("petición inesperada: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respuestaAjena)
	}))
	servidor.TLS = &tls.Config{
		MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{parServidor},
		ClientCAs: pool, ClientAuth: tls.RequireAndVerifyClientCert,
	}
	servidor.StartTLS()
	defer servidor.Close()
	cliente, err := NuevoCliente(ConfiguracionCliente{
		Endpoint: servidor.URL, NombreServidor: "localhost", RutaCA: rutaCA,
		RutaCertificado: rutaCliente, RutaClave: rutaClaveCliente, ConectorID: "analizador_test",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = cliente.Analizar(context.Background(), solicitud)
	if !errors.Is(err, ports.ErrResultadoAnalisisContenidoInvalido) || peticiones.Load() != 1 {
		t.Fatalf("respuesta ajena aceptada: peticiones=%d error=%v", peticiones.Load(), err)
	}
}
