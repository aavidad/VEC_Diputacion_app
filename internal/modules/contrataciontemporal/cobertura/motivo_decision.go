package cobertura

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	atributoClaveI18nMotivoDecisionCobertura = "clave_i18n"
	redaccionResolucionMotivoCobertura       = "[RESOLUCION-MOTIVO-DECISION-COBERTURA-REDACTADA]"
)

var (
	ErrConfiguracionResolutorMotivoDecisionCobertura = errors.New(
		"contratacion temporal: configuracion del motivo de cobertura invalida",
	)
	ErrMotivoDecisionCoberturaNoConfiable = errors.New(
		"contratacion temporal: motivo de cobertura no confiable",
	)
)

// ResolucionMotivoDecisionCobertura conserva el motivo funcional C2, no el
// motivo de autorización VEC. Solo contiene coordenadas nominales copiadas:
// no es capacidad, autorización, atestación ni prueba durable. O4-04 debe
// repetir la consulta dentro de su transacción.
type ResolucionMotivoDecisionCobertura struct {
	motivo     domain.MotivoGobernadoDecisionCobertura
	resueltaEn time.Time
}

func (r ResolucionMotivoDecisionCobertura) Motivo() (
	domain.MotivoGobernadoDecisionCobertura,
	error,
) {
	if !motivoDecisionCoberturaNominalValido(r.motivo) ||
		!instanteResolucionMotivoCoberturaValido(r.resueltaEn) {
		return domain.MotivoGobernadoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	return r.motivo, nil
}

func (r ResolucionMotivoDecisionCobertura) ResueltaEn() (time.Time, error) {
	if _, err := r.Motivo(); err != nil {
		return time.Time{}, err
	}
	return r.resueltaEn, nil
}

func (ResolucionMotivoDecisionCobertura) String() string {
	return redaccionResolucionMotivoCobertura
}

func (r ResolucionMotivoDecisionCobertura) GoString() string {
	return r.String()
}

func (r ResolucionMotivoDecisionCobertura) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ResolucionMotivoDecisionCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r ResolucionMotivoDecisionCobertura) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r ResolucionMotivoDecisionCobertura) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

// ResolutorMotivoDecisionCobertura fija el catálogo permitido desde
// composición. La referencia sellada selecciona una versión y entrada, pero
// nunca sustituye la lectura de la publicación VEC.
type ResolutorMotivoDecisionCobertura struct {
	consulta   puertosvec.ConsultaCatalogosConfigurables
	catalogoID string
}

func NuevoResolutorMotivoDecisionCobertura(
	consulta puertosvec.ConsultaCatalogosConfigurables,
	catalogoID string,
) (*ResolutorMotivoDecisionCobertura, error) {
	centinela := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: catalogoID, CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_cobertura_valido",
	}
	if dependenciaResolucionMotivoNula(consulta) ||
		centinela.Validar() != nil {
		return nil, ErrConfiguracionResolutorMotivoDecisionCobertura
	}
	return &ResolutorMotivoDecisionCobertura{
		consulta: consulta, catalogoID: catalogoID,
	}, nil
}

func (r *ResolutorMotivoDecisionCobertura) Resolver(
	ctx context.Context,
	motivo domain.MotivoGobernadoDecisionCobertura,
	instante time.Time,
) (ResolucionMotivoDecisionCobertura, error) {
	if r == nil || dependenciaResolucionMotivoNula(r.consulta) ||
		dependenciaResolucionMotivoNula(ctx) || r.catalogoID == "" ||
		motivo.ReferenciaCatalogo.CatalogoID != r.catalogoID ||
		!motivoDecisionCoberturaNominalValido(motivo) ||
		!instanteResolucionMotivoCoberturaValido(instante) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	catalogoVivo, err := r.consulta.ObtenerCatalogo(
		ctx,
		motivo.ReferenciaCatalogo.CatalogoID,
		motivo.ReferenciaCatalogo.CatalogoVersion,
	)
	if err != nil || ctx.Err() != nil ||
		catalogoVivo.Validar() != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	// La consulta debe entregar una instantánea, pero el resolutor vuelve a
	// clonar colecciones y mapas antes de calcular o conservar cualquier dato.
	catalogo, err := catalogoVivo.ClonarCanonico()
	if err != nil || ctx.Err() != nil ||
		catalogo.ID != motivo.ReferenciaCatalogo.CatalogoID ||
		catalogo.Version != motivo.ReferenciaCatalogo.CatalogoVersion ||
		catalogo.Estado != dominiovec.EstadoCatalogoPublicado ||
		catalogo.PublicadoEn.After(instante) {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil || !huellasMotivoCoberturaIguales(
		huella,
		motivo.ReferenciaCatalogo.CatalogoHuellaSHA256,
	) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	entrada, err := catalogo.ObtenerEntradaVigente(
		motivo.ReferenciaCatalogo.EntradaClave,
		instante,
	)
	if err != nil || ctx.Err() != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	claveI18n, existe := entrada.Atributos[atributoClaveI18nMotivoDecisionCobertura]
	if !existe || claveI18n != string(motivo.ClaveI18n) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	return ResolucionMotivoDecisionCobertura{
		motivo: motivo, resueltaEn: instante,
	}, nil
}

func motivoDecisionCoberturaNominalValido(
	motivo domain.MotivoGobernadoDecisionCobertura,
) bool {
	return motivo.ReferenciaCatalogo.Validar() == nil &&
		motivo.ClaveI18n.Valida()
}

func instanteResolucionMotivoCoberturaValido(instante time.Time) bool {
	return domain.InstanteUTCCanonico(instante) &&
		instante.Year() >= 1 && instante.Year() <= 9999
}

func huellasMotivoCoberturaIguales(primera string, segunda string) bool {
	return len(primera) == 64 && len(segunda) == 64 &&
		subtle.ConstantTimeCompare([]byte(primera), []byte(segunda)) == 1
}

func dependenciaResolucionMotivoNula(dependencia any) bool {
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

func errorResolucionMotivoCobertura(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrMotivoDecisionCoberturaNoConfiable, err)
		}
	}
	return ErrMotivoDecisionCoberturaNoConfiable
}
