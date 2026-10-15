package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestConfiguracionSubidaLocalFallaSoloSuFuncion(t *testing.T) {
	raiz := t.TempDir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó subida sin material privado")
	}
	material := filepath.Join(raiz, "externo", "documentos")
	if err := os.MkdirAll(material, 0o700); err != nil {
		t.Fatal(err)
	}
	directorio := filepath.Join(raiz, "custodia")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	retencionDeclarada := int64(0)
	c := configuracionSubidaDocumentosExterna{
		Version: 1, DSNDatos: "doc-datos", DSNFrontera: "doc-frontera",
		Directorio: directorio, RetencionMinimaDias: &retencionDeclarada,
		ClaveCorrelacionArchivo: "correlacion.key",
		ClaveSeudonimoArchivo:   "seudonimo.key", ClaveSolicitudArchivo: "solicitud.key",
		Analizador: configuracionAnalizadorSubidaExterna{Endpoint: "https://analizador.invalid",
			NombreServidor: "analizador.invalid", RutaCA: "ca.pem", RutaCertificado: "cliente.pem",
			RutaClave: "cliente.key", ConectorID: "analizador_pdf"},
	}
	ruta := filepath.Join(material, "subidas.json")
	escribir := func() {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err != nil {
		t.Fatalf("rechazó material local válido: %v", err)
	}
	c.RetencionMinimaDias = nil
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó retención no declarada")
	}
	c.RetencionMinimaDias = &retencionDeclarada
	escribir()
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó material legible por terceros")
	}
	if err := os.Chmod(ruta, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directorio, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó custodia accesible por terceros")
	}
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	c.ClaveSolicitudArchivo = "../otra.key"
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó clave fuera del directorio privado")
	}
}

type subidorLocalPuertoPrueba struct {
	subida    docports.SolicitudSubidaLocal
	consulta  docports.SolicitudConsultaLocal
	resultado docports.ResultadoSubidaLocal
	err       error
}

func (p *subidorLocalPuertoPrueba) Subir(_ context.Context, q docports.SolicitudSubidaLocal, _ []byte) (docports.ResultadoSubidaLocal, error) {
	p.subida = q
	return p.resultado, p.err
}
func (p *subidorLocalPuertoPrueba) Consultar(_ context.Context, q docports.SolicitudConsultaLocal) (docports.ResultadoSubidaLocal, error) {
	p.consulta = q
	return p.resultado, p.err
}
func (p *subidorLocalPuertoPrueba) AbrirOriginal(_ context.Context, q docports.SolicitudConsultaLocal) (io.ReadCloser, error) {
	p.consulta = q
	return io.NopCloser(strings.NewReader("%PDF-1.4\ncontenido")), p.err
}

func TestPuenteHTTPSubidaLocalNoFabricaConfirmacion(t *testing.T) {
	fecha := time.Date(2026, 10, 14, 20, 0, 0, 0, time.UTC)
	puerto := &subidorLocalPuertoPrueba{resultado: docports.ResultadoSubidaLocal{
		CargaID: "ref:carga", ComprobanteRef: "ref:documento", RegistradoEn: fecha,
		Estado: docports.EstadoSubidaLocalAdmitida}}
	cargador := cargadorComprobanteDocumentosExterno{servicio: puerto}
	q := meritoshttp.IdentidadComprobanteCartera{CorrelacionRef: "correlacion_12345678901234567890123456789012"}
	const idOperacion = "declaracion:uno"
	admitido, err := cargador.SubirComprobante(context.Background(), q, idOperacion, []byte("%PDF-1.4\ncontenido"))
	if err != nil || admitido.Estado != "admitido" || admitido.ComprobanteRef != "ref:documento" ||
		!admitido.RegistradoEn.Equal(fecha) || puerto.subida.ClaveIdempotencia != idOperacion {
		t.Fatalf("recibo documental alterado: %+v %v", admitido, err)
	}
	puerto.resultado = docports.ResultadoSubidaLocal{CargaID: "ref:carga", Estado: docports.EstadoSubidaLocalAnalizando}
	puerto.err = docports.ErrSubidaLocalPendiente
	pendiente, err := cargador.SubirComprobante(context.Background(), q, idOperacion, []byte("%PDF-1.4\ncontenido"))
	if !errors.Is(err, docports.ErrSubidaLocalPendiente) || pendiente.ComprobanteRef != "" || pendiente.Estado != "analizando" {
		t.Fatalf("pendiente publicado como admitido: %+v %v", pendiente, err)
	}
	puerto.err = docports.ErrSubidaLocalNoEncontrada
	if _, err := cargador.ConsultarEstadoComprobante(context.Background(), q, idOperacion); !errors.Is(err, meritoshttp.ErrEstadoComprobanteNoEncontrado) {
		t.Fatalf("no distinguió clave nunca reservada: %v", err)
	}
}

func TestClavesHMACSubidaLocalExigenTreintaYDosBytesPrivados(t *testing.T) {
	raiz := t.TempDir()
	material := filepath.Join(raiz, "externo", "documentos")
	if err := os.MkdirAll(material, 0o700); err != nil {
		t.Fatal(err)
	}
	c := configuracionSubidaDocumentosExterna{ClaveCorrelacionArchivo: "correlacion.key",
		ClaveSeudonimoArchivo: "seudonimo.key", ClaveSolicitudArchivo: "solicitud.key"}
	ruta := filepath.Join(material, "solicitud.key")
	if _, err := leerClavesSubidaDocumentosExterna(raiz, c); err == nil {
		t.Fatal("aceptó clave ausente")
	}
	for nombre, valor := range map[string]string{
		"correlacion.key": "12345678901234567890123456789012",
		"seudonimo.key":   "22345678901234567890123456789012",
		"solicitud.key":   "32345678901234567890123456789012",
	} {
		if err := os.WriteFile(filepath.Join(material, nombre), []byte(valor), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	claves, err := leerClavesSubidaDocumentosExterna(raiz, c)
	if err != nil || claves == (clavesSubidaDocumentosExterna{}) {
		t.Fatalf("rechazó clave privada válida: %v", err)
	}
	correlaciones, seudonimos, sellador, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil || correlaciones == nil || seudonimos == nil || sellador == nil {
		t.Fatalf("no compuso tres finalidades separadas: %v", err)
	}
	if claves != (clavesSubidaDocumentosExterna{}) {
		t.Fatal("no borró copia temporal")
	}
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerClavesSubidaDocumentosExterna(raiz, c); err == nil {
		t.Fatal("aceptó clave legible por terceros")
	}
}

func TestSubidaDocumentalAusenteCierraSoloPDF(t *testing.T) {
	var documental *componenteSubidaDocumentosExterna
	siguiente := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := documental.proteger(siguiente)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, meritoshttp.RutaComprobantesCartera, nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("subida sin configuración: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet,
		meritoshttp.RutaComprobantesCartera+"/ref%3Adocumento/original", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("descarga sin configuración: %d", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, meritoshttp.RutaCartera, nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("cerró la cartera sin PDF: %d", w.Code)
	}
}
