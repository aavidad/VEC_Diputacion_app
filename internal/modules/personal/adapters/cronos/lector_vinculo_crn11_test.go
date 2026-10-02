package cronos

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuenteVinculoCRN11Prueba func(context.Context, domain.SolicitudVinculoPropioCRN11) (personalports.ResultadoVinculoPropioCRN11, error)

func (f fuenteVinculoCRN11Prueba) ConsultarVinculoPropioCRN11(ctx context.Context, in domain.SolicitudVinculoPropioCRN11) (personalports.ResultadoVinculoPropioCRN11, error) {
	return f(ctx, in)
}

func TestPuenteCRN11ProyectaSoloCincoCamposYCuatroDeEvidencia(t *testing.T) {
	instante := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	in := cronosports.InputConsultaVinculoPropioCRN11{EmpleadoRef: "emp_" + strings.Repeat("a", 24)}
	lecturas := 0
	f := fuenteVinculoCRN11Prueba(func(_ context.Context, got domain.SolicitudVinculoPropioCRN11) (personalports.ResultadoVinculoPropioCRN11, error) {
		lecturas++
		if got.EmpleadoRef != in.EmpleadoRef {
			t.Fatal("selector alterado")
		}
		return personalports.ResultadoVinculoPropioCRN11{
			Vinculo:   domain.VinculoHistoricoCRN11{PersonaRef: "per_" + strings.Repeat("a", 24), EmpleadoRef: got.EmpleadoRef, VinculoRef: "pep_" + strings.Repeat("a", 24), FuenteRef: "prc_" + strings.Repeat("a", 24), Version: 3},
			Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:personal:1", DecisionRef: "decision:personal:1", EfectoRef: got.EmpleadoRef, ConsumoHuellaSHA256: strings.Repeat("f", 64), AuditoriaRef: "auditoria:personal:1", ConsultadaEn: instante},
		}, nil
	})
	l, err := NuevoLectorVinculoPropioCRN11(f)
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.ConsultarVinculoPropioCRN11(context.Background(), in)
	if err != nil || lecturas != 1 {
		t.Fatalf("consulta: %v, llamadas=%d", err, lecturas)
	}
	if reflect.TypeOf(got).NumField() != 6 || reflect.TypeOf(got.Evidencia).NumField() != 4 {
		t.Fatal("proyeccion Cronos incorpora campos ajenos")
	}
	if got.PersonaRef != "per_"+strings.Repeat("a", 24) || got.EmpleadoRef != in.EmpleadoRef || got.VinculoRef != "pep_"+strings.Repeat("a", 24) || got.FuenteRef != "prc_"+strings.Repeat("a", 24) || got.Version != 3 || got.Evidencia.ReciboRef != "recibo:personal:1" || got.Evidencia.DecisionRef != "decision:personal:1" || got.Evidencia.AuditoriaRef != "auditoria:personal:1" || !got.Evidencia.ConsultadaEn.Equal(instante) {
		t.Fatalf("proyeccion incompleta: %+v", got)
	}
}

func TestPuenteCRN11TraduceErroresSinDetalle(t *testing.T) {
	for _, caso := range []struct {
		nombre           string
		origen, esperado error
	}{
		{"denegado", domain.ErrVinculoCRN11Denegado, cronosports.ErrCorreccionNoAutorizada},
		{"invalido", domain.ErrVinculoCRN11Invalido, cronosports.ErrCorreccionNoAutorizada},
		{"indisponible", domain.ErrVinculoCRN11NoDisponible, cronosports.ErrDependenciaNoDisponible},
		{"detalle_interno", errors.New("detalle sensible de Personal"), cronosports.ErrDependenciaNoDisponible},
		{"cancelado", context.Canceled, context.Canceled},
		{"plazo", context.DeadlineExceeded, context.DeadlineExceeded},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f := fuenteVinculoCRN11Prueba(func(context.Context, domain.SolicitudVinculoPropioCRN11) (personalports.ResultadoVinculoPropioCRN11, error) {
				return personalports.ResultadoVinculoPropioCRN11{}, caso.origen
			})
			l, _ := NuevoLectorVinculoPropioCRN11(f)
			got, err := l.ConsultarVinculoPropioCRN11(context.Background(), cronosports.InputConsultaVinculoPropioCRN11{})
			if !errors.Is(err, caso.esperado) || got != (cronosports.VinculoPropioHistoricoCRN11{}) || strings.Contains(err.Error(), "sensible") {
				t.Fatalf("error filtrado: %+v, %v", got, err)
			}
		})
	}
}

func TestPuenteCRN11ConstructorYReceptorNulos(t *testing.T) {
	var fuenteNula *fuenteVinculoCRN11Prueba
	for _, f := range []personalports.LectorVinculoPropioCRN11{nil, fuenteNula} {
		l, err := NuevoLectorVinculoPropioCRN11(f)
		if l != nil || !errors.Is(err, cronosports.ErrDependenciaNoDisponible) {
			t.Fatal("lector nulo aceptado", err)
		}
	}
	var l *LectorVinculoPropioCRN11
	got, err := l.ConsultarVinculoPropioCRN11(context.Background(), cronosports.InputConsultaVinculoPropioCRN11{})
	if !errors.Is(err, cronosports.ErrDependenciaNoDisponible) || got != (cronosports.VinculoPropioHistoricoCRN11{}) {
		t.Fatal("receptor nulo aceptado", err)
	}
}

