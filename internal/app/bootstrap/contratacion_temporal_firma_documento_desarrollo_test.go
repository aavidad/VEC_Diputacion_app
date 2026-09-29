package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func materialFirmaDesarrolloPrueba() ports.MaterialFirmaDocumento {
	return ports.MaterialFirmaDocumento{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, ExpedienteRef: "expediente:ct:001",
		VersionExpediente: 7, Documento: "informe_definitivo", CatalogoRef: "vec.contratacion_temporal.circuito_firma:1",
		CatalogoHuella: strings.Repeat("c", 64), PasoRef: "vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p1",
		PasoOrden: 1, Secuencia: 1, Resultado: ctdomain.ResultadoFirmaDevuelto, MotivoDevolucion: "Falta la fecha",
		ClaveIdempotencia: "clave-devolucion-00001"}
}

func TestPredicadoFirmaDocumentoLigadoAlMaterial(t *testing.T) {
	m := materialFirmaDesarrolloPrueba()
	recurso, err := ctapplication.RecursoFirmaDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: ports.AccionFirmarDocumento, Finalidad: ports.FinalidadFirmaDocumento,
		ReferenciaMotivo: motivoFirmaDocumentoCTDesarrollo(), Recurso: recurso}
	ctx := context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, m)
	if !solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, datos) {
		t.Fatal("la solicitud exacta no se admite")
	}
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.Background(), datos) {
		t.Fatal("sin material ligado se admite")
	}
	otro := m
	otro.MotivoDevolucion = "Otro motivo"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, otro), datos) {
		t.Fatal("un material distinto reutiliza la solicitud")
	}
	cambiada := datos
	cambiada.Accion = "contratacion_temporal.seguimiento.cerrar"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, cambiada) {
		t.Fatal("otra acción admitida")
	}
	ajena := m
	ajena.OrganizacionRef = "organizacion:ajena"
	if solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(context.WithValue(context.Background(), claveMaterialFirmaDocumentoCTDesarrollo{}, ajena), datos) {
		t.Fatal("organización ajena admitida")
	}
}

func TestFirmaDocumentoCTApagadaNoCompone(t *testing.T) {
	f, err := nuevaFirmaDocumentoCTDesarrollo(config.Config{}, nil, relojContratacionTemporalDesarrollo{}, nil)
	if f != nil || err != nil {
		t.Fatalf("selector apagado: %v %v", f, err)
	}
	var nula *firmaDocumentoCTDesarrollo
	if rutas, err := nula.rutas(config.Config{}, nil); rutas != nil || err != nil {
		t.Fatal("sin firma no hay rutas")
	}
	if _, err := nula.ResolverOrganizacionFirmaDocumento(context.Background()); err == nil {
		t.Fatal("canal sin composición admitido")
	}
	if _, err := nula.AutorizarFirmaDocumento(context.Background(), materialFirmaDesarrolloPrueba()); err == nil {
		t.Fatal("autorización sin composición")
	}
	if _, err := nuevaFirmaDocumentoCTDesarrollo(config.Config{CTFirmaRegistroEnabled: "si"}, nil, relojContratacionTemporalDesarrollo{}, nil); err == nil {
		t.Fatal("selector inválido admitido")
	}
}

func TestFirmaDocumentoCTRutasYConsumidor(t *testing.T) {
	if !rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaFirmaDocumento) || !rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaConsultaFirmaDocumento) ||
		rutaFirmaDocumentoCTDesarrollo(httpinterno.RutaFirmaDocumento+"/") {
		t.Fatal("rutas de firma")
	}
	if !rutaMutacionDurableContratacionTemporalDesarrollo(httpinterno.RutaFirmaDocumento) ||
		rutaMutacionDurableContratacionTemporalDesarrollo(httpinterno.RutaConsultaFirmaDocumento) ||
		!rutaContextoAutorizacionContratacionTemporalDesarrollo(httpinterno.RutaFirmaDocumento) {
		t.Fatal("la escritura debe ser mutación durable y la consulta no")
	}
	d := descriptorMaterialFirmaDocumentoCTDesarrollo()
	if d.Audiencia != ports.AudienciaFirmaDocumentoV3 || !slices.Contains(audienciasConsumoGobiernoCTDesarrollo(), d.Audiencia) {
		t.Fatal("la audiencia de firma no está en la lista única")
	}
	if dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivoFirmaDocumentoCTDesarrollo()) == false {
		t.Fatal("motivo de firma inválido")
	}
}

type servicioFirmaConsultaCerradaPrueba struct{ consultas int }

func (*servicioFirmaConsultaCerradaPrueba) Firmar(context.Context, ctapplication.SolicitudFirmaDocumento) (ctapplication.ResultadoFirmaDocumento, error) {
	return ctapplication.ResultadoFirmaDocumento{}, ports.ErrFirmaDocumentoDenegada
}

func (s *servicioFirmaConsultaCerradaPrueba) Consultar(context.Context, string, string) (ctapplication.EstadoFirmasExpediente, error) {
	s.consultas++
	return ctapplication.EstadoFirmasExpediente{}, nil
}

func (*servicioFirmaConsultaCerradaPrueba) VerificacionDisponible() bool { return false }

func TestFirmaDocumentoCTConsultaCerradaAntesDeLeerRegistro(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	firma := &firmaDocumentoCTDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{soporte: soporte}}
	registro := &servicioFirmaConsultaCerradaPrueba{}
	h, err := httpinterno.NuevoManejadorFirmaDocumento(firma, registro)
	if err != nil {
		t.Fatal(err)
	}
	peticion := httptest.NewRequest(http.MethodPost, httpinterno.RutaConsultaFirmaDocumento,
		strings.NewReader(`{"expediente_ref":"expediente:ct:001"}`))
	peticion.Header.Set("Content-Type", "application/json")
	peticion = peticion.WithContext(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaConsultaFirmaDocumento))
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden || !strings.Contains(respuesta.Body.String(), `"codigo":"acceso_denegado"`) || registro.consultas != 0 {
		t.Fatalf("consulta sin autoridad nominal: estado=%d cuerpo=%s lecturas=%d", respuesta.Code, respuesta.Body, registro.consultas)
	}
	organizacion, err := firma.ResolverOrganizacionFirmaDocumento(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaFirmaDocumento))
	if err != nil || organizacion != organizacionAltaContratacionTemporalDesarrollo {
		t.Fatalf("registro de firma nominal cerrado por error: organizacion=%q error=%v", organizacion, err)
	}
}
