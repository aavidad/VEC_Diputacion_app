package composicion

import (
	"context"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type claveIntentoFichaPropia struct{}
type intentoFichaPropia struct {
	identidad  IdentidadRegistradaFichaPropia
	referencia string
	mu         sync.Mutex
	orden      *vecports.OrdenIntentoAuditoria
	acuse      *vecports.AcuseIntentoAuditoria
}

// PrepararContextoIntentoFichaPropia captura la identidad registrada antes de
// validar la petición o abrir la consulta. La correlación ya debe proceder de
// la frontera común. Reutilizar el contexto no vuelve a resolver el perfil.
func PrepararContextoIntentoFichaPropia(ctx context.Context, resolver ResolutorIdentidadFichaPropia, limite time.Duration) (context.Context, error) {
	if ctx == nil || dependenciaNula(resolver) || limite <= 0 || limite > 30*time.Second {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	if _, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx); err != nil {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	if _, err := estadoIntentoFichaPropia(ctx); err == nil {
		return ctx, nil
	}
	auditCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(limite))
	defer cancelar()
	identidad, err := resolver.ResolverIdentidadFichaPropia(auditCtx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	copia, err := identidad.Resultado.Clonar()
	if err != nil {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	identidad.Resultado = copia
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	return context.WithValue(ctx, claveIntentoFichaPropia{}, &intentoFichaPropia{identidad: identidad, referencia: ref}), nil
}

// IdentidadOriginalFichaPropia entrega una copia de la captura, sin consultar
// otra fuente de persona, empleado o perfil durante la operación.
func IdentidadOriginalFichaPropia(ctx context.Context) (IdentidadRegistradaFichaPropia, error) {
	intento, err := estadoIntentoFichaPropia(ctx)
	if err != nil {
		return IdentidadRegistradaFichaPropia{}, err
	}
	copia, err := intento.identidad.Resultado.Clonar()
	if err != nil {
		return IdentidadRegistradaFichaPropia{}, domain.ErrFichaPropiaNoDisponible
	}
	return IdentidadRegistradaFichaPropia{Vinculo: intento.identidad.Vinculo, Resultado: copia}, nil
}
func identidadOriginalFichaPropia(ctx context.Context) (IdentidadRegistradaFichaPropia, error) {
	return IdentidadOriginalFichaPropia(ctx)
}
func estadoIntentoFichaPropia(ctx context.Context) (*intentoFichaPropia, error) {
	if ctx != nil {
		if intento, ok := ctx.Value(claveIntentoFichaPropia{}).(*intentoFichaPropia); ok && intento != nil {
			return intento, nil
		}
	}
	return nil, domain.ErrFichaPropiaNoDisponible
}

type ConsultorFichaPropia interface {
	Consultar(context.Context, domain.SolicitudFichaPropia) (ports.ResultadoFichaPropia, error)
}
type consultorFichaPropiaConIdentidad struct {
	siguiente ConsultorFichaPropia
	identidad ResolutorIdentidadFichaPropia
	limite    time.Duration
}

// NuevoConsultorFichaPropiaConIdentidad sirve también canales sin HTTP. El
// servidor aporta identidad y correlación; el DTO no puede reemplazarlas.
func NuevoConsultorFichaPropiaConIdentidad(c ConsultorFichaPropia, i ResolutorIdentidadFichaPropia, limite time.Duration) (ConsultorFichaPropia, error) {
	if dependenciaNula(c) || dependenciaNula(i) || limite <= 0 || limite > 30*time.Second {
		return nil, domain.ErrFichaPropiaNoDisponible
	}
	return consultorFichaPropiaConIdentidad{c, i, limite}, nil
}
func (c consultorFichaPropiaConIdentidad) Consultar(ctx context.Context, in domain.SolicitudFichaPropia) (ports.ResultadoFichaPropia, error) {
	capturado, err := PrepararContextoIntentoFichaPropia(ctx, c.identidad, c.limite)
	if err != nil || dependenciaNula(c.siguiente) {
		return ports.ResultadoFichaPropia{}, domain.ErrFichaPropiaNoDisponible
	}
	return c.siguiente.Consultar(capturado, in)
}
