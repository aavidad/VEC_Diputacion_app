package composicion

import (
	"context"
	"sync"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type claveIntentoLectorRelacionRPT struct{}

type intentoLectorRelacionRPT struct {
	identidad  IdentidadRegistradaLectorRelacionRPT
	referencia string
	mu         sync.Mutex
	orden      *vecports.OrdenIntentoAuditoria
	acuse      *vecports.AcuseIntentoAuditoria
}

// LectorRelacionSeleccionadaRPT conserva la operación que consume la preparación RPT.
type LectorRelacionSeleccionadaRPT interface {
	ConsultarSeleccionada(context.Context, vecdomain.ContextoActor, personaldomain.PreparacionRelacionParaRPT, string, string) (personalports.ResultadoRelacionParaRPTV1, error)
}

type lectorRelacionSeleccionadaCorrelacion struct {
	siguiente       LectorRelacionSeleccionadaRPT
	identidad       ResolutorIdentidadLectorRelacionRPT
	limiteIdentidad time.Duration
}

// NuevoLectorRelacionSeleccionadaRPTConIdentidad enlaza la selección existente
// con la frontera nominal y el contexto técnico común, también fuera de HTTP.
func NuevoLectorRelacionSeleccionadaRPTConIdentidad(l LectorRelacionSeleccionadaRPT, i ResolutorIdentidadLectorRelacionRPT, limite time.Duration) (LectorRelacionSeleccionadaRPT, error) {
	if dependenciaNula(l) || dependenciaNula(i) || limite <= 0 || limite > 30*time.Second {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return lectorRelacionSeleccionadaCorrelacion{siguiente: l, identidad: i, limiteIdentidad: limite}, nil
}

func (l lectorRelacionSeleccionadaCorrelacion) ConsultarSeleccionada(ctx context.Context, actor vecdomain.ContextoActor, preparacion personaldomain.PreparacionRelacionParaRPT, organismo, seleccion string) (personalports.ResultadoRelacionParaRPTV1, error) {
	var vacio personalports.ResultadoRelacionParaRPTV1
	if ctx == nil || dependenciaNula(l.siguiente) || dependenciaNula(l.identidad) || l.limiteIdentidad <= 0 || l.limiteIdentidad > 30*time.Second {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	if _, err := correlacionLectorRelacionRPT(ctx); err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	// La identidad histórica se captura antes de validar o abrir la lectura.
	// Una cancelación posterior no sustituye persona, perfil ni vínculo originales.
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), l.limiteIdentidad)
	defer cancelar()
	identidad, err := l.identidad.ResolverIdentidadLectorRelacionRPT(auditCtx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	copia, err := identidad.Resultado.Clonar()
	if err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	identidad.Resultado = copia
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	intento := &intentoLectorRelacionRPT{identidad: identidad, referencia: ref}
	return l.siguiente.ConsultarSeleccionada(context.WithValue(ctx, claveIntentoLectorRelacionRPT{}, intento), actor, preparacion, organismo, seleccion)
}

func identidadOriginalLectorRelacionRPT(ctx context.Context) (IdentidadRegistradaLectorRelacionRPT, error) {
	intento, err := estadoIntentoLectorRelacionRPT(ctx)
	if err != nil {
		return IdentidadRegistradaLectorRelacionRPT{}, err
	}
	copia, err := intento.identidad.Resultado.Clonar()
	if err != nil {
		return IdentidadRegistradaLectorRelacionRPT{}, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return IdentidadRegistradaLectorRelacionRPT{Vinculo: intento.identidad.Vinculo, Resultado: copia}, nil
}

func estadoIntentoLectorRelacionRPT(ctx context.Context) (*intentoLectorRelacionRPT, error) {
	if ctx != nil {
		if intento, ok := ctx.Value(claveIntentoLectorRelacionRPT{}).(*intentoLectorRelacionRPT); ok && intento != nil {
			return intento, nil
		}
	}
	return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
}

// El generador sólo traduce la marca privada común; no admite texto del cliente
// ni acuña otra correlación. V3 y los registros técnicos comparten los mismos bits.
type correlacionTecnicaLectorRPT struct{}

func (correlacionTecnicaLectorRPT) NuevaReferenciaCorrelacionAutorizacionV2(ctx context.Context) (string, error) {
	ref, ok := vecports.CorrelacionIncidenciasPeticion(ctx)
	if !ok {
		return "", personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return "correlacion_" + ref, nil
}

func correlacionLectorRelacionRPT(ctx context.Context) (vecdomain.ReferenciaCorrelacionAutorizacionV2, error) {
	if ctx == nil {
		return vecdomain.ReferenciaCorrelacionAutorizacionV2{}, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.WithoutCancel(ctx), correlacionTecnicaLectorRPT{})
}

var _ LectorRelacionSeleccionadaRPT = lectorRelacionSeleccionadaCorrelacion{}
