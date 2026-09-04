package cobertura

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sort"
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

// ConsultaMotivoDecisionCoberturaAcotada evita reconstruir un catálogo
// completo cuando la fuente durable ya ofrece la entrada vigente exacta.
// La implementación debe resolver únicamente desde una publicación
// autoritativa para el instante solicitado.
type ConsultaMotivoDecisionCoberturaAcotada interface {
	ConsultarMotivoDecisionCobertura(
		context.Context,
		string,
		string,
		domain.ClaveCatalogo,
		time.Time,
	) (domain.MotivoGobernadoDecisionCobertura, error)
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
	consulta        puertosvec.ConsultaCatalogosConfigurablesAcotada
	consultaAcotada ConsultaMotivoDecisionCoberturaAcotada
	catalogoID      string
	moduloID        string
}

func NuevoResolutorMotivoDecisionCobertura(
	consulta puertosvec.ConsultaCatalogosConfigurablesAcotada,
	catalogoID string,
	moduloID string,
) (*ResolutorMotivoDecisionCobertura, error) {
	if dependenciaResolucionMotivoNula(consulta) ||
		!configuracionResolutorMotivoDecisionCoberturaValida(
			catalogoID,
			moduloID,
		) {
		return nil, ErrConfiguracionResolutorMotivoDecisionCobertura
	}
	return &ResolutorMotivoDecisionCobertura{
		consulta: consulta, catalogoID: catalogoID, moduloID: moduloID,
	}, nil
}

// NuevoResolutorMotivoDecisionCoberturaAcotado conserva la validación de
// dominio pero permite que PostgreSQL entregue solo la entrada gobernada
// necesaria para la decisión.
func NuevoResolutorMotivoDecisionCoberturaAcotado(
	consulta ConsultaMotivoDecisionCoberturaAcotada,
	catalogoID string,
	moduloID string,
) (*ResolutorMotivoDecisionCobertura, error) {
	if dependenciaResolucionMotivoNula(consulta) ||
		!configuracionResolutorMotivoDecisionCoberturaValida(
			catalogoID,
			moduloID,
		) {
		return nil, ErrConfiguracionResolutorMotivoDecisionCobertura
	}
	return &ResolutorMotivoDecisionCobertura{
		consultaAcotada: consulta,
		catalogoID:      catalogoID,
		moduloID:        moduloID,
	}, nil
}

// ResolverClave selecciona el motivo funcional a partir de la clave elegida
// por el cliente. Catálogo, módulo, versión, huella y clave i18n proceden
// exclusivamente de la publicación gobernada resuelta en el servidor.
func (r *ResolutorMotivoDecisionCobertura) ResolverClave(
	ctx context.Context,
	clave domain.ClaveCatalogo,
	instante time.Time,
) (ResolucionMotivoDecisionCobertura, error) {
	if !r.configuracionValida() ||
		dependenciaResolucionMotivoNula(ctx) || !clave.Valida() ||
		!instanteResolucionMotivoCoberturaValido(instante) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	if r.consultaAcotada != nil {
		motivo, err := r.consultaAcotada.ConsultarMotivoDecisionCobertura(
			ctx,
			r.catalogoID,
			r.moduloID,
			clave,
			instante,
		)
		if err != nil || ctx.Err() != nil {
			return ResolucionMotivoDecisionCobertura{},
				errorResolucionMotivoCobertura(ctx)
		}
		if motivo.ReferenciaCatalogo.CatalogoID != r.catalogoID ||
			motivo.ReferenciaCatalogo.EntradaClave != string(clave) ||
			!motivoDecisionCoberturaNominalValido(motivo) {
			return ResolucionMotivoDecisionCobertura{},
				ErrMotivoDecisionCoberturaNoConfiable
		}
		return ResolucionMotivoDecisionCobertura{
			motivo: motivo, resueltaEn: instante,
		}, nil
	}
	limites := limitesConsultaMotivosDecisionCobertura()
	resultado, err := r.consulta.ListarVersionesCatalogoAcotado(
		ctx,
		r.catalogoID,
		limites,
	)
	if err != nil || ctx.Err() != nil ||
		resultado.Truncado ||
		len(resultado.Catalogos) == 0 ||
		len(resultado.Catalogos) > limites.Versiones {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	versionesVivas := resultado.Catalogos
	versiones := make([]dominiovec.CatalogoConfigurable, len(versionesVivas))
	var consumo puertosvec.ConsumoConsultaCatalogosAcotada
	for indice := range versionesVivas {
		medida, medible := puertosvec.MedirCatalogoConfigurable(
			versionesVivas[indice],
		)
		siguiente, cabe := consumo.Agregar(medida, limites)
		if ctx.Err() != nil || !medible || !cabe {
			return ResolucionMotivoDecisionCobertura{},
				errorResolucionMotivoCobertura(ctx)
		}
		versiones[indice], err = versionesVivas[indice].ClonarCanonico()
		if err != nil ||
			versiones[indice].ID != r.catalogoID ||
			versiones[indice].ModuloID != r.moduloID {
			return ResolucionMotivoDecisionCobertura{},
				ErrMotivoDecisionCoberturaNoConfiable
		}
		consumo = siguiente
	}
	sort.Slice(versiones, func(primera, segunda int) bool {
		return versiones[primera].Version < versiones[segunda].Version
	})
	if !historialMotivosDecisionCoberturaValido(versiones) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	catalogo, encontrado := seleccionarCatalogoMotivoDecisionCobertura(
		versiones,
		instante,
	)
	if !encontrado ||
		(catalogo.Estado == dominiovec.EstadoCatalogoRetirado &&
			!catalogo.RetiradoEn.After(instante)) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	entrada, encontrada := entradaMotivoDecisionCoberturaVigente(
		catalogo,
		string(clave),
		instante,
	)
	if !encontrada {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	claveI18n, existe := entrada.Atributos[atributoClaveI18nMotivoDecisionCobertura]
	motivo := domain.MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:      catalogo.ID,
			CatalogoVersion: catalogo.Version,
			EntradaClave:    entrada.Clave,
		},
		ClaveI18n: domain.ClaveCatalogo(claveI18n),
	}
	motivo.ReferenciaCatalogo.CatalogoHuellaSHA256, err =
		catalogo.HuellaSHA256()
	if !existe || err != nil || !motivo.ClaveI18n.Valida() ||
		!motivoDecisionCoberturaNominalValido(motivo) {
		return ResolucionMotivoDecisionCobertura{},
			ErrMotivoDecisionCoberturaNoConfiable
	}
	// La segunda lectura exacta evita aceptar una referencia derivada de un
	// listado que haya cambiado antes de utilizarla.
	return r.Resolver(ctx, motivo, instante)
}

