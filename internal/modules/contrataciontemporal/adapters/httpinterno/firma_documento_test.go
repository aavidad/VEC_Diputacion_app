package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type autoridadFirmaPrueba struct{ err error }

func (a autoridadFirmaPrueba) ResolverOrganizacionFirmaDocumento(context.Context) (string, error) {
	return "organizacion:desarrollo:dipgra", a.err
}

type servicioFirmaPrueba struct {
	err       error
	recibida  application.SolicitudFirmaDocumento
	verifica  bool
	resultado application.ResultadoFirmaDocumento
}

func (s *servicioFirmaPrueba) Firmar(_ context.Context, sol application.SolicitudFirmaDocumento) (application.ResultadoFirmaDocumento, error) {
	s.recibida = sol
	s.recibida.Original = append([]byte(nil), sol.Original...)
	s.recibida.Firmado = append([]byte(nil), sol.Firmado...)
	return s.resultado, s.err
}

func (s *servicioFirmaPrueba) Consultar(context.Context, string, string) (application.EstadoFirmasExpediente, error) {
	c := domain.CircuitoFirmaDocumento{Documento: "informe_definitivo", Etiqueta: "Informe definitivo", Pasos: []domain.PasoCircuitoFirma{
		{Orden: 1, Cargo: "Técnico", Accion: "firma", Devolucion: domain.DevolucionVuelveARedaccion, Referencia: "c:1:p1"}}}
	return application.EstadoFirmasExpediente{
		Circuito:   domain.CircuitoFirma{CatalogoRef: "circuito:1", HuellaCatalogo: strings.Repeat("c", 64), Ejemplo: true, Documentos: []domain.CircuitoFirmaDocumento{c}},
		Documentos: []domain.EstadoCircuitoDocumento{{Documento: "informe_definitivo", PasoPendiente: 1, Pasos: []domain.EstadoPasoCalculado{{Orden: 1, Estado: domain.EstadoPasoPendienteFirma}}}},
	}, s.err
}

func (s *servicioFirmaPrueba) VerificacionDisponible() bool { return s.verifica }

func peticionFirma(ruta, cuerpo string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(cuerpo))
	r.Header.Set("Content-Type", "application/json")
	return r
}

const cuerpoFirmaPrueba = `{"expediente_ref":"expediente:ct:001","version_expediente":7,"documento":"informe_definitivo","paso_orden":1,"resultado":"firmado","motivo_devolucion":"","original_base64":"JVBERi0x","firmado_base64":"JVBERi0xIGZpcm1h","clave_idempotencia":"clave-firma-000000001"}`

func TestManejadorFirmaDocumentoRegistraSinEficacia(t *testing.T) {
	s := &servicioFirmaPrueba{resultado: application.ResultadoFirmaDocumento{
		Recibo: ports.ReciboFirmaDocumento{FirmaRef: "firma-ct:1", ReciboRef: "recibo-firma-ct:1", Secuencia: 1, Resultado: domain.ResultadoFirmaFirmado,
			ExpedienteVersion: 7, PerfilRef: "perfil:x", RegistradaEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)},
		Material:           ports.MaterialFirmaDocumento{ExpedienteRef: "expediente:ct:001", Documento: "informe_definitivo", PasoOrden: 1, PoliticaVerificacion: ports.PoliticaVerificacionFirma},
		MotivoVerificacion: docports.MotivoFirmaVerificada,
	}}
	h, err := NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{}, s)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionFirma(RutaFirmaDocumento, cuerpoFirmaPrueba))
	var salida struct {
		Data map[string]any `json:"data"`
	}
	if w.Code != http.StatusCreated || json.Unmarshal(w.Body.Bytes(), &salida) != nil || salida.Data["firma_eficaz"] != false ||
		salida.Data["firma_verificada"] != true || s.recibida.OrganizacionRef != "organizacion:desarrollo:dipgra" ||
		string(s.recibida.Original) != "%PDF-1" || string(s.recibida.Firmado) != "%PDF-1 firma" {
		t.Fatalf("registro: %d %s %+v", w.Code, w.Body, s.recibida)
	}
}

