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
