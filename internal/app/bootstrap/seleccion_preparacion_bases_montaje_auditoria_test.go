package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registroAuditoriaPreparacionBasesPrueba struct {
	ordenes []vecports.OrdenAuditoriaFronteraRutaExacta
	err     error
}

func (a *registroAuditoriaPreparacionBasesPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o vecports.OrdenAuditoriaFronteraRutaExacta) error {
	a.ordenes = append(a.ordenes, o)
	return a.err
}

func TestPreparacionBasesAuditoriaCallbackComunMinimizado(t *testing.T) {
	a := &registroAuditoriaPreparacionBasesPrueba{}
	callback, err := NuevaAuditoriaFronteraPreparacionBasesV3(a)
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{selhttp.RutaGuardarPreparacionBases, selhttp.RutaConsultarPreparacionBases} {
		r := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(`{"actor_ref":"cliente","correlacion":"cliente"}`))
		r.Header.Set("X-Actor-Ref", "cliente")
		r.Header.Set("X-Correlation-ID", "cliente")
		if callback(r) != nil {
			t.Fatal("auditor común rechazó ruta nominal")
		}
		o := a.ordenes[len(a.ordenes)-1]
		if o.Validar() != nil || o.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases ||
			o.Ruta != ruta || o.ActorRef != "" || o.Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado ||
			!strings.HasPrefix(o.CorrelacionRef, "corr_") || o.CorrelacionRef == "corr_no_disponible" || len(o.CorrelacionRef) != len("corr_")+32 {
			t.Fatal("auditoría dejó de usar superficie nominal o datos minimizados")
		}
	}
	previas := len(a.ordenes)
	for _, r := range []*http.Request{
		httptest.NewRequest(http.MethodGet, selhttp.RutaGuardarPreparacionBases, nil),
		httptest.NewRequest(http.MethodPost, selhttp.RutaGuardarPreparacionBases+"?actor=cliente", nil),
		httptest.NewRequest(http.MethodPost, selhttp.RutaGuardarPreparacionBases+"/otra", nil),
		httptest.NewRequest(http.MethodPost, "/api/vec/bolsa/reglas-baremo/borradores/alta", nil),
	} {
		if !errors.Is(callback(r), bolsaports.ErrPreparacionBasesNoDisponible) || len(a.ordenes) != previas {
			t.Fatal("callback abrió otra ruta o método")
		}
	}
	a.err = errors.New("fallo sintético de auditoría")
	if !errors.Is(callback(httptest.NewRequest(http.MethodPost, selhttp.RutaGuardarPreparacionBases, nil)), bolsaports.ErrPreparacionBasesNoDisponible) {
		t.Fatal("fallo del registrador no mantuvo cierre técnico")
	}
	var nulo *registroAuditoriaPreparacionBasesPrueba
	if _, err := NuevaAuditoriaFronteraPreparacionBasesV3(nulo); err == nil {
		t.Fatal("auditor tipado nulo admitido")
	}
}

func TestPreparacionBasesCSRFEnlazaAuditorComun(t *testing.T) {
	for i := range 2 {
		broker, ctx := brokerPreparacionBasesPrueba(t, i)
		a := &registroAuditoriaPreparacionBasesPrueba{}
		callback, err := NuevaAuditoriaFronteraPreparacionBasesV3(a)
		if err != nil {
			t.Fatal(err)
		}
		frontera, err := nuevaFronteraPreparacionBasesHTTPV3(broker, callback)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(http.MethodPost, broker.perfiles[i].ruta, nil).WithContext(ctx)
		r.Header.Set("Sec-Fetch-Site", "cross-site")
		if !errors.Is(frontera(r), bolsaports.ErrPreparacionBasesDenegada) || len(a.ordenes) != 1 ||
			a.ordenes[0].Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases || a.ordenes[0].ActorRef != "" {
			t.Fatal("rechazo CSRF no confirmó auditoría común minimizada")
		}
		a.err = errors.New("fallo sintético de auditoría")
		if !errors.Is(frontera(r), bolsaports.ErrPreparacionBasesNoDisponible) {
			t.Fatal("rechazo sin auditoría confirmada dejó de ser cierre técnico")
		}
	}
}