func TestManejadorFirmaDocumentoErrores(t *testing.T) {
	casos := []struct {
		err    error
		estado int
		codigo string
	}{
		{application.ErrVerificacionFirmaApagada, http.StatusServiceUnavailable, "verificacion_no_disponible"},
		{application.DictamenRechazado{Estado: docports.EstadoVerificacionNoValida, Motivo: docports.MotivoIntegridadNoValida}, http.StatusUnprocessableEntity, "firma_no_verificada"},
		{application.ErrPasoFirmaNoPendiente, http.StatusConflict, "paso_no_pendiente"},
		{ports.ErrCadenaFirmaDocumentoRota, http.StatusConflict, "cadena_rota"},
		{ports.ErrFirmaDocumentoDenegada, http.StatusForbidden, "acceso_denegado"},
		{errors.New("otro"), http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, c := range casos {
		h, _ := NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{}, &servicioFirmaPrueba{err: c.err})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticionFirma(RutaFirmaDocumento, cuerpoFirmaPrueba))
		if w.Code != c.estado || !strings.Contains(w.Body.String(), `"codigo":"`+c.codigo+`"`) {
			t.Fatalf("%v: %d %s", c.err, w.Code, w.Body)
		}
	}
	h, _ := NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{}, &servicioFirmaPrueba{err: application.DictamenRechazado{Motivo: docports.MotivoRevocacionNoAcreditada}})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionFirma(RutaFirmaDocumento, cuerpoFirmaPrueba))
	if !strings.Contains(w.Body.String(), `"motivo":"revocacion_no_acreditada"`) {
		t.Fatalf("motivo del dictamen ausente: %s", w.Body)
	}
}

func TestManejadorFirmaDocumentoRechazaEntradas(t *testing.T) {
	s := &servicioFirmaPrueba{}
	h, _ := NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{}, s)
	casos := map[string]*http.Request{
		"campo ajeno":      peticionFirma(RutaFirmaDocumento, strings.Replace(cuerpoFirmaPrueba, `"documento"`, `"actor_ref":"x","documento"`, 1)),
		"base64 inválido":  peticionFirma(RutaFirmaDocumento, strings.Replace(cuerpoFirmaPrueba, "JVBERi0x\"", "JVBERi0x!\"", 1)),
		"consulta ajena":   peticionFirma(RutaConsultaFirmaDocumento, `{"expediente_ref":"expediente:ct:001","otro":1}`),
		"con cadena query": peticionFirma(RutaFirmaDocumento+"?x=1", cuerpoFirmaPrueba),
	}
	for nombre, r := range casos {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code < 400 {
			t.Fatalf("%s aceptado: %d", nombre, w.Code)
		}
	}
	r := peticionFirma(RutaFirmaDocumento, cuerpoFirmaPrueba)
	r.Header.Set("Cookie", "a=b")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("cookie admitida: %d", w.Code)
	}
	h, _ = NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{err: errors.New("sin canal")}, s)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionFirma(RutaFirmaDocumento, cuerpoFirmaPrueba))
	if w.Code != http.StatusForbidden {
		t.Fatalf("sin canal: %d", w.Code)
	}
}

func TestManejadorConsultaFirmasDocumento(t *testing.T) {
	h, _ := NuevoManejadorFirmaDocumento(autoridadFirmaPrueba{}, &servicioFirmaPrueba{verifica: false})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionFirma(RutaConsultaFirmaDocumento, `{"expediente_ref":"expediente:ct:001"}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"firma_eficaz":false`) ||
		!strings.Contains(w.Body.String(), `"verificacion_disponible":false`) || !strings.Contains(w.Body.String(), `"estado":"pendiente_firma"`) {
		t.Fatalf("consulta: %d %s", w.Code, w.Body)
	}
}
