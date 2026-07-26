package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type autoridadCoberturaPrueba struct {
	contexto ContextoCanalCobertura
	err      error
}

func (a autoridadCoberturaPrueba) ResolverContextoCanalCobertura(
	context.Context,
) (ContextoCanalCobertura, error) {
	return a.contexto, a.err
}

type servicioCoberturaPrueba struct {
	propuesta  application.PresentacionPropuestaCobertura
	recibo     cobertura.ReciboOperacionDecisionCobertura
	err        error
	decidir    application.SolicitudDecidirCobertura
	rectificar application.SolicitudRectificarCobertura
}

func (s *servicioCoberturaPrueba) Proponer(
	context.Context,
	application.SolicitudProponerCobertura,
) (application.PresentacionPropuestaCobertura, error) {
	return s.propuesta, s.err
}

func (s *servicioCoberturaPrueba) Decidir(
	_ context.Context,
	solicitud application.SolicitudDecidirCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	s.decidir = solicitud
	return s.recibo, s.err
}

func (s *servicioCoberturaPrueba) Rectificar(
	_ context.Context,
	solicitud application.SolicitudRectificarCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	s.rectificar = solicitud
	return s.recibo, s.err
}

func cuerpoDecisionCoberturaPrueba(rectificacion bool) string {
	identidad, _ := json.Marshal(identidadSemanticaCoberturaPrueba())
	cuerpo := `{"expediente_ref":"expediente:ct:0001","version_esperada":1,"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732","identidad_semantica":` + string(identidad) + `,"via_elegida":"bolsa_vigente","motivo_clave":""}`
	if rectificacion {
		cuerpo = `{"expediente_ref":"expediente:ct:0001","version_esperada":1,"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732","identidad_semantica":` + string(identidad) + `,"via_elegida":"bolsa_vigente","motivo_clave":"rectificacion","predecesora_ref":"decision-cobertura:sha256:` + strings.Repeat("c", 64) + `","predecesora_huella":"` + strings.Repeat("c", 64) + `"}`
	}
	return cuerpo
}

func contextoCoberturaValidoPrueba() ContextoCanalCobertura {
	return ContextoCanalCobertura{
		AutenticacionRef: "aut_0123456789abcdefghijkl",
		SesionRef:        "ses_0123456789abcdefghijkl",
		PerfilRef:        "prf_0123456789abcdefghijkl",
		OrganizacionRef:  "organizacion:rrhh:001",
	}
}

func identidadSemanticaCoberturaPrueba() domain.IdentidadSemanticaPropuestaDecisionCobertura {
	huella := strings.Repeat("a", 64)
	return domain.IdentidadSemanticaPropuestaDecisionCobertura{
		Referencia:   "propuesta-cobertura-semantica:sha256:" + huella,
		HuellaSHA256: huella,
		Canon:        domain.CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
	}
}

