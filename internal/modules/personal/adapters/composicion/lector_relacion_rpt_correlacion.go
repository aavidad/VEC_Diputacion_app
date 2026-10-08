package composicion

import (
	"context"
	"errors"
	"sync"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
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
	siguiente               LectorRelacionSeleccionadaRPT
	identidad               ResolutorIdentidadLectorRelacionRPT
	limiteIdentidad         time.Duration
	emisorTecnico           vecports.EmisorResultadosTecnicosConContexto
	configuracionResultados ConfiguracionResultadosTecnicosLectorRPT
}

// ConfiguracionResultadosTecnicosLectorRPT sólo admite componente y etapa
// del catálogo común. No incorpora identidad, recursos ni errores libres.
type ConfiguracionResultadosTecnicosLectorRPT struct {
	Componente vecdomain.ComponenteIncidenciaTecnica
	Etapa      vecdomain.EtapaIncidenciaTecnica
}

func (c ConfiguracionResultadosTecnicosLectorRPT) validar() error {
	_, err := vecdomain.ClasificarResultadoTecnico(vecdomain.SolicitudResultadoTecnico{Resultado: vecdomain.ResultadoTecnicoCorrecto, Componente: c.Componente, Etapa: c.Etapa})
	return err
}

// NuevoLectorRelacionSeleccionadaRPTConIdentidad enlaza la selección existente
// con la frontera nominal y el contexto técnico común, también fuera de HTTP.
func NuevoLectorRelacionSeleccionadaRPTConIdentidad(l LectorRelacionSeleccionadaRPT, i ResolutorIdentidadLectorRelacionRPT, limite time.Duration, e vecports.EmisorResultadosTecnicosConContexto, c ConfiguracionResultadosTecnicosLectorRPT) (LectorRelacionSeleccionadaRPT, error) {
	if dependenciaNula(l) || dependenciaNula(i) || dependenciaNula(e) || c.validar() != nil || limite <= 0 || limite > 30*time.Second {
		return nil, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return lectorRelacionSeleccionadaCorrelacion{siguiente: l, identidad: i, limiteIdentidad: limite, emisorTecnico: e, configuracionResultados: c}, nil
}

func (l lectorRelacionSeleccionadaCorrelacion) ConsultarSeleccionada(ctx context.Context, actor vecdomain.ContextoActor, preparacion personaldomain.PreparacionRelacionParaRPT, organismo, seleccion string) (resultado personalports.ResultadoRelacionParaRPTV1, errResultado error) {
	var vacio personalports.ResultadoRelacionParaRPTV1
	errResultado = personaldomain.ErrLectorRelacionRPTNoDisponible
	// Se emite una vez al conocer la respuesta final, también si el selector
	// rechaza antes del servicio. La cola común no cambia el recibo de negocio.
	defer func() {
		if !dependenciaNula(l.emisorTecnico) {
			l.emisorTecnico.EmitirResultadoConContexto(ctx, vecdomain.SolicitudResultadoTecnico{Resultado: codigoResultadoTecnicoLectorRPT(errResultado), Componente: l.configuracionResultados.Componente, Etapa: l.configuracionResultados.Etapa})
		}
	}()
	if ctx == nil || dependenciaNula(l.siguiente) || dependenciaNula(l.identidad) || dependenciaNula(l.emisorTecnico) || l.configuracionResultados.validar() != nil || l.limiteIdentidad <= 0 || l.limiteIdentidad > 30*time.Second {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	if _, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx); err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	// La identidad histórica se captura antes de validar o abrir la lectura.
	// Una cancelación posterior no sustituye persona, perfil ni vínculo originales.
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(l.limiteIdentidad))
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

func codigoResultadoTecnicoLectorRPT(err error) vecdomain.CodigoResultadoTecnico {
	switch {
	case err == nil:
		return vecdomain.ResultadoTecnicoCorrecto
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return vecdomain.ResultadoTecnicoCancelado
	case errors.Is(err, personaldomain.ErrLectorRelacionRPTDenegado):
		return vecdomain.ResultadoTecnicoDenegado
	case errors.Is(err, personaldomain.ErrLectorRelacionRPTInvalido):
		return vecdomain.ResultadoTecnicoEntradaInvalida
	default:
		return vecdomain.ResultadoTecnicoNoDisponible
	}
}

var _ LectorRelacionSeleccionadaRPT = lectorRelacionSeleccionadaCorrelacion{}
