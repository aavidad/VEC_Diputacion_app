package composicion

import (
	"context"
	"errors"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestClasificarErrorAutorizacionFichaPropia(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		err      error
		esperado error
	}{
		{"denegacion registrada", context.Background(), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrFichaPropiaDenegada},
		{"emisor caido", context.Background(), errors.New("emisor caido"), personaldomain.ErrFichaPropiaNoDisponible},
		{"registro de denegacion caido", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), personaldomain.ErrFichaPropiaNoDisponible},
		{"timeout mezclado", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.DeadlineExceeded), personaldomain.ErrFichaPropiaNoDisponible},
		{"sin contexto", nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrFichaPropiaNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := clasificarErrorAutorizacionFichaPropia(caso.ctx, caso.err); !errors.Is(got, caso.esperado) {
				t.Fatalf("clasificación: %v", got)
			}
		})
	}
}

type identidadFichaPropiaPrueba struct{ err error }

func (i identidadFichaPropiaPrueba) ResolverIdentidadFichaPropia(context.Context) (IdentidadRegistradaFichaPropia, error) {
	return IdentidadRegistradaFichaPropia{}, i.err
}

type emisorFichaPropiaPrueba struct{ llamadas int }

func (e *emisorFichaPropiaPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("no debe emitirse")
}

func TestProveedorFichaPropiaNoEmiteSinIdentidadDeLaPeticion(t *testing.T) {
	if _, err := NuevoProveedorAutorizacionFichaPropia(identidadFichaPropiaPrueba{}, &emisorFichaPropiaPrueba{}, vecdomain.ReferenciaEntradaCatalogo{}); err == nil {
		t.Fatal("proveedor compuesto sin motivo válido")
	}
	emisor := &emisorFichaPropiaPrueba{}
	p := &ProveedorAutorizacionFichaPropia{identidad: identidadFichaPropiaPrueba{err: errors.New("sin sesión")}, emisor: emisor}
	if _, err := p.AutorizarFichaPropia(context.Background(), personaldomain.MaterialFichaPropia{}); !errors.Is(err, personaldomain.ErrFichaPropiaNoDisponible) || emisor.llamadas != 0 {
		t.Fatal("se emitió sin material o sin identidad", err)
	}
}