func nuevaPeticionCoberturaPrueba(ruta string, cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, ruta, bytes.NewBufferString(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func TestManejadorCoberturaProyectaPropuestaSinAutoridadDelCliente(t *testing.T) {
	servicio := &servicioCoberturaPrueba{propuesta: application.PresentacionPropuestaCobertura{
		Estado:         domain.PropuestaCoberturaViable,
		ViaRecomendada: "bolsa_vigente",
		Evaluaciones: []domain.EvaluacionViaPropuestaCobertura{{
			ViaClave: "bolsa_vigente", Prioridad: 1, Estado: domain.EvaluacionViaCoberturaViable,
		}},
		IdentidadSemantica: identidadSemanticaCoberturaPrueba(),
	}}
	manejador, err := NuevoManejadorCobertura(
		autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(RutaPropuestaCobertura,
		`{"expediente_ref":"expediente:ct:0001","version_esperada":1}`))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	for _, prohibido := range []string{"aut_", "ses_", "prf_", "organizacion", "cookie", "auditoria"} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), strings.ToLower(prohibido)) {
			t.Fatalf("salida contiene %q", prohibido)
		}
	}
	var envoltorio struct {
		Data struct {
			ViaRecomendada string          `json:"via_recomendada"`
			Identidad      json.RawMessage `json:"identidad_semantica"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
		t.Fatal(err)
	}
	if envoltorio.Data.ViaRecomendada != "bolsa_vigente" || len(envoltorio.Data.Identidad) == 0 {
		t.Fatalf("proyección incompleta: %s", respuesta.Body.String())
	}
	if respuesta.Header().Get("Set-Cookie") != "" {
		t.Fatal("emitió Set-Cookie")
	}
}

func TestManejadorCoberturaRechazaCookiesYAutoridadDeclarada(t *testing.T) {
	servicio := &servicioCoberturaPrueba{}
	manejador, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio)
	if err != nil {
		t.Fatal(err)
	}
	for _, cabecera := range []string{"Cookie", "Authorization", "X-Actor", "Perfil"} {
		t.Run(cabecera, func(t *testing.T) {
			peticion := nuevaPeticionCoberturaPrueba(RutaPropuestaCobertura, `{"expediente_ref":"expediente:ct:0001","version_esperada":1}`)
			peticion.Header.Set(cabecera, "fabricada")
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
		})
	}
}

func TestManejadorCoberturaConstruyeDecisionDesdeContextoConfiable(t *testing.T) {
	servicio := &servicioCoberturaPrueba{recibo: cobertura.ReciboOperacionDecisionCobertura{
		ReciboRef: "recibo:cobertura:001", ConfirmadaEn: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC),
		Aplicada: &cobertura.ResultadoAplicadoOperacionDecisionCobertura{DecisionCoberturaRef: "decision-cobertura:sha256:" + strings.Repeat("b", 64), DecisionCoberturaHuella: strings.Repeat("b", 64), VersionResultante: 2, EventoRef: "evento:cobertura:001", ActuacionRef: "actuacion:cobertura:001"},
	}}
	manejador, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(RutaDecisionCobertura, cuerpoDecisionCoberturaPrueba(false)))
	if respuesta.Code != http.StatusCreated {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if servicio.decidir.AutenticacionRef != contextoCoberturaValidoPrueba().AutenticacionRef || servicio.decidir.OrganizacionRef != contextoCoberturaValidoPrueba().OrganizacionRef {
		t.Fatalf("contexto no ligado: %+v", servicio.decidir)
	}
	if strings.Contains(respuesta.Body.String(), "EventoRef") || strings.Contains(respuesta.Body.String(), "evento:cobertura") {
		t.Fatalf("recibo no minimizado: %s", respuesta.Body.String())
	}
}

func TestManejadorCoberturaRectificaSoloConPredecesoraCompleta(t *testing.T) {
	servicio := &servicioCoberturaPrueba{recibo: cobertura.ReciboOperacionDecisionCobertura{ReciboRef: "recibo:cobertura:001", ConfirmadaEn: time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), DenegadaVEC: &cobertura.ResultadoDenegadoVECOperacionDecisionCobertura{}}}
	manejador, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(RutaRectificacionCobertura, cuerpoDecisionCoberturaPrueba(true)))
	if respuesta.Code != http.StatusCreated || servicio.rectificar.PredecesoraRef == "" || servicio.rectificar.MotivoClave != "rectificacion" {
		t.Fatalf("rectificación=%+v estado=%d", servicio.rectificar, respuesta.Code)
	}
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(RutaRectificacionCobertura, cuerpoDecisionCoberturaPrueba(false)))
	if respuesta.Code != http.StatusUnprocessableEntity {
		t.Fatalf("estado sin predecesora=%d", respuesta.Code)
	}
}

func TestManejadorCoberturaCierraRutasJSONYDependencias(t *testing.T) {
	servicio := &servicioCoberturaPrueba{}
	if _, err := NuevoManejadorCobertura(nil, servicio, servicio); !errors.Is(err, ErrManejadorCoberturaInvalido) {
		t.Fatalf("sin autoridad=%v", err)
	}
	if _, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{}, (*servicioCoberturaPrueba)(nil), servicio); !errors.Is(err, ErrManejadorCoberturaInvalido) {
		t.Fatalf("presentador typed nil=%v", err)
	}
	manejador, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, ruta, cuerpo string
		preparar             func(*http.Request)
		estado               int
	}{
		{"query", RutaPropuestaCobertura + "?expediente_ref=x", `{}`, nil, http.StatusNotFound},
		{"get", RutaPropuestaCobertura, `{}`, func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"duplicado", RutaPropuestaCobertura, `{"expediente_ref":"expediente:ct:0001","expediente_ref":"expediente:ct:0002","version_esperada":1}`, nil, http.StatusBadRequest},
		{"extra", RutaPropuestaCobertura, `{"expediente_ref":"expediente:ct:0001","version_esperada":1,"perfil":"forjado"}`, nil, http.StatusBadRequest},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := nuevaPeticionCoberturaPrueba(caso.ruta, caso.cuerpo)
			if caso.preparar != nil {
				caso.preparar(r)
			}
			w := httptest.NewRecorder()
			manejador.ServeHTTP(w, r)
			if w.Code != caso.estado {
				t.Fatalf("estado=%d cuerpo=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestManejadorCoberturaRechazaResultadosAdulteradosYCerrarCancelacion(t *testing.T) {
	servicio := &servicioCoberturaPrueba{propuesta: application.PresentacionPropuestaCobertura{Estado: domain.PropuestaCoberturaViable, ViaRecomendada: "bolsa_vigente", IdentidadSemantica: identidadSemanticaCoberturaPrueba(), Evaluaciones: []domain.EvaluacionViaPropuestaCobertura{{ViaClave: "bolsa_vigente", Prioridad: 1, Estado: domain.EvaluacionViaCoberturaViable, Conflictos: []domain.ClaveCatalogo{"NO VALIDA"}}}}}
	manejador, err := NuevoManejadorCobertura(autoridadCoberturaPrueba{contexto: contextoCoberturaValidoPrueba()}, servicio, servicio)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	manejador.ServeHTTP(w, nuevaPeticionCoberturaPrueba(RutaPropuestaCobertura, `{"expediente_ref":"expediente:ct:0001","version_esperada":1}`))
	if w.Code != http.StatusBadGateway {
		t.Fatalf("mutante=%d", w.Code)
	}
	servicio.propuesta = application.PresentacionPropuestaCobertura{}
	servicio.err = context.Canceled
	w = httptest.NewRecorder()
	manejador.ServeHTTP(w, nuevaPeticionCoberturaPrueba(RutaPropuestaCobertura, `{"expediente_ref":"expediente:ct:0001","version_esperada":1}`))
	if w.Code != http.StatusRequestTimeout {
		t.Fatalf("cancelación=%d", w.Code)
	}
}
