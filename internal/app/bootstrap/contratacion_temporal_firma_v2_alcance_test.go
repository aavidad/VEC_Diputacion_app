package bootstrap

import (
	"context"
	"strings"
	"testing"

	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type resolutorAlcanceFirmaV2Prueba struct {
	proyecciones core.AlcanceProyeccionesContextoActor
	llamadas     int
}

func (r *resolutorAlcanceFirmaV2Prueba) ResolverYRegistrarContextoActorV2(
	_ context.Context, solicitud vp.SolicitudResolucionRegistroContextoActorV2,
) (vp.ConfirmacionRegistroContextoActorV2, error) {
	r.proyecciones = solicitud.Proyecciones
	r.llamadas++
	return vp.ConfirmacionRegistroContextoActorV2{}, vp.ErrResolutorRegistroContextoActorNoDisponible
}

func TestFirmaVecF1PideEmpleadoSinAmpliarConsultasCT(t *testing.T) {
	peticion := core.SolicitudContextoActor{
		Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: "cta_" + strings.Repeat("a", 32),
			Metodo:    core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial,
		},
		PerfilActivoRef: "prf_" + strings.Repeat("b", 32),
	}
	firma := &resolutorAlcanceFirmaV2Prueba{}
	autoridad, err := nuevaAutoridadContextoEmpleadoFirmaVecR5(firma, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := autoridad.ResolverContextoActorRegistradoV2(t.Context(), peticion); err == nil ||
		firma.llamadas != 1 || !firma.proyecciones.IncluyeEmpleado() {
		t.Fatalf("F1 no exigió el empleado canónico o aceptó fuente ausente: llamadas=%d error=%v", firma.llamadas, err)
	}

	legado := &resolutorAlcanceFirmaV2Prueba{}
	servicio, err := vecapp.NuevoServicioContextoActorProductivoV2(
		legado, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	autoridadLegada, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicio)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := autoridadLegada.ResolverContextoActorRegistradoV2(context.Background(), peticion); err == nil ||
		legado.llamadas != 1 || !legado.proyecciones.Vacio() {
		t.Fatalf("las consultas CT cambiaron de alcance: llamadas=%d error=%v", legado.llamadas, err)
	}
}
