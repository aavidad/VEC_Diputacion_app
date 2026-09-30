package application

import (
	"context"
	"errors"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type fuentePlanB2Prueba struct{ llamadas int }

func (f *fuentePlanB2Prueba) ResolverPlanNominalB2(context.Context, ports.SolicitudPlanNominalB2, ports.ActorIncorporacionPersonalB2) (domain.PlanIncorporacionPersonalB2, error) {
	f.llamadas++
	return domain.PlanIncorporacionPersonalB2{}, ports.ErrPlanNominalB2NoDisponible
}
func (f *fuentePlanB2Prueba) VerificarHechosPersonalB2(context.Context, ports.ContratoPlanNominalB2, ports.HechosPersonalIncorporacionB2, ports.ActorIncorporacionPersonalB2) (ports.HechosPersonalIncorporacionB2, error) {
	f.llamadas++
	return ports.HechosPersonalIncorporacionB2{}, ports.ErrPlanNominalB2NoDisponible
}

type autoridadPlanB2Prueba struct {
	llamadas int
	accion   string
}

func (a *autoridadPlanB2Prueba) AutorizarPlanNominalB2(_ context.Context, accion string, _ []byte, _ ports.ActorIncorporacionPersonalB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.accion = accion
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrPlanNominalB2Denegado
}

type repoPlanB2Prueba struct{ llamadas int }

func (r *repoPlanB2Prueba) RegistrarPlanNominalB2(context.Context, ports.RegistroPlanNominalB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ContratoPlanNominalB2, error) {
	r.llamadas++
	return ports.ContratoPlanNominalB2{}, nil
}
func (r *repoPlanB2Prueba) LeerContratoPlanNominal(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.ContratoPlanNominalB2, error) {
	r.llamadas++
	return ports.ContratoPlanNominalB2{}, nil
}
func (r *repoPlanB2Prueba) ConfirmarOrigenIncorporacionB2(context.Context, ports.ConfirmacionOrigenIncorporacionB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.OrigenIncorporacionPersonalB2, error) {
	r.llamadas++
	return ports.OrigenIncorporacionPersonalB2{}, nil
}
func (r *repoPlanB2Prueba) LeerOrigenIncorporacionB2(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.OrigenIncorporacionPersonalB2, bool, error) {
	r.llamadas++
	return ports.OrigenIncorporacionPersonalB2{}, false, nil
}
func TestPlanPersonalB2RevalidacionNominalEnCadaLectura(t *testing.T) {
	f, a, r := &fuentePlanB2Prueba{}, &autoridadPlanB2Prueba{}, &repoPlanB2Prueba{}
	s, e := NuevoServicioPlanNominalB2(f, a, r)
	if e != nil {
		t.Fatal(e)
	}
	for range 2 {
		if _, e = s.LeerContratoPlanNominal(context.Background(), "org:uno", "expediente:uno"); !errors.Is(e, ports.ErrPlanNominalB2Denegado) {
			t.Fatal(e)
		}
	}
	if _, _, e = s.LeerOrigenIncorporacionB2(context.Background(), "org:uno", "expediente:uno"); !errors.Is(e, ports.ErrPlanNominalB2Denegado) {
		t.Fatal(e)
	}
	if a.llamadas != 3 || a.accion != ports.AccionLeerPlanNominalB2 || r.llamadas != 0 || f.llamadas != 0 {
		t.Fatal("lectura/replay evita concesión nominal actual o ejecuta un efecto")
	}
}
func TestPlanPersonalB2NoReservaPersonalParaActorSinResolver(t *testing.T) {
	f, a, r := &fuentePlanB2Prueba{}, &autoridadPlanB2Prueba{}, &repoPlanB2Prueba{}
	s, _ := NuevoServicioPlanNominalB2(f, a, r)
	_, e := s.RegistrarPlanNominalB2(context.Background(), ports.SolicitudPlanNominalB2{}, ports.ActorIncorporacionPersonalB2{})
	if !errors.Is(e, ports.ErrPlanNominalB2Invalido) || f.llamadas+a.llamadas+r.llamadas != 0 {
		t.Fatal("petición sin actor llega a una reserva")
	}
}

func (f *fuentePlanB2Prueba) ResolverUnidadPlanNominalB2(context.Context, string, string) (string, error) {
	return "unidad:rrhh", nil
}