func (r *ResolutorMotivoDecisionCobertura) Resolver(
	ctx context.Context,
	motivo domain.MotivoGobernadoDecisionCobertura,
	instante time.Time,
) (ResolucionMotivoDecisionCobertura, error) {
	if !r.configuracionValida() ||
		dependenciaResolucionMotivoNula(ctx) ||
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
	if r.consultaAcotada != nil {
		resuelto, err := r.consultaAcotada.ConsultarMotivoDecisionCobertura(
			ctx,
			r.catalogoID,
			r.moduloID,
			domain.ClaveCatalogo(motivo.ReferenciaCatalogo.EntradaClave),
			instante,
		)
		if err != nil || ctx.Err() != nil {
			return ResolucionMotivoDecisionCobertura{},
				errorResolucionMotivoCobertura(ctx)
		}
		if resuelto != motivo {
			return ResolucionMotivoDecisionCobertura{},
				ErrMotivoDecisionCoberturaNoConfiable
		}
		return ResolucionMotivoDecisionCobertura{
			motivo: motivo, resueltaEn: instante,
		}, nil
	}
	resultado, err := r.consulta.ObtenerCatalogoAcotado(
		ctx,
		motivo.ReferenciaCatalogo.CatalogoID,
		motivo.ReferenciaCatalogo.CatalogoVersion,
		limitesConsultaMotivosDecisionCobertura(),
	)
	catalogoVivo := resultado.Catalogo
	if err != nil || ctx.Err() != nil || resultado.Truncado ||
		!catalogoMotivoDecisionCoberturaAcotado(catalogoVivo) {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	// La consulta debe entregar una instantánea, pero el resolutor vuelve a
	// clonar colecciones y mapas antes de calcular o conservar cualquier dato.
	catalogo, err := catalogoVivo.ClonarCanonico()
	if err != nil || ctx.Err() != nil ||
		catalogo.ID != motivo.ReferenciaCatalogo.CatalogoID ||
		catalogo.Version != motivo.ReferenciaCatalogo.CatalogoVersion ||
		catalogo.ModuloID != r.moduloID ||
		!catalogoMotivoDecisionCoberturaPublicadoEn(catalogo, instante) {
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
	entrada, encontrada := entradaMotivoDecisionCoberturaVigente(
		catalogo,
		motivo.ReferenciaCatalogo.EntradaClave,
		instante,
	)
	if !encontrada || ctx.Err() != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	claveI18n, existe := entrada.Atributos[atributoClaveI18nMotivoDecisionCobertura]
	if !existe || claveI18n != string(motivo.ClaveI18n) ||
		ctx.Err() != nil {
		return ResolucionMotivoDecisionCobertura{},
			errorResolucionMotivoCobertura(ctx)
	}
	return ResolucionMotivoDecisionCobertura{
		motivo: motivo, resueltaEn: instante,
	}, nil
}

func (r *ResolutorMotivoDecisionCobertura) configuracionValida() bool {
	if r == nil ||
		!configuracionResolutorMotivoDecisionCoberturaValida(
			r.catalogoID,
			r.moduloID,
		) {
		return false
	}
	consultaCompleta := !dependenciaResolucionMotivoNula(r.consulta)
	consultaAcotada := !dependenciaResolucionMotivoNula(r.consultaAcotada)
	return consultaCompleta != consultaAcotada
}

func configuracionResolutorMotivoDecisionCoberturaValida(
	catalogoID string,
	moduloID string,
) bool {
	centinela := dominiovec.CatalogoConfigurable{
		ID:             catalogoID,
		Version:        1,
		Revision:       1,
		ModuloID:       moduloID,
		Nombre:         "Configuración de motivos",
		FuenteRef:      "configuracion_motivos_cobertura",
		MotivoCreacion: "Validación de composición.",
		Estado:         dominiovec.EstadoCatalogoBorrador,
		CreadoPor:      "composicion_aplicacion",
		CreadoEn:       time.Unix(0, 0).UTC(),
	}
	return centinela.Validar() == nil
}

func catalogoMotivoDecisionCoberturaAcotado(
	catalogo dominiovec.CatalogoConfigurable,
) bool {
	medida, medible := puertosvec.MedirCatalogoConfigurable(catalogo)
	_, cabe := (puertosvec.ConsumoConsultaCatalogosAcotada{}).Agregar(
		medida,
		limitesConsultaMotivosDecisionCobertura(),
	)
	return medible && cabe
}

func limitesConsultaMotivosDecisionCobertura() (
	limites puertosvec.LimitesConsultaCatalogosAcotada,
) {
	// El catálogo de motivos es pequeño. Estos topes reducen la superficie de
	// agotamiento y admiten hasta 64 revisiones o miles de opciones dentro del
	// presupuesto agregado.
	return puertosvec.LimitesConsultaCatalogosAcotada{
		Versiones:        64,
		Entradas:         4_096,
		Atributos:        8_192,
		BytesAproximados: 4 << 20,
	}
}

func historialMotivosDecisionCoberturaValido(
	versiones []dominiovec.CatalogoConfigurable,
) bool {
	for indice := range versiones {
		actual := versiones[indice]
		if actual.Version != indice+1 {
			return false
		}
		if indice == 0 {
			continue
		}
		anterior := versiones[indice-1]
		if actual.VersionAnteriorRef != anterior.Referencia() ||
			actual.CreadoEn.Before(anterior.CreadoEn) ||
			anterior.Estado == dominiovec.EstadoCatalogoBorrador ||
			actual.CreadoEn.Before(anterior.PublicadoEn) ||
			(actual.Estado != dominiovec.EstadoCatalogoBorrador &&
				actual.PublicadoEn.Before(anterior.PublicadoEn)) {
			return false
		}
	}
	return true
}

func seleccionarCatalogoMotivoDecisionCobertura(
	versiones []dominiovec.CatalogoConfigurable,
	instante time.Time,
) (dominiovec.CatalogoConfigurable, bool) {
	for indice := len(versiones) - 1; indice >= 0; indice-- {
		catalogo := versiones[indice]
		if catalogo.Estado != dominiovec.EstadoCatalogoBorrador &&
			!catalogo.PublicadoEn.After(instante) {
			return catalogo, true
		}
	}
	return dominiovec.CatalogoConfigurable{}, false
}

func catalogoMotivoDecisionCoberturaPublicadoEn(
	catalogo dominiovec.CatalogoConfigurable,
	instante time.Time,
) bool {
	if catalogo.PublicadoEn.After(instante) {
		return false
	}
	switch catalogo.Estado {
	case dominiovec.EstadoCatalogoPublicado:
		return true
	case dominiovec.EstadoCatalogoRetirado:
		return catalogo.RetiradoEn.After(instante)
	default:
		return false
	}
}

func entradaMotivoDecisionCoberturaVigente(
	catalogo dominiovec.CatalogoConfigurable,
	clave string,
	instante time.Time,
) (dominiovec.EntradaCatalogoConfigurable, bool) {
	if !catalogoMotivoDecisionCoberturaPublicadoEn(catalogo, instante) ||
		clave != strings.TrimSpace(clave) {
		return dominiovec.EntradaCatalogoConfigurable{}, false
	}
	for _, entrada := range catalogo.Entradas {
		if entrada.Clave == clave && entrada.VigenteEn(instante) {
			return entrada, true
		}
	}
	return dominiovec.EntradaCatalogoConfigurable{}, false
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
