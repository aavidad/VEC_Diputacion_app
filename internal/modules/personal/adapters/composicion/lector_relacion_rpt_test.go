package composicion

import (
	"context"
	"errors"
	"testing"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestClasificarErrorAutorizacionLectorRelacionRPT(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		ctx      context.Context
		err      error
		esperado error
	}{
		{"denegacion registrada", context.Background(), vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrLectorRelacionRPTDenegado},
		{"emisor caido", context.Background(), errors.New("emisor caido"), personaldomain.ErrLectorRelacionRPTNoDisponible},
		{"registro de denegacion caido", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible), personaldomain.ErrLectorRelacionRPTNoDisponible},
		{"timeout mezclado", context.Background(), errors.Join(vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, context.DeadlineExceeded), personaldomain.ErrLectorRelacionRPTNoDisponible},
		{"sin contexto", nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3, personaldomain.ErrLectorRelacionRPTNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := clasificarErrorAutorizacionLectorRelacionRPT(caso.ctx, caso.err); !errors.Is(got, caso.esperado) {
				t.Fatalf("clasificación: %v", got)
			}
		})
	}
}

type identidadLectorRelacionRPTPrueba struct{ err error }

func (i identidadLectorRelacionRPTPrueba) ResolverIdentidadLectorRelacionRPT(context.Context) (IdentidadRegistradaLectorRelacionRPT, error) {
	return IdentidadRegistradaLectorRelacionRPT{}, i.err
}

type emisorLectorRelacionRPTPrueba struct{ llamadas int }

func (e *emisorLectorRelacionRPTPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("no debe emitirse")
}

func TestProveedorLectorRelacionRPTNoEmiteSinIdentidadDeLaPeticion(t *testing.T) {
	if _, err := NuevoProveedorAutorizacionLectorRelacionRPT(identidadLectorRelacionRPTPrueba{}, &emisorLectorRelacionRPTPrueba{}, vecdomain.ReferenciaEntradaCatalogo{}); err == nil {
		t.Fatal("proveedor compuesto sin motivo válido")
	}
	emisor := &emisorLectorRelacionRPTPrueba{}
	p := &ProveedorAutorizacionLectorRelacionRPT{identidad: identidadLectorRelacionRPTPrueba{err: errors.New("sin sesión")}, emisor: emisor}
	if _, err := p.AutorizarRelacionParaRPT(context.Background(), personaldomain.MaterialLectorRelacionRPT{}); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) || emisor.llamadas != 0 {
		t.Fatal("se emitió sin material o sin identidad", err)
	}
}
