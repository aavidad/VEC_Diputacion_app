package seguridad

import (
	"context"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// AutenticadorFuentesAnalisisConConfianza implementa la frontera criptográfica
// en infraestructura. La aplicación solo conoce el contrato de ports.
type AutenticadorFuentesAnalisisConConfianza struct {
	confianza ports.ConfianzaAutoridadesFuenteAnalisis
}

func NuevoAutenticadorFuentesAnalisisConConfianza(
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
) (*AutenticadorFuentesAnalisisConConfianza, error) {
	if !domain.ReferenciaOpacaValida(confianza.OrganizacionRef()) ||
		!domain.ReferenciaOpacaValida(confianza.Audiencia()) {
		return nil, ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return &AutenticadorFuentesAnalisisConConfianza{
		confianza: confianza,
	}, nil
}

func (a *AutenticadorFuentesAnalisisConConfianza) OrganizacionAutoridadFuenteAnalisis() string {
	if a == nil {
		return ""
	}
	return a.confianza.OrganizacionRef()
}

func (a *AutenticadorFuentesAnalisisConConfianza) AutenticarAutoridadFuenteAnalisis(
	ctx context.Context,
	presentador ports.PresentadorAutoridadFuenteAnalisis,
	materialPeticion []byte,
	rol ports.RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (ports.IdentidadAutoridadFuenteAnalisis, error) {
	if a == nil || ctx == nil || dependenciaAdaptadorNula(presentador) {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{}, err
	}
	desafio, err := ports.NuevoDesafioAutoridadFuenteAnalisis(
		append([]byte(nil), materialPeticion...),
		a.confianza.OrganizacionRef(),
		a.confianza.Audiencia(),
		rol,
	)
	if err != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	presentacion, errPresentacion :=
		presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{}, errContexto
	}
	if errPresentacion != nil {
		return ports.IdentidadAutoridadFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return a.confianza.VerificarPresentacion(
		presentacion,
		desafio,
		rol,
		comprobadaEn,
	)
}

func dependenciaAdaptadorNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

var _ ports.AutenticadorAutoridadesFuenteAnalisis = (*AutenticadorFuentesAnalisisConConfianza)(nil)