func TestPreparacionBasesSesionRechazadaAuditaAntesDelPDP(t *testing.T) {
	for i := range 2 {
		for _, auditorDisponible := range []bool{true, false} {
			broker, ctx := brokerPreparacionBasesPrueba(t, i)
			a := &registroAuditoriaPreparacionBasesPrueba{}
			if !auditorDisponible {
				a.err = errors.New("fallo sintético de auditoría")
			}
			callback, err := NuevaAuditoriaFronteraPreparacionBasesV3(a)
			if err != nil {
				t.Fatal(err)
			}
			broker.registrarRechazo = callback
			pdp := &pdpPreparacionBasesDenegadoPrueba{}
			broker.pdp = pdp
			// La capacidad y la frontera son válidas, pero el holder pertenece
			// a otra sesión. El rechazo sucede al resolver contexto HTTP.
			c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			c.contextoOperacion = &contextoOperacionCTDesarrollo{soporte: broker.perfiles[1-i].soporte, contexto: broker.perfiles[1-i].soporte.contexto}
			ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			frontera, err := nuevaFronteraPreparacionBasesHTTPV3(broker, callback)
			if err != nil {
				t.Fatal(err)
			}
			preparador := &servicioPreparacionFronteraPrueba{}
			h, err := selhttp.NuevaPreparacionBasesHandler(selhttp.ConfigPreparacionBases{Preparador: preparador, ResolverContexto: broker.ResolverContextoHTTP, ValidarFrontera: frontera})
			if err != nil {
				t.Fatal(err)
			}
			cuerpo := `{"modo":"actual","preparacion_ref":"prep:sintetica"}`
			if i == 0 {
				cuerpo = `{"esperada":{"preparacion_ref":"prep:sintetica","revision":0,"huella_material_sha256":""},"material":{"contenido":{},"referencias":[]},"clave_operacion":"operacion:sintetica"}`
			}
			r := httptest.NewRequest(http.MethodPost, broker.perfiles[i].ruta, strings.NewReader(cuerpo)).WithContext(ctx)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Actor-Ref", "cliente")
			if err := frontera(r); err != nil || len(a.ordenes) != 0 {
				t.Fatal("fixture rechazado antes del resolver de contexto")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			esperado := http.StatusForbidden
			if !auditorDisponible {
				esperado = http.StatusServiceUnavailable
			}
			if w.Code != esperado || preparador.efectos != 0 || pdp.llamadas != 0 || len(a.ordenes) != 1 {
				t.Fatalf("ruta=%s auditor=%t HTTP=%d servicio=%d PDP=%d auditoría=%d", r.URL.Path, auditorDisponible, w.Code, preparador.efectos, pdp.llamadas, len(a.ordenes))
			}
			if a.ordenes[0].ActorRef != "" || a.ordenes[0].Ruta != r.URL.Path || a.ordenes[0].Motivo != vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado || strings.Contains(w.Body.String(), `"preparacion"`) {
				t.Fatal("rechazo expone identidad o datos antes de resolver contexto")
			}
		}
	}
}

func TestPreparacionBasesSesionSinAuditorCierraYFalloTecnicoNoFingeDenegacion(t *testing.T) {
	p, ctx := brokerPreparacionBasesPrueba(t, 0)
	r := httptest.NewRequest(http.MethodPost, p.perfiles[0].ruta, nil)
	if _, err := p.ResolverContextoHTTP(r); !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) {
		t.Fatal("denegación sin auditor no cerró técnicamente")
	}
	llamadas := 0
	p.registrarRechazo = func(*http.Request) error { llamadas++; return nil }
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := p.ResolverContextoHTTP(r.WithContext(cancelado)); !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) || llamadas != 0 {
		t.Fatal("fallo técnico se convirtió en denegación nominal")
	}
}
