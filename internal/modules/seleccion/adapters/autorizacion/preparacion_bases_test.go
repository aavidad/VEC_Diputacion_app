package autorizacion

import (
	"context"
	"errors"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

type identidadPreparacionFalloPrueba struct {
	err      error
	llamadas int
}

func (i *identidadPreparacionFalloPrueba) ResolverIdentidadConvocatoria(context.Context) (IdentidadRegistrada, error) {
	i.llamadas++
	return IdentidadRegistrada{}, i.err
}

type emisorPreparacionNoInvocadoPrueba struct{ llamadas int }

func (e *emisorPreparacionNoInvocadoPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, vec.SolicitudAutorizacionLigadaV3, vec.ResultadoContextoActorRegistradoV2) (vec.DecisionAutorizacionLigadaV3, core.ConfirmacionRegistroConcesionAutorizacionLigadaV3, core.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return vec.DecisionAutorizacionLigadaV3{}, core.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ports.ErrPreparacionBasesNoDisponible
}

func TestPreparacionProveedorConservaDenegacionNominalYFalloTecnico(t *testing.T) {
	for _, caso := range []struct{ causa, esperado error }{
		{ports.ErrPreparacionBasesDenegada, ports.ErrPreparacionBasesDenegada},
		{ports.ErrPreparacionBasesNoDisponible, ports.ErrPreparacionBasesNoDisponible},
		{context.DeadlineExceeded, ports.ErrPreparacionBasesNoDisponible},
		{errors.Join(ports.ErrPreparacionBasesDenegada, context.DeadlineExceeded), ports.ErrPreparacionBasesNoDisponible},
	} {
		i, e := &identidadPreparacionFalloPrueba{err: caso.causa}, &emisorPreparacionNoInvocadoPrueba{}
		p := &ProveedorPreparacionBases{identidad: i, emisor: e}
		_, err := p.emitirPreparacion(context.Background(), vec.ContextoActor{}, vec.ReferenciaCorrelacionAutorizacionV2{}, bolsa.PreparacionOperacionBasesV3{}, ports.AccionGuardarPreparacionBases, bolsa.AudienciaGuardarPreparacionBasesV3)
		if !errors.Is(err, caso.esperado) || i.llamadas != 1 || e.llamadas != 0 {
			t.Fatalf("clasificacion=%v emisor=%d", err, e.llamadas)
		}
	}
}
