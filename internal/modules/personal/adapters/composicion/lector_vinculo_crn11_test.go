package composicion

import (
	"context"
	"errors"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestClasificarErrorAutorizacionVinculoCRN11(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		err      error
		esperado error
	}{
		{"denegacion registrada", context.Background(), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrVinculoCRN11Denegado},
		{"emisor caido", context.Background(), errors.New("emisor caido"), personaldomain.ErrVinculoCRN11NoDisponible},
		{"registro de denegacion caido", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), personaldomain.ErrVinculoCRN11NoDisponible},
		{"timeout mezclado", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.DeadlineExceeded), personaldomain.ErrVinculoCRN11NoDisponible},
		{"sin contexto", nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrVinculoCRN11NoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := clasificarErrorAutorizacionVinculoCRN11(caso.ctx, caso.err); !errors.Is(got, caso.esperado) {
				t.Fatalf("clasificación: %v", got)
			}
		})
	}
}

type identidadVinculoCRN11Prueba struct{ err error }

func (i identidadVinculoCRN11Prueba) ResolverIdentidadVinculoCRN11(context.Context) (IdentidadRegistradaVinculoCRN11, error) {
	return IdentidadRegistradaVinculoCRN11{}, i.err
}

type emisorVinculoCRN11Prueba struct{ llamadas int }

func (e *emisorVinculoCRN11Prueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("no debe emitirse")
}

func TestProveedorVinculoCRN11NoEmiteSinIdentidadDeLaPeticion(t *testing.T) {
	if _, err := NuevoProveedorAutorizacionVinculoCRN11(identidadVinculoCRN11Prueba{}, &emisorVinculoCRN11Prueba{}, vecdomain.ReferenciaEntradaCatalogo{}); err == nil {
		t.Fatal("proveedor compuesto sin motivo válido")
	}
	emisor := &emisorVinculoCRN11Prueba{}
	p := &ProveedorAutorizacionVinculoCRN11{identidad: identidadVinculoCRN11Prueba{err: errors.New("sin sesión")}, emisor: emisor}
	if _, err := p.AutorizarVinculoPropioCRN11(context.Background(), personaldomain.MaterialVinculoPropioCRN11{}); !errors.Is(err, personaldomain.ErrVinculoCRN11NoDisponible) || emisor.llamadas != 0 {
		t.Fatal("se emitió sin material o sin identidad", err)
	}
}