type relojPuenteCRN11Prueba struct{ instante time.Time }

func (r relojPuenteCRN11Prueba) AhoraUTC() time.Time { return r.instante }

type proveedorCorreccionPuenteCRN11Prueba struct{}

func (proveedorCorreccionPuenteCRN11Prueba) ProveerMaterialCorreccion(context.Context, cronosdomain.MaterialAutorizacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, cronosports.ErrDependenciaNoDisponible
}

type repoCorreccionPuenteCRN11Prueba struct {
	original             cronosports.ReciboCorreccion
	lecturas, escrituras int
	decisiones           []string
}

func (r *repoCorreccionPuenteCRN11Prueba) SolicitarOlvido(context.Context, cronosdomain.SolicitudCorreccion, cronosports.OrdenConsumoCorreccion) (cronosports.ReciboCorreccion, error) {
	r.escrituras++
	return cronosports.ReciboCorreccion{}, cronosports.ErrDependenciaNoDisponible
}
func (r *repoCorreccionPuenteCRN11Prueba) RegistrarActuacion(context.Context, cronosdomain.ActuacionCorreccion, cronosports.OrdenConsumoCorreccion) (cronosports.ReciboCorreccion, error) {
	r.escrituras++
	return cronosports.ReciboCorreccion{}, cronosports.ErrDependenciaNoDisponible
}
func (r *repoCorreccionPuenteCRN11Prueba) RecuperarRecibo(_ context.Context, _ cronosports.ClaveRecuperacionCorreccion, orden cronosports.OrdenConsumoCorreccion) (cronosports.ReciboCorreccion, error) {
	r.lecturas++
	v, ok := orden.VinculoPropioHistoricoCRN11()
	if !ok {
		return cronosports.ReciboCorreccion{}, cronosports.ErrDependenciaNoDisponible
	}
	r.decisiones = append(r.decisiones, v.Evidencia.DecisionRef)
	return r.original, nil
}

func TestPuenteCRN11ConConsumidorRealConservaReciboYRenuevaLectura(t *testing.T) {
	instante := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "pep_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, instante)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := cronosports.NuevaOrdenConsumoCorreccion(actor, proveedorCorreccionPuenteCRN11Prueba{})
	if err != nil {
		t.Fatal(err)
	}
	clave := cronosports.ClaveRecuperacionCorreccion{SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001", Paso: cronosdomain.PasoSolicitudCorreccion}
	original := cronosports.ReciboCorreccion{SolicitudRef: clave.SolicitudRef, ActuacionRef: "correccion:actuacion:1", ReciboRef: "recibo:cronos:1", Estado: cronosdomain.CorreccionPendienteResponsable, Version: 1, InstanteUTC: instante.Add(-time.Minute), Replay: true}
	repo := &repoCorreccionPuenteCRN11Prueba{original: original}
	consultas := 0
	f := fuenteVinculoCRN11Prueba(func(_ context.Context, in domain.SolicitudVinculoPropioCRN11) (personalports.ResultadoVinculoPropioCRN11, error) {
		consultas++
		if in.Actor.PersonaRef != actor.PersonaRef || in.EmpleadoRef != "emp_"+z {
			t.Fatal("consulta cruzada")
		}
		return personalports.ResultadoVinculoPropioCRN11{Vinculo: domain.VinculoHistoricoCRN11{PersonaRef: in.Actor.PersonaRef, EmpleadoRef: in.EmpleadoRef, VinculoRef: "pep_" + z, FuenteRef: "prc_" + z, Version: 1}, Evidencia: personalports.EvidenciaRegistroEmpleadoB2{ReciboRef: fmt.Sprintf("recibo:personal:%d", consultas), DecisionRef: fmt.Sprintf("decision:personal:%d", consultas), AuditoriaRef: fmt.Sprintf("auditoria:personal:%d", consultas), ConsultadaEn: instante}}, nil
	})
	lector, err := NuevoLectorVinculoPropioCRN11(f)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := cronosapp.NuevoServicioCorreccionesConVinculoHistorico(repo, relojPuenteCRN11Prueba{instante}, lector)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		got, err := servicio.RecuperarRecibo(context.Background(), orden, clave)
		if err != nil || got != original {
			t.Fatalf("recibo inicial alterado: %+v, %v", got, err)
		}
	}
	if consultas != 2 || repo.lecturas != 2 || repo.escrituras != 0 || repo.decisiones[0] == repo.decisiones[1] {
		t.Fatalf("lecturas nuevas o historia alterada: consultas=%d lecturas=%d escrituras=%d decisiones=%v", consultas, repo.lecturas, repo.escrituras, repo.decisiones)
	}
}
