package application

import (
	"context"
	"errors"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorAsignacionContratoPrueba struct{ llamadas int }

func (p *proveedorAsignacionContratoPrueba) AutorizarAsignacionDietas(context.Context, personaldomain.MaterialAsignacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("autorización no configurada")
}

type repositorioAsignacionContratoPrueba struct{ llamadas int }

func (r *repositorioAsignacionContratoPrueba) EjecutarAsignacionDietas(context.Context, personalports.OrdenAsignacionDietas) (personalports.ResultadoAsignacionDietas, error) {
	r.llamadas++
	return personalports.ResultadoAsignacionDietas{}, errors.New("repositorio no configurado")
}

func TestConsultaPropiaNoReinterpretaContratoAntiguo(t *testing.T) {
	p, r := &proveedorAsignacionContratoPrueba{}, &repositorioAsignacionContratoPrueba{}
	s, err := NuevoServicioAsignacionDietas(p, r)
	if err != nil {
		t.Fatal(err)
	}
	if audiencia, accion := contratoAsignacion(personaldomain.ConsultarAsignacionDietas); audiencia != personalports.AudienciaConsultarAsignacionDietas || accion != personalports.AccionConsultarAsignacionDietas {
		t.Fatal("se alteró la consulta antigua")
	}
	if audiencia, accion := contratoAsignacion(personaldomain.ConsultarPropiaAsignacionDietas); audiencia != personalports.AudienciaConsultarPropiaAsignacionDietas || accion != personalports.AccionConsultarPropiaAsignacionDietas {
		t.Fatal("consulta propia sin audiencia o acción nominal nueva")
	}
	if _, err := s.Ejecutar(context.Background(), personaldomain.SolicitudAsignacionDietas{Operacion: personaldomain.ConsultarPropiaAsignacionDietas}); !errors.Is(err, personalports.ErrSolicitudAsignacionDietasInvalida) || p.llamadas != 0 || r.llamadas != 0 {
		t.Fatalf("la ruta antigua aceptó consulta propia: %v", err)
	}
	if _, err := s.ConsultarPropia(context.Background(), personaldomain.SolicitudAsignacionDietas{Operacion: personaldomain.ConsultarAsignacionDietas}); !errors.Is(err, personalports.ErrSolicitudAsignacionDietasInvalida) || p.llamadas != 0 || r.llamadas != 0 {
		t.Fatalf("la ruta propia aceptó consulta antigua: %v", err)
	}
}
