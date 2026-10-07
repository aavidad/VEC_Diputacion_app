package internactproveedores

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	ports "vec-diputacion-granada/internal/vec/ports"
)

type auditorOHPrueba struct {
	orden    ports.OrdenAuditoriaFronteraRutaExacta
	llamadas int
	err      error
}

func (a *auditorOHPrueba) RegistrarAuditoriaFronteraRutaExacta(_ context.Context, o ports.OrdenAuditoriaFronteraRutaExacta) error {
	a.orden = o
	a.llamadas++
	return a.err
}

func TestAuditoriaOrganizacionHistoricaNominalYFalloPropagado(t *testing.T) {
	r := &auditorOHPrueba{}
	a := AuditorDenegacionOrganizacionHistorica{Registrador: r}
	d := httpapi.DenegacionOrganizacionHistorica{CorrelacionRef: "corr_no_disponible", Motivo: "acceso_denegado", Ruta: httpapi.RutaOrganizacionHistoricaPersonal, ActorRef: "per_0123456789abcdefghijkl"}
	if err := a.RegistrarDenegacionOrganizacionHistorica(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if r.orden.Superficie != "organizacion_historica_personal" || r.orden.Ruta != d.Ruta || r.orden.ActorRef != d.ActorRef {
		t.Fatalf("orden %v", r.orden)
	}
	r.err = errors.New("sin destino")
	if err := a.RegistrarDenegacionOrganizacionHistorica(context.Background(), d); !errors.Is(err, r.err) {
		t.Fatal("fallo auditor perdido")
	}
	d.Ruta += "?vigente_en=2026-01-01"
	if err := a.RegistrarDenegacionOrganizacionHistorica(context.Background(), d); err == nil || r.llamadas != 2 {
		t.Fatal("filtro transportado a auditoría")
	}
}
